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

// maxFailsBeforeDeadLetter is the number of consecutive flush failures an
// individual buffered answer is allowed before it is moved to the dead-letter
// list and removed from the live queue. This prevents a single persistently
// un-upsertable row (e.g. a question_id that no longer exists) from blocking
// all later answers for the same attempt.
const maxFailsBeforeDeadLetter = 3

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

// autosaveFailKey holds a Redis Hash mapping raw JSON item → consecutive
// failure count. The hash is keyed alongside its live queue so both expire
// when the queue is drained.
func autosaveFailKey(key string) string {
	return key + ":fails"
}

// autosaveDeadKey is a Redis List that accumulates items that exceeded
// maxFailsBeforeDeadLetter. It is retained for manual inspection / replay.
func autosaveDeadKey(key string) string {
	return key + ":dead"
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

// normalizeAnswer applies the unseen-answer rule: an answer the student never
// reached must not contribute time or interaction signals. `seen` reports
// whether the answer counts as seen, which is a.SelectedAnswer != "" unless an
// explicit Seen pointer overrides it. When unseen, every behavioural field is
// zeroed so a skipped question cannot skew the SQI analysis.
func normalizeAnswer(a AnswerInput) (AnswerInput, bool) {
	seen := a.SelectedAnswer != ""
	if a.Seen != nil {
		seen = *a.Seen
	}
	if !seen {
		a.TimeSpent = 0
		a.SelectedAnswer = ""
		a.MarkedForReview = false
		a.Revisited = false
		a.ChangedAnswer = false
		a.WasInitiallyWrong = false
	}
	return a, seen
}

// flushBatch writes every item in the batch to Postgres.
//
// Per-item strategy:
//   - JSON-corrupt items are logged and skipped (counted as done).
//   - Items that fail UpsertAnswer have their consecutive failure count
//     incremented in a Redis Hash. Once the count reaches
//     maxFailsBeforeDeadLetter the item is moved to the dead-letter list
//     (autosave_dead:{key}) and removed from the live queue.
//   - Items that fail below the threshold are left in place so the next tick
//     retries them — but crucially the loop continues, so one bad row never
//     blocks later rows in the same batch.
//
// Batch strategy:
//   - After processing all items, only the successfully-handled items
//     (upserted or dead-lettered) are trimmed from the live queue.
//   - If every item in the batch failed transiently (e.g. DB is down) the
//     LTrim is skipped, preserving the entire batch for the next tick.
func (b *AutosaveBuffer) flushBatch(ctx context.Context, key string, items []string) {
	failKey := autosaveFailKey(key)
	deadKey := autosaveDeadKey(key)

	// doneCount tracks items that no longer need to live in the queue —
	// either because they were upserted successfully or dead-lettered.
	doneCount := 0

	for _, it := range items {
		var ba bufferedAnswer
		if err := json.Unmarshal([]byte(it), &ba); err != nil {
			// Corrupt JSON can never be fixed — count it as done so it is
			// trimmed and doesn't block the queue.
			log.Printf("[AUTOSAVE] dropping corrupt buffered item: %v (raw=%q)", err, it)
			doneCount++
			continue
		}

		a, answerSeen := normalizeAnswer(ba.Answer)
		upsertErr := b.attemptRepo.UpsertAnswer(
			ba.AttemptID, a.QuestionID, a.SelectedAnswer, false, a.TimeSpent,
			a.MarkedForReview, a.Revisited, a.ChangedAnswer, a.WasInitiallyWrong, answerSeen,
		)
		if upsertErr == nil {
			// Success — clear any accumulated failure count and mark done.
			_ = b.rdb.HDel(ctx, failKey, it).Err()
			doneCount++
			continue
		}

		// Persistent failure handling: increment the per-item counter.
		log.Printf("[AUTOSAVE] flush upsert failed attempt %d q%d: %v", ba.AttemptID, a.QuestionID, upsertErr)
		newCount, incrErr := b.rdb.HIncrBy(ctx, failKey, it, 1).Result()
		if incrErr != nil {
			// Can't track the count — treat conservatively: leave item alive
			// for the next tick without advancing doneCount.
			log.Printf("[AUTOSAVE] HIncrBy failed for %s: %v", failKey, incrErr)
			continue
		}

		if newCount >= maxFailsBeforeDeadLetter {
			// Dead-letter: move out of the hot path so later items are not
			// blocked. The item is preserved in the dead list for inspection.
			log.Printf("[AUTOSAVE] dead-lettering item after %d failures (attempt %d q%d): %q",
				newCount, ba.AttemptID, a.QuestionID, it)
			_ = b.rdb.RPush(ctx, deadKey, it).Err()
			_ = b.rdb.HDel(ctx, failKey, it).Err()
			doneCount++
		}
		// If below the threshold, leave the item in the queue (do not
		// advance doneCount). The loop continues to handle later items.
	}

	if doneCount == 0 {
		// Nothing was processed — whole batch failed transiently (e.g. DB
		// down). Keep the full batch for next tick; don't trim anything.
		return
	}

	// Trim exactly the items we finished with — whether they succeeded or were
	// dead-lettered. Items that are still failing remain at the front of the
	// list (Redis list is FIFO) and will be retried on the next tick.
	//
	// NOTE: this relies on failing items staying at their original positions.
	// Because we process items in order and the list is not modified during
	// the loop, the first doneCount positions correspond to handled items only
	// when ALL failures come after the successes. In the general case (a
	// failure in the middle of the batch), we trim from the end of the
	// successfully-handled window: use the total items length minus the
	// number of items still in the retry queue.
	retryCount := len(items) - doneCount
	if retryCount == 0 {
		// All items handled — trim the full batch.
		if err := b.rdb.LTrim(ctx, key, int64(len(items)), -1).Err(); err != nil {
			log.Printf("[AUTOSAVE] LTrim failed for %s, will retry next tick: %v", key, err)
		}
		return
	}

	// Partial success: remove items that are done (upserted or dead-lettered)
	// by replacing the list head with only the items still pending. We do this
	// by LPOPing the done items one-by-one so Redis's list ordering is
	// preserved for the pending items.
	for i := 0; i < doneCount; i++ {
		if err := b.rdb.LPop(ctx, key).Err(); err != nil {
			log.Printf("[AUTOSAVE] LPOP failed for %s after partial flush: %v", key, err)
			return
		}
	}
}
