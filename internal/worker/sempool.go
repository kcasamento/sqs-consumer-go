package worker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/semaphore"
)

type SemPool struct {
	stop          *atomic.Bool
	wg            *sync.WaitGroup
	sem           *semaphore.Weighted
	aquireTimeout time.Duration
}

func NewSemPool(
	concurrency int,
	aquireTimeout time.Duration,
) *SemPool {
	return &SemPool{
		wg:            &sync.WaitGroup{},
		sem:           semaphore.NewWeighted(int64(concurrency)),
		aquireTimeout: aquireTimeout,
		stop:          &atomic.Bool{},
	}
}

func (p *SemPool) Dispatch(ctx context.Context, task func()) error {
	if p.stop.Load() {
		return fmt.Errorf("pool is stopped")
	}

	timeoutCtx, timeout := context.WithTimeout(ctx, p.aquireTimeout)
	defer timeout()

	h := func() {
		defer p.sem.Release(1)
		defer p.wg.Done()
		task()
	}

	if err := p.sem.Acquire(timeoutCtx, 1); err != nil {
		return err
	}

	p.wg.Add(1)
	go h()
	return nil
}

func (p *SemPool) Stop(ctx context.Context) error {
	p.stop.Store(true)
	waitCh := make(chan struct{})
	go func() {
		defer close(waitCh)
		p.wg.Wait()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-waitCh:
		return nil
	}
}
