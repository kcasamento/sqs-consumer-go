package pool

type WaitingRoom interface {
	Size() int
	Queue(func())
	Push(func())
	Pop() func()
}

// Very basic and not very good
// implementation of a sliced based
// waiting room
// Could be better to use a ring buffer
type SliceWaitingRoom struct {
	q []func()
}

func NewSliceWaitingRoom() *SliceWaitingRoom {
	return &SliceWaitingRoom{
		q: make([]func(), 10),
	}
}

func (wr *SliceWaitingRoom) Size() int {
	return len(wr.q)
}

func (wr *SliceWaitingRoom) Queue(task func()) {
	wr.q = append(wr.q, task)
}

func (wr *SliceWaitingRoom) Push(task func()) {
	wr.q = prependFunc(wr.q, task)
}

func (wr *SliceWaitingRoom) Pop() func() {
	f, q := wr.q[0], wr.q[1:]
	wr.q = q
	return f
}

func prependFunc(arr []func(), f func()) []func() {
	arr = append(arr, nil)
	copy(arr[1:], arr)
	arr[0] = f
	return arr
}
