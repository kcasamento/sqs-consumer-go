package harvester

import (
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

type BatchHarvester[T any] struct {
	clock         *time.Ticker
	in            chan T
	inClosed      *atomic.Bool
	stopped       chan struct{}
	onBatch       func([]T)
	buffer        []T
	flushInterval time.Duration
	l             sync.Mutex
	ptr           int
	maxBatchSize  int
}

func NewBatchHarvester[T any](
	onBatch func([]T),
	maxBatchSize int,
	flushInterval time.Duration,
) Harvester[T] {
	h := &BatchHarvester[T]{
		in:            make(chan T, 1024),
		inClosed:      &atomic.Bool{},
		onBatch:       onBatch,
		maxBatchSize:  maxBatchSize,
		flushInterval: flushInterval,
		ptr:           0,
		buffer:        make([]T, maxBatchSize),
		clock:         time.NewTicker(flushInterval),
		stopped:       make(chan struct{}),
	}

	go h.start()
	return h
}

func (h *BatchHarvester[T]) Add(item T) {
	if closed := h.inClosed.Load(); closed {
		return
	}

	// add the item to be harvested to the in channel
	h.in <- item

	// since the clock is stopped after a flush,
	// we need to start it back up once we get a new item
	h.clock.Reset(h.flushInterval)
}

func (h *BatchHarvester[T]) Stop() {
	// signal the in channel is closed
	// and close the channel
	h.inClosed.Store(true)
	close(h.in)

	// stop the clock to free up resouces
	h.clock.Stop()

	// wait for current buffer to be flushed
	// and then exit
	<-h.stopped
}

func (h *BatchHarvester[T]) Flush() {
	// try and flush buffer
	// if data is empty, ignore
	// else notify the callback

	data := h.flushData()
	if len(data) == 0 {
		return
	}

	h.onBatch(data)

	// stop the clock since at this point
	// we know there is nothing left in the buffer
	// to be harvested
	// the clock will get restarted when a new item comes
	// in
	h.clock.Stop()
}

func (h *BatchHarvester[T]) append(item T) {
	// add the item to the next open slot
	// and increment the pointer
	h.l.Lock()
	h.buffer[h.ptr] = item
	h.ptr++
	h.l.Unlock()

	// If the ptr is equal to the max
	// size, flush the data and reset
	// the buffer
	if h.ptr >= h.maxBatchSize {
		h.Flush()
	}
}

func (h *BatchHarvester[T]) flushData() []T {
	h.l.Lock()
	defer h.l.Unlock()

	// TODO: make a deep copy of the buffer
	// to return to the caller
	data := make([]T, h.ptr)
	for i := 0; i < h.ptr; i++ {
		// copy(data[i], h.buffer[i])
		data[i] = h.buffer[i]
	}

	// reset the buffer
	h.ptr = 0

	return data
}

// Doesn't work for nested pointers in the struct
// func copy[T any](dest T, source T) {
// 	x := reflect.ValueOf(source)
// 	if x.Kind() == reflect.Ptr {
// 		starX := x.Elem()
// 		y := reflect.New(starX.Type())
// 		starY := y.Elem()
// 		starY.Set(starX)
// 		reflect.ValueOf(dest).Elem().Set(y.Elem())
// 	} else {
// 		dest = x.Interface().(T)
// 	}
// }

func (h *BatchHarvester[T]) start() {
	for {
		select {
		case item, ok := <-h.in:
			if !ok {
				h.Flush()
				goto Stop
			}

			h.append(item)
		case <-h.clock.C:
			h.Flush()
		}
	}

Stop:
	h.stopped <- struct{}{}
}
