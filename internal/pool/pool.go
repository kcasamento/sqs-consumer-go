package pool

import (
	"log"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type Pool struct {
	// tasks are submitted here
	// and moved to worker queue
	taskQueue chan func()

	// queue that workers are consuming
	workerQueue chan func()
	waiting     *atomic.Int32

	stoppedChan  chan struct{}
	waitingQueue WaitingRoom
	stopOnce     sync.Once
	stopped      bool
	wait         bool
	maxWorkers   int
	idleTimeout  time.Duration
}

func NewPool(opts ...PoolOpt) *Pool {
	pool := &Pool{
		maxWorkers:   1,
		taskQueue:    make(chan func()),
		workerQueue:  make(chan func()),
		stoppedChan:  make(chan struct{}),
		idleTimeout:  math.MaxInt64,
		waiting:      &atomic.Int32{},
		waitingQueue: NewSliceWaitingRoom(),
	}

	for _, opt := range opts {
		opt(pool)
	}

	// Start the task dispatcher.
	go pool.dispatch()

	return pool
}

func (p *Pool) Submit(task func()) {
	log.Printf("submitting task %v\n", task)
	if task != nil {
		p.taskQueue <- task
	}
}

func (p *Pool) Stop(wait bool) {
	p.stop(wait)
}

func (p *Pool) WaitingTasks() int {
	return int(p.waiting.Load())
}

func (p *Pool) dispatch() {
	defer close(p.stoppedChan)

	timeout := time.NewTimer(p.idleTimeout)
	var workerCount int
	var idle bool
	var wg sync.WaitGroup

Loop:
	for {
		// check the waiting room
		if p.waitingQueue.Size() != 0 {
			log.Printf("waiting queue size: %d\n", p.waitingQueue.Size())
			if !p.processWaitingQueue() {
				break Loop
			}

			continue
		}

		select {
		case task, ok := <-p.taskQueue:
			if !ok {
				break Loop
			}

			log.Printf("task received: %v\n", task)

			// new task
			select {
			case p.workerQueue <- task:
			default:
				if workerCount < p.maxWorkers {
					wg.Add(1)
					go worker(task, p.workerQueue, &wg)
				} else {
					// TODO: queue in waiting
					p.waitingQueue.Queue(task)
					p.waiting.Store(int32(p.waitingQueue.Size()))
				}
			}
			idle = false
		case <-timeout.C:
			if idle && workerCount > 0 {
				// TODO:
				p.killIdleWorker()
				workerCount--
			}
			idle = true
			timeout.Reset(p.idleTimeout)
		}
	}

	if p.wait {
		p.runQueuedTasks()
	}

	for workerCount > 0 {
		// the workers are setup to
		// shutdown when they get a nil task
		// so this will ensure that all active
		// workers receive a nil value
		p.workerQueue <- nil
		workerCount--
	}

	// the workers will indicate then they have
	// exited, so here we wait for all of them
	// to have received the nil task and exit
	wg.Wait()
	timeout.Stop()
}

func worker(task func(), workerQueue chan func(), wg *sync.WaitGroup) {
	// this worker will process the task it was given
	// after, it will stay alive to see if there is more work
	// it can process
	// Once it receives a nil task it will shutdown
	for task != nil {
		log.Printf("worker received task %v\n", task)
		task()
		task = <-workerQueue
	}

	wg.Done()
}

func (p *Pool) stop(wait bool) {
	p.stopOnce.Do(func() {
		// dispatcher will use
		// wait to gauge whether it should
		// run queued tasks or just exit
		p.wait = wait

		// signal to dispatcher to start
		// shutting down workers
		close(p.taskQueue)
	})

	// wait for dispatcher to cleanup
	<-p.stoppedChan
}

func (p *Pool) processWaitingQueue() bool {
	select {
	case task, ok := <-p.taskQueue:
		if !ok {
			return false
		}

		p.waitingQueue.Queue(task)
	case p.workerQueue <- p.waitingQueue.Pop():
	}
	p.waiting.Store(int32(p.waitingQueue.Size()))
	return true
}

func (p *Pool) killIdleWorker() bool {
	select {
	case p.workerQueue <- nil:
		// at least worker was idle
		return true
	default:
		// all workers are busy
		// so we will not kill them
		return false
	}
}

func (p *Pool) runQueuedTasks() {
	for p.waitingQueue.Size() != 0 {
		p.workerQueue <- p.waitingQueue.Pop()
		p.waiting.Store(int32(p.waitingQueue.Size()))
	}
}
