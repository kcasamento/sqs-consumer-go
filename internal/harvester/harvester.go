package harvester

import (
	"sync"
	"time"
)

type BatchHarvester[T any] struct {
	clock         *time.Ticker
	in            chan T
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
) *BatchHarvester[T] {
	h := &BatchHarvester[T]{
		in:            make(chan T, 1024),
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
	h.in <- item
	h.clock.Reset(h.flushInterval)
}

func (h *BatchHarvester[T]) Stop() {
	close(h.in)
	h.clock.Stop()

	<-h.stopped
}

func (h *BatchHarvester[T]) Flush() {
	// try and flush buffered
	// if data is empty ignore
	// else notify the callback

	data := h.flushData()
	if len(data) == 0 {
		return
	}

	h.onBatch(data)
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

	// make a copy of the buffer
	// to return to the caller
	data := make([]T, h.ptr)
	for i := 0; i < h.ptr; i++ {
		data[i] = h.buffer[i]
	}

	// reset the buffer
	h.ptr = 0

	return data
}

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
