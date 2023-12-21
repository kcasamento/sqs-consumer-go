package worker

import (
	"context"

	"github.com/kcasamento/sqs-consumer-go/internal/pool"
)

type WorkerPool struct {
	wp pool.Pool
}

func NewWorker(concurrency int) *WorkerPool {
	return &WorkerPool{
		wp: *pool.NewPool(
			pool.WithMaxWorkers(concurrency),
		),
	}
}

func (p *WorkerPool) Dispatch(ctx context.Context, task func()) error {
	p.wp.Submit(task)

	return nil
}
