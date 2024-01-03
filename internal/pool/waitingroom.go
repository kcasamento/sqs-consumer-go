package pool

import "sync/atomic"

type WaitingRoom interface {
	Size() int
	Queue(func())
	Push(func())
	Pop() func()
}

// Very basic and not very good
// implementation of a sliced based
// waiting room
// TODO: Could be better to use a ring buffer
// TODO: other implementations beside a ring buffer?
type SliceWaitingRoom struct {
	size *atomic.Int32
	q    []func()
}

func NewSliceWaitingRoom() *SliceWaitingRoom {
	return &SliceWaitingRoom{
		q:    make([]func(), 10),
		size: &atomic.Int32{},
	}
}

func (wr *SliceWaitingRoom) Size() int {
	return int(wr.size.Load())
}

func (wr *SliceWaitingRoom) Queue(task func()) {
	wr.q = append(wr.q, task)
	wr.size.Add(1)
}

func (wr *SliceWaitingRoom) Push(task func()) {
	wr.q = prependFunc(wr.q, task)
	wr.size.Add(1)
}

func (wr *SliceWaitingRoom) Pop() func() {
	f, q := wr.q[0], wr.q[1:]
	wr.q = q
	wr.size.Add(-1)
	return f
}

func prependFunc(arr []func(), f func()) []func() {
	arr = append(arr, nil)
	copy(arr[1:], arr)
	arr[0] = f
	return arr
}
