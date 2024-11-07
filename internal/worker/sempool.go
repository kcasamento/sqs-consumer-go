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
	// need to atomically read the stop flag since many goroutines
	// can dispatch in parallel
	if p.stop.Load() {
		// if the stop flag is set, something tried to dispatch
		// work to a shutdown pool which is not allowed
		return fmt.Errorf("pool is stopped")
	}

	// wrap the unit of work such that
	// we release the lock when done and
	// decrement the wait group counter
	h := func() {
		defer p.sem.Release(1)
		defer p.wg.Done()

		// the unit of work that was dispatched
		task()
	}

	for {
		// timeout if aquiring a lock takes too long
		timeoutCtx, timeout := context.WithTimeout(ctx, p.aquireTimeout)
		// try to aquire a lock
		if err := p.sem.Acquire(timeoutCtx, 1); err != nil {
			/* return err */
			continue
		}

		timeout()
		break
	}

	// we got a lock, so increment the wait group
	// to indicate work is pending
	p.wg.Add(1)

	// Note: Because we are using a weighted semaphore,
	// we are limiting the amount of go routines
	// that can run at any given time.  With that said,
	// there could be some inefficiencies here since every
	// dispatch will create a new goroutine with resources
	// that may need to be GC'd.
	// This is where a worker pool has the advantage of keeping
	// "workers" alive so that new goroutines don't have to be spun up/down
	// on every dispatch but rather can be kept idle until they have work to
	// do.

	// run the unit of work in the background
	go h()
	return nil
}

func (p *SemPool) Stop(ctx context.Context) error {
	// need to atomically set the stop flag
	// since many goroutines may be reading from it
	p.stop.Store(true)

	// Note: Here we wait for all the running
	// tasks to finish.
	// We will do this by setting up a wait channel
	// and using the wait group counter.  Once all tasks
	// have finished, the wait group will stop waiting and
	// the wait channel is closed indicating there are no tasks
	// running
	waitCh := make(chan struct{})
	go func() {
		defer close(waitCh)
		p.wg.Wait()
	}()

	// So here we wait for the context to get cancled or the
	// wait channel to close
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-waitCh:
		return nil
	}
}
