package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ai-student-diagnostic/backend/internal/config"
	"github.com/redis/go-redis/v9"
)

// maxRetries caps how many times a failing message is replayed before it is
// treated as poison and dropped (acked) so it cannot loop forever.
const maxRetries = 5

// inProcessQueue is the standard-mode queue: a buffered channel per type.
type inProcessQueue struct {
	computeCh  chan ComputePayload
	finalizeCh chan FinalizePayload
	wg         sync.WaitGroup
	stopOnce   sync.Once
	closed     atomic.Bool
}

func (q *inProcessQueue) EnqueueCompute(jobID, tenantID int) error {
	if q.closed.Load() {
		return fmt.Errorf("queue stopped")
	}
	q.computeCh <- ComputePayload{JobID: jobID, TenantID: tenantID}
	return nil
}

func (q *inProcessQueue) EnqueueFinalize(p FinalizePayload) error {
	if q.closed.Load() {
		return fmt.Errorf("queue stopped")
	}
	q.finalizeCh <- p
	return nil
}

func (q *inProcessQueue) Start(computeHandler func(int, int) error, finalizeHandler func(FinalizePayload) error) {
	q.wg.Add(2)
	go func() {
		defer q.wg.Done()
		for pl := range q.computeCh {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[QUEUE] compute handler panic (job %d): %v", pl.JobID, r)
					}
				}()
				if err := computeHandler(pl.JobID, pl.TenantID); err != nil {
					log.Printf("[QUEUE] compute handler failed (job %d, tenant %d): %v", pl.JobID, pl.TenantID, err)
				}
			}()
		}
	}()
	go func() {
		defer q.wg.Done()
		for p := range q.finalizeCh {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[QUEUE] finalize handler panic (attempt %d): %v", p.AttemptID, r)
					}
				}()
				if err := finalizeHandler(p); err != nil {
					log.Printf("[QUEUE] finalize handler failed (attempt %d, student %d): %v", p.AttemptID, p.StudentID, err)
				}
			}()
		}
	}()
}

// Stop closes the channels so the consumer goroutines drain any buffered jobs
// and exit, then waits for them to finish — no goroutine leak and no dropped
// in-flight jobs on shutdown. Enqueue must not be called after Stop.
func (q *inProcessQueue) Stop() {
	q.stopOnce.Do(func() {
		q.closed.Store(true)
		close(q.computeCh)
		close(q.finalizeCh)
	})
	q.wg.Wait()
}

// redisQueue is the scale-mode queue backed by a Redis Stream. A single
// consumer group drains both task types; in-flight messages are ACKed only
// after the handler succeeds, so a failure (or crash) replays them
// (at-least-once). Each API instance uses a unique consumer name so work is
// actually distributed, and a periodic XAUTOCLAIM loop reclaims messages left
// pending by crashed instances.
type redisQueue struct {
	client    *redis.Client
	cancel    context.CancelFunc
	consumer  string
	failCount sync.Map
}

func newRedisClient(cfg *config.Config) *redis.Client {
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil
	}
	return redis.NewClient(opts)
}

func (q *redisQueue) EnqueueCompute(jobID, tenantID int) error {
	payload, err := marshalCompute(jobID, tenantID)
	if err != nil {
		log.Printf("[QUEUE] enqueue compute marshal failed (job %d): %v", jobID, err)
		return err
	}
	if err := q.client.XAdd(context.Background(), &redis.XAddArgs{
		Stream: StreamKey,
		Values: map[string]interface{}{"type": TaskComputeSQI, "payload": payload},
	}).Err(); err != nil {
		log.Printf("[QUEUE] enqueue compute failed (job %d): %v", jobID, err)
		return err
	}
	return nil
}

func (q *redisQueue) EnqueueFinalize(p FinalizePayload) error {
	payload, err := marshalFinalize(p)
	if err != nil {
		log.Printf("[QUEUE] enqueue finalize marshal failed (attempt %d): %v", p.AttemptID, err)
		return err
	}
	if err := q.client.XAdd(context.Background(), &redis.XAddArgs{
		Stream: StreamKey,
		Values: map[string]interface{}{"type": TaskFinalize, "payload": payload},
	}).Err(); err != nil {
		log.Printf("[QUEUE] enqueue finalize failed (attempt %d): %v", p.AttemptID, err)
		return err
	}
	return nil
}

func (q *redisQueue) Start(computeHandler func(int, int) error, finalizeHandler func(FinalizePayload) error) {
	ctx, cancel := context.WithCancel(context.Background())
	q.cancel = cancel

	// Unique consumer per instance so multiple API instances share the group
	// and distribute pending entries instead of colliding as one "worker-1".
	host, err := os.Hostname()
	if err != nil {
		log.Printf("[QUEUE] hostname lookup failed, using fallback: %v", err)
		host = "unknown"
	}
	q.consumer = fmt.Sprintf("worker-%s-%d", host, os.Getpid())

	// Ensure the stream + consumer group exist.
	if err := q.client.XGroupCreateMkStream(ctx, StreamKey, ConsumerGroup, "$").Err(); err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		log.Printf("[QUEUE] consumer group create failed: %v", err)
	}

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
				log.Printf("[QUEUE] XReadGroup error: %v", err)
				if strings.Contains(err.Error(), "NOGROUP") {
					// Stream/group missing — recreate and keep reading (self-heal).
					if crerr := q.client.XGroupCreateMkStream(ctx, StreamKey, ConsumerGroup, "$").Err(); crerr != nil && !strings.Contains(crerr.Error(), "BUSYGROUP") {
						log.Printf("[QUEUE] recreate consumer group failed: %v", crerr)
					}
				}
				time.Sleep(200 * time.Millisecond)
				continue
			}

			for _, stream := range res {
				for _, msg := range stream.Messages {
					func() {
						defer func() {
							if r := recover(); r != nil {
								log.Printf("[QUEUE] panic processing message %s: %v", msg.ID, r)
							}
						}()
						if err := q.dispatch(msg, computeHandler, finalizeHandler); err != nil {
							// Leave unacknowledged so the message replays, unless it
							// has exhausted its retries and is treated as poison.
							if q.recordFailure(msg.ID) {
								log.Printf("[QUEUE] poison message %s dropped after %d retries: %v", msg.ID, maxRetries, err)
								q.failCount.Delete(msg.ID)
								q.ack(ctx, msg.ID)
								return
							}
							log.Printf("[QUEUE] handler failed for message %s, will retry: %v", msg.ID, err)
							return
						}
						q.failCount.Delete(msg.ID)
						q.ack(ctx, msg.ID)
					}()
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
	cursor := "0-0"
	for {
		msgs, next, err := q.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream:   StreamKey,
			Group:    ConsumerGroup,
			Consumer: q.consumer,
			MinIdle:  minIdle,
			Start:    cursor,
			Count:    10,
		}).Result()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[QUEUE] XAUTOCLAIM failed: %v", err)
			return
		}
		for _, msg := range msgs {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[QUEUE] panic processing recovered message %s: %v", msg.ID, r)
					}
				}()
				if err := q.dispatch(msg, computeHandler, finalizeHandler); err != nil {
					if q.recordFailure(msg.ID) {
						log.Printf("[QUEUE] poison recovered message %s dropped after %d retries: %v", msg.ID, maxRetries, err)
						q.failCount.Delete(msg.ID)
						q.ack(ctx, msg.ID)
						return
					}
					log.Printf("[QUEUE] recovered handler failed for message %s, will retry: %v", msg.ID, err)
					return
				}
				q.failCount.Delete(msg.ID)
				q.ack(ctx, msg.ID)
			}()
		}
		if next == "0-0" {
			return
		}
		cursor = next
	}
}

// ack acknowledges a message with a bounded retry so a transient XACK failure
// does not force a full replay of an already-completed task.
func (q *redisQueue) ack(ctx context.Context, id string) {
	for i := 0; i < 3; i++ {
		if err := q.client.XAck(ctx, StreamKey, ConsumerGroup, id).Err(); err == nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	log.Printf("[QUEUE] XACK ultimately failed for message %s", id)
}

// recordFailure counts replays for a message and reports true once it should be
// treated as poison (dropped) to avoid an infinite replay loop.
func (q *redisQueue) recordFailure(id string) bool {
	n, _ := q.failCount.LoadOrStore(id, 0)
	cnt := n.(int) + 1
	q.failCount.Store(id, cnt)
	return cnt >= maxRetries
}

func (q *redisQueue) dispatch(msg redis.XMessage, computeHandler func(int, int) error, finalizeHandler func(FinalizePayload) error) error {
	msgType, ok := msg.Values["type"].(string)
	if !ok {
		return fmt.Errorf("message %s: missing or invalid type", msg.ID)
	}
	payload, ok := msg.Values["payload"].(string)
	if !ok {
		return fmt.Errorf("message %s: missing or invalid payload", msg.ID)
	}
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
	return fmt.Errorf("unknown task type %q for message %s", msgType, msg.ID)
}

func (q *redisQueue) Stop() {
	if q.cancel != nil {
		q.cancel()
	}
}
