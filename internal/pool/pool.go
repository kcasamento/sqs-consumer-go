package pool

import "sync"

type Pool struct {
	taskQueue   chan func()
	workerQueue chan func()
	stoppedChan chan struct{}
	stopSignal  chan struct{}
	// waitingQueue deque.Deque[func()]
	stopLock   sync.Mutex
	stopOnce   sync.Once
	stopped    bool
	waiting    int32
	wait       bool
	maxWorkers int
}

func NewPool(
	maxWorkers int,
) *Pool {
	if maxWorkers < 1 {
		maxWorkers = 1
	}

	pool := &Pool{
		maxWorkers:  maxWorkers,
		taskQueue:   make(chan func()),
		workerQueue: make(chan func()),
		stopSignal:  make(chan struct{}),
		stoppedChan: make(chan struct{}),
	}

	// Start the task dispatcher.
	go pool.dispatch()

	return pool
}

func (p *Pool) Submit(task func()) {}

func (p *Pool) dispatch() {}
