package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"ai-student-diagnostic/backend/internal/repository"
	"github.com/redis/go-redis/v9"
)

// AutosaveBuffer absorbs high-frequency autosave writes in Redis and flushes
// them to Postgres in batches, decoupling request latency from DB write rate
// (absorbs the 3k–6k writes/sec surge at scale).
type AutosaveBuffer struct {
	rdb          *redis.Client
	attemptRepo  *repository.AttemptRepo
	flushEvery   time.Duration
	maxBatch     int
	shutdown     chan struct{}
	done         chan struct{}
	once         sync.Once
}

func NewAutosaveBuffer(rdb *redis.Client, attemptRepo *repository.AttemptRepo) *AutosaveBuffer {
	return &AutosaveBuffer{
		rdb:         rdb,
		attemptRepo: attemptRepo,
		flushEvery:  1 * time.Second,
		maxBatch:    200,
		shutdown:    make(chan struct{}),
		done:        make(chan struct{}),
	}
}

type bufferedAnswer struct {
	AttemptID int           `json:"attempt_id"`
	Answer    AnswerInput   `json:"answer"`
}

func autosaveKey(attemptID int) string {
	return "autosave:" + strconv.Itoa(attemptID)
}

// Push records answers into the Redis buffer. It returns an error if the
// buffer cannot be written so the caller can signal the client — answers are
// never silently dropped under a Redis hiccup.
func (b *AutosaveBuffer) Push(attemptID int, answers []AnswerInput) error {
	if b.rdb == nil {
		return nil
	}
	ctx := context.Background()
	pipe := b.rdb.Pipeline()
	for _, a := range answers {
		ba, err := json.Marshal(bufferedAnswer{AttemptID: attemptID, Answer: a})
		if err != nil {
			return fmt.Errorf("autosave: marshal answer for attempt %d: %w", attemptID, err)
		}
		pipe.RPush(ctx, autosaveKey(attemptID), string(ba))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("autosave: failed to buffer answers for attempt %d: %w", attemptID, err)
	}
	return nil
}

// FlushAttempt synchronously drains and persists any buffered answers for a
// single attempt. Used by the sweeper before finalizing an expired attempt so
// no late answers are lost because the background flush had not yet run.
func (b *AutosaveBuffer) FlushAttempt(attemptID int) error {
	if b.rdb == nil {
		return nil
	}
	ctx := context.Background()
	key := autosaveKey(attemptID)
	const maxAttempts = 5
	for attempt := 0; attempt < maxAttempts; attempt++ {
		items, err := b.rdb.LRange(ctx, key, 0, int64(b.maxBatch-1)).Result()
		if err != nil {
			return fmt.Errorf("autosave: read attempt %d failed: %w", attemptID, err)
		}
		if len(items) == 0 {
			return nil
		}
		b.flushBatch(ctx, key, items)
		remaining, err := b.rdb.LLen(ctx, key).Result()
		if err != nil {
			return fmt.Errorf("autosave: len attempt %d failed: %w", attemptID, err)
		}
		if remaining == 0 {
			return nil
		}
		// flushBatch kept items due to a write error; retry.
	}
	return fmt.Errorf("autosave: failed to flush attempt %d after %d attempts", attemptID, maxAttempts)
}

// Start launches the background flush loop. It first performs a synchronous
// drain of any residual buffered answers left in Redis by a previous run, so
// no data is lost across restarts before new traffic is accepted.
func (b *AutosaveBuffer) Start() {
	b.flushAll()
	go func() {
		defer close(b.done)
		ticker := time.NewTicker(b.flushEvery)
		defer ticker.Stop()
		for {
			select {
			case <-b.shutdown:
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("[AUTOSAVE] flush panic on shutdown: %v", r)
						}
					}()
					b.flushAll()
				}()
				return
			case <-ticker.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("[AUTOSAVE] flush panic: %v", r)
						}
					}()
					b.flushAll()
				}()
			}
		}
	}()
}

// Stop signals the flush loop to drain and blocks until the final flush to
// Postgres has completed, guaranteeing no buffered answers are lost on
// shutdown.
func (b *AutosaveBuffer) Stop() {
	b.once.Do(func() { close(b.shutdown) })
	<-b.done
}

func (b *AutosaveBuffer) flushAll() {
	if b.rdb == nil {
		return
	}
	ctx := context.Background()
	// Scan attempt keys; cap work per tick to avoid long pauses.
	iter := b.rdb.Scan(ctx, 0, "autosave:*", 50).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		items, err := b.rdb.LRange(ctx, key, 0, int64(b.maxBatch-1)).Result()
		if err != nil {
			log.Printf("[AUTOSAVE] LRange failed for %s: %v", key, err)
			continue
		}
		if len(items) == 0 {
			continue
		}
		b.flushBatch(ctx, key, items)
	}
	if err := iter.Err(); err != nil {
		log.Printf("[AUTOSAVE] scan failed, possible unflushed keys: %v", err)
	}
}

func (b *AutosaveBuffer) flushBatch(ctx context.Context, key string, items []string) {
	for _, it := range items {
		var ba bufferedAnswer
		if err := json.Unmarshal([]byte(it), &ba); err != nil {
			log.Printf("[AUTOSAVE] dropping corrupt buffered item: %v (raw=%q)", err, it)
			continue
		}
		a := ba.Answer
		answerSeen := a.SelectedAnswer != ""
		if a.Seen != nil {
			answerSeen = *a.Seen
		}
		if !answerSeen {
			a.TimeSpent = 0
			a.SelectedAnswer = ""
			a.MarkedForReview = false
			a.Revisited = false
			a.ChangedAnswer = false
			a.WasInitiallyWrong = false
		}
		if err := b.attemptRepo.UpsertAnswer(
			ba.AttemptID, a.QuestionID, a.SelectedAnswer, false, a.TimeSpent,
			a.MarkedForReview, a.Revisited, a.ChangedAnswer, a.WasInitiallyWrong, answerSeen,
		); err != nil {
			log.Printf("[AUTOSAVE] flush upsert failed attempt %d q%d: %v", ba.AttemptID, a.QuestionID, err)
			return // keep items for retry on next tick
		}
	}
	if err := b.rdb.LTrim(ctx, key, int64(len(items)), -1).Err(); err != nil {
		log.Printf("[AUTOSAVE] LTrim failed for %s, will retry next tick: %v", key, err)
		return // keep items for retry
	}
}
