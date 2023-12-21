package worker

import (
	"context"
	"time"

	"golang.org/x/sync/semaphore"
)

type SemPool struct {
	sem *semaphore.Weighted
}

func NewSemPool(concurrency int) *SemPool {
	return &SemPool{
		sem: semaphore.NewWeighted(int64(concurrency)),
	}
}

func (p *SemPool) Dispatch(ctx context.Context, task func()) error {
	timeoutCtx, timeout := context.WithTimeout(ctx, 10*time.Second)
	defer timeout()

	h := func() {
		defer p.sem.Release(1)
		task()
	}

	if err := p.sem.Acquire(timeoutCtx, 1); err != nil {
		return err
	}

	go h()
	return nil
}
