package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"ai-student-diagnostic/backend/internal/config"
	"github.com/redis/go-redis/v9"
)

// inProcessQueue is the standard-mode queue: a buffered channel per type.
type inProcessQueue struct {
	computeCh  chan ComputePayload
	finalizeCh chan FinalizePayload
}

func (q *inProcessQueue) EnqueueCompute(jobID, tenantID int) {
	q.computeCh <- ComputePayload{JobID: jobID, TenantID: tenantID}
}

func (q *inProcessQueue) EnqueueFinalize(p FinalizePayload) {
	q.finalizeCh <- p
}

func (q *inProcessQueue) Start(computeHandler func(int, int) error, finalizeHandler func(FinalizePayload) error) {
	go func() {
		for pl := range q.computeCh {
			_ = computeHandler(pl.JobID, pl.TenantID)
		}
	}()
	go func() {
		for p := range q.finalizeCh {
			_ = finalizeHandler(p)
		}
	}()
}

func (q *inProcessQueue) Stop() {}

// redisQueue is the scale-mode queue backed by a Redis Stream. A single
// consumer group drains both task types; in-flight messages are ACKed only
// after the handler succeeds, so a failure (or crash) replays them
// (at-least-once). Each API instance uses a unique consumer name so work is
// actually distributed, and a periodic XAUTOCLAIM loop reclaims messages left
// pending by crashed instances.
type redisQueue struct {
	client   *redis.Client
	cancel   context.CancelFunc
	consumer string
}

func newRedisClient(cfg *config.Config) *redis.Client {
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil
	}
	return redis.NewClient(opts)
}

func (q *redisQueue) EnqueueCompute(jobID, tenantID int) {
	payload, err := marshalCompute(jobID, tenantID)
	if err != nil {
		log.Printf("[QUEUE] enqueue compute marshal failed (job %d): %v", jobID, err)
		return
	}
	if err := q.client.XAdd(context.Background(), &redis.XAddArgs{
		Stream: StreamKey,
		Values: map[string]interface{}{"type": TaskComputeSQI, "payload": payload},
	}).Err(); err != nil {
		log.Printf("[QUEUE] enqueue compute failed (job %d): %v", jobID, err)
	}
}

func (q *redisQueue) EnqueueFinalize(p FinalizePayload) {
	payload, err := marshalFinalize(p)
	if err != nil {
		log.Printf("[QUEUE] enqueue finalize marshal failed (attempt %d): %v", p.AttemptID, err)
		return
	}
	if err := q.client.XAdd(context.Background(), &redis.XAddArgs{
		Stream: StreamKey,
		Values: map[string]interface{}{"type": TaskFinalize, "payload": payload},
	}).Err(); err != nil {
		log.Printf("[QUEUE] enqueue finalize failed (attempt %d): %v", p.AttemptID, err)
	}
}

func (q *redisQueue) Start(computeHandler func(int, int) error, finalizeHandler func(FinalizePayload) error) {
	ctx, cancel := context.WithCancel(context.Background())
	q.cancel = cancel

	// Unique consumer per instance so multiple API instances share the group
	// and distribute pending entries instead of colliding as one "worker-1".
	host, _ := os.Hostname()
	q.consumer = fmt.Sprintf("worker-%s-%d", host, os.Getpid())

	// Ensure the stream + consumer group exist.
	_ = q.client.XGroupCreateMkStream(ctx, StreamKey, ConsumerGroup, "$").Err()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			res, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    ConsumerGroup,
				Consumer: q.consumer,
				Streams:  []string{StreamKey, ">"},
				Count:    10,
				Block:    2 * time.Second,
			}).Result()
			if err != nil {
				if err == redis.Nil || ctx.Err() != nil {
					continue
				}
				time.Sleep(200 * time.Millisecond)
				continue
			}

			for _, stream := range res {
				for _, msg := range stream.Messages {
					if err := q.dispatch(msg, computeHandler, finalizeHandler); err != nil {
						// Leave unacknowledged so the message replays (at-least-once).
						log.Printf("[QUEUE] handler failed for message %s, will retry: %v", msg.ID, err)
						continue
					}
					if err := q.client.XAck(ctx, StreamKey, ConsumerGroup, msg.ID).Err(); err != nil {
						log.Printf("[QUEUE] XACK failed for message %s: %v", msg.ID, err)
					}
				}
			}
		}
	}()

	// Reclaim messages left pending by crashed/stopped instances.
	go func() {
		recoverTicker := time.NewTicker(30 * time.Second)
		defer recoverTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-recoverTicker.C:
				q.recoverPending(ctx, computeHandler, finalizeHandler)
			}
		}
	}()
}

// recoverPending claims entries that have been idle longer than the threshold
// (e.g. owned by a crashed instance) and redelivers them to this consumer.
func (q *redisQueue) recoverPending(ctx context.Context, computeHandler func(int, int) error, finalizeHandler func(FinalizePayload) error) {
	const minIdle = 60 * time.Second
	for {
		msgs, _, err := q.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream:   StreamKey,
			Group:    ConsumerGroup,
			Consumer: q.consumer,
			MinIdle:  minIdle,
			Start:    "0",
			Count:    10,
		}).Result()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[QUEUE] XAUTOCLAIM failed: %v", err)
			return
		}
		if len(msgs) == 0 {
			return
		}
		for _, msg := range msgs {
			if err := q.dispatch(msg, computeHandler, finalizeHandler); err != nil {
				log.Printf("[QUEUE] recovered handler failed for message %s, will retry: %v", msg.ID, err)
				continue
			}
			if err := q.client.XAck(ctx, StreamKey, ConsumerGroup, msg.ID).Err(); err != nil {
				log.Printf("[QUEUE] XACK failed for recovered message %s: %v", msg.ID, err)
			}
		}
	}
}

func (q *redisQueue) dispatch(msg redis.XMessage, computeHandler func(int, int) error, finalizeHandler func(FinalizePayload) error) error {
	msgType, _ := msg.Values["type"].(string)
	payload, _ := msg.Values["payload"].(string)
	switch msgType {
	case TaskComputeSQI:
		var pl ComputePayload
		if err := json.Unmarshal([]byte(payload), &pl); err != nil {
			return err
		}
		return computeHandler(pl.JobID, pl.TenantID)
	case TaskFinalize:
		var pl FinalizePayload
		if err := json.Unmarshal([]byte(payload), &pl); err != nil {
			return err
		}
		return finalizeHandler(pl)
	}
	return nil
}

func (q *redisQueue) Stop() {
	if q.cancel != nil {
		q.cancel()
	}
}
