package pool

import (
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
	if task != nil {
		// only submit non-nil tasks
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
		// check if any tasks were queued up in memory
		// and handle start processing those before
		// we take on any "new" work
		// if new tasks come in during this period they will get
		// get added to the end of the queue
		if p.waitingQueue.Size() != 0 {
			if !p.processWaitingQueue() {
				break Loop
			}

			continue
		}

		// We've cleared the queued work and are ready
		// to handle new incoming work

		select {
		case task, ok := <-p.taskQueue:
			if !ok {
				break Loop
			}

			// we have a new task.  conceptually this task can only be handled
			// by a worker (goroutine).  the only way to send work to a worker is through the
			// workerQueue channel.  Keep in mind, the workQueueChannel is unbuffered and will block
			// after the first task is put in it.  this sounds like it will block very quickly, but when you
			// have many workers running at the same time this channel can be drained relatively quickly and
			// the more workers you have the faster it will drain.  Any work that would cause the pool to exceed
			// its quota gets help in memory in the "waiting room" and will be flushed from time to time as workers
			// free up

			// so how this works:
			// first, try to add work to the workerQueue
			// if the workerQueue if full up, see if we can spin up another worker and keep doing this until
			// we exceed the quota for workers.  at this point, we start queuing up work in the waiting room
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
			// While all that is happening, this will keep track
			// of how active our pool is.  if after a defined amount
			// of time no new work has come in, we can start spinning down
			// workers to free up resources
			// this happens in 2 phases
			// first time we timeout we set the idle flag but do not
			// shutdown any workers
			// if we timeout again and we still have not gotten any new
			// work we shut down 1 work every timeout interval until
			// all workers are shutdown or new work comes in and resets
			// the idle state
			if idle && workerCount > 0 {
				// TODO:
				p.killIdleWorker()
				workerCount--
			}
			idle = true
			timeout.Reset(p.idleTimeout)
		}
	}

	// at this point we have broken out of the
	// main event loop and will proceed to shutdown the pool

	// if the wait flag is set, we will drain the waiting room queue
	// before exiting otherwise we exit and those tasks will be lost
	if p.wait {
		p.runQueuedTasks()
	}

	// shutdown/release the workers and their resources
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

	// stop the timeout so we don't keep
	// consuming resources running a timer
	timeout.Stop()
}

func worker(task func(), workerQueue chan func(), wg *sync.WaitGroup) {
	// this worker will process the task it was given
	// after, it will stay alive to see if there is more work
	// it can process
	// Once it receives a nil task it will shutdown
	for task != nil {
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
	// since the only way to run work is by signaling the
	// workerQueue, here we keep popping off the next task
	// in the waiting room queue and sending it to the workerQueue
	// if new work comes in during this time, it will get added to the end
	// of the waiting room queue
	select {
	case task, ok := <-p.taskQueue:
		if !ok {
			// something has gone wrong,
			// so we return false here to signal
			// to the caller that things are unstable
			// and it should not continue listening for
			// new work
			return false
		}

		p.waitingQueue.Queue(task)
	case p.workerQueue <- p.waitingQueue.Pop():
	}

	// at this point we have either queued a new task
	// or processed a task.  in either case we need to
	// update (atomically) the waiting room queue size
	p.waiting.Store(int32(p.waitingQueue.Size()))

	// we return true, to indicate
	// it is safe to confinue listening for new work.
	return true
}

func (p *Pool) killIdleWorker() bool {
	// keeping in mind that the workerQueue is unbuffered
	// if we can't immediatly send a nil signal that means
	// a worker is actively doing something at which point we return
	// false to indicate nothing was killed
	// if we can send the nil signal that means all workers are idle
	// and the first work to pick up the nil signal will shutdown
	// and we return true to indicate we successfully killed a worker
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
