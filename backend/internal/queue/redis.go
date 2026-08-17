package queue

import (
	"context"
	"encoding/json"
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

func (q *inProcessQueue) Start(computeHandler func(int, int), finalizeHandler func(FinalizePayload)) {
	go func() {
		for pl := range q.computeCh {
			computeHandler(pl.JobID, pl.TenantID)
		}
	}()
	go func() {
		for p := range q.finalizeCh {
			finalizeHandler(p)
		}
	}()
}

func (q *inProcessQueue) Stop() {}

// redisQueue is the scale-mode queue backed by a Redis Stream. A single
// consumer group drains both task types; in-flight messages are ACKed after
// successful handling so a crash replays them (at-least-once).
type redisQueue struct {
	client *redis.Client
	cancel context.CancelFunc
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
		return
	}
	_ = q.client.XAdd(context.Background(), &redis.XAddArgs{
		Stream: StreamKey,
		Values: map[string]interface{}{"type": TaskComputeSQI, "payload": payload},
	})
}

func (q *redisQueue) EnqueueFinalize(p FinalizePayload) {
	payload, err := marshalFinalize(p)
	if err != nil {
		return
	}
	_ = q.client.XAdd(context.Background(), &redis.XAddArgs{
		Stream: StreamKey,
		Values: map[string]interface{}{"type": TaskFinalize, "payload": payload},
	})
}

func (q *redisQueue) Start(computeHandler func(int, int), finalizeHandler func(FinalizePayload)) {
	ctx, cancel := context.WithCancel(context.Background())
	q.cancel = cancel

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
				Consumer: "worker-1",
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
					q.dispatch(msg, computeHandler, finalizeHandler)
					_ = q.client.XAck(ctx, StreamKey, ConsumerGroup, msg.ID).Err()
				}
			}
		}
	}()
}

func (q *redisQueue) dispatch(msg redis.XMessage, computeHandler func(int, int), finalizeHandler func(FinalizePayload)) {
	msgType, _ := msg.Values["type"].(string)
	payload, _ := msg.Values["payload"].(string)
	switch msgType {
	case TaskComputeSQI:
		var pl ComputePayload
		if err := json.Unmarshal([]byte(payload), &pl); err == nil {
			computeHandler(pl.JobID, pl.TenantID)
		}
	case TaskFinalize:
		var pl FinalizePayload
		if err := json.Unmarshal([]byte(payload), &pl); err == nil {
			finalizeHandler(pl)
		}
	}
}

func (q *redisQueue) Stop() {
	if q.cancel != nil {
		q.cancel()
	}
}
