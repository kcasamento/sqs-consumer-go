package pool

import "time"

type PoolOpt = func(*Pool)

func WithMaxWorkers(maxWorkers int) PoolOpt {
	return func(p *Pool) {
		if maxWorkers > 0 {
			p.maxWorkers = maxWorkers
		}
	}
}

func WithIdleTimeout(timeout time.Duration) PoolOpt {
	return func(p *Pool) {
		p.idleTimeout = timeout
	}
}

func WithWaitingRoom(wr WaitingRoom) PoolOpt {
	return func(p *Pool) {
		if wr != nil {
			p.waitingQueue = wr
		}
	}
}
