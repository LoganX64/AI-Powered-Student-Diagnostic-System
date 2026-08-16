package services

import (
	"context"
	"encoding/json"
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
	once         sync.Once
}

func NewAutosaveBuffer(rdb *redis.Client, attemptRepo *repository.AttemptRepo) *AutosaveBuffer {
	return &AutosaveBuffer{
		rdb:         rdb,
		attemptRepo: attemptRepo,
		flushEvery:  1 * time.Second,
		maxBatch:    200,
		shutdown:    make(chan struct{}),
	}
}

type bufferedAnswer struct {
	AttemptID int           `json:"attempt_id"`
	Answer    AnswerInput   `json:"answer"`
}

func autosaveKey(attemptID int) string {
	return "autosave:" + strconv.Itoa(attemptID)
}

// Push records an answer into the Redis buffer.
func (b *AutosaveBuffer) Push(attemptID int, answers []AnswerInput) {
	if b.rdb == nil {
		return
	}
	ctx := context.Background()
	pipe := b.rdb.Pipeline()
	for _, a := range answers {
		ba, _ := json.Marshal(bufferedAnswer{AttemptID: attemptID, Answer: a})
		pipe.RPush(ctx, autosaveKey(attemptID), string(ba))
	}
	_, _ = pipe.Exec(ctx)
}

// Start launches the background flush loop.
func (b *AutosaveBuffer) Start() {
	go func() {
		ticker := time.NewTicker(b.flushEvery)
		defer ticker.Stop()
		for {
			select {
			case <-b.shutdown:
				b.flushAll()
				return
			case <-ticker.C:
				b.flushAll()
			}
		}
	}()
}

// Stop signals the flush loop to drain and exit.
func (b *AutosaveBuffer) Stop() {
	b.once.Do(func() { close(b.shutdown) })
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
		if err != nil || len(items) == 0 {
			continue
		}
		b.flushBatch(ctx, key, items)
	}
}

func (b *AutosaveBuffer) flushBatch(ctx context.Context, key string, items []string) {
	for _, it := range items {
		var ba bufferedAnswer
		if err := json.Unmarshal([]byte(it), &ba); err != nil {
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
	_ = b.rdb.LTrim(ctx, key, int64(len(items)), -1)
}
