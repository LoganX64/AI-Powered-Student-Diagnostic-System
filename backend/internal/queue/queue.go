package queue

type Handler func(jobID int)

type Queue struct {
	ch chan int
}

func New(bufferSize int) *Queue {
	if bufferSize <= 0 {
		bufferSize = 1024
	}
	return &Queue{ch: make(chan int, bufferSize)}
}

func (q *Queue) Enqueue(jobID int) {
	q.ch <- jobID
}

// Start launches the worker goroutine that processes enqueued job IDs.
func (q *Queue) Start(handler Handler) {
	go func() {
		for jobID := range q.ch {
			handler(jobID)
		}
	}()
}

func (q *Queue) Stop() {
	close(q.ch)
}
