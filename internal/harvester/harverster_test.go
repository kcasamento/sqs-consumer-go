package harvester

import (
	"fmt"
	"testing"
	"time"
)

func TestBatchHarvester_BatchesItems(t *testing.T) {
	results := make([]int, 0)
	flushCalled := 0

	h := NewBatchHarvester[int](
		func(data []int) {
			// log.Printf("flushing %d items", len(data))
			results = append(results, data...)
			flushCalled++
		},
		2,
		10*time.Second,
	)
	h.Add(1)
	h.Add(2)
	h.Add(3)

	h.Stop()

	if flushCalled != 2 {
		t.Fatalf("expected flush to be called twice, got %d", flushCalled)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
}

func TestBatchHarvester_FlushItemsInterval(t *testing.T) {
	results := make([]int, 0)
	flushCalled := 0

	h := NewBatchHarvester[int](
		func(data []int) {
			results = append(results, data...)
			flushCalled++
		},
		3,
		1*time.Microsecond,
	)
	h.Add(1)
	h.Add(2)

	h.Stop()

	if flushCalled != 1 {
		t.Fatalf("expected flush to be called twice, got %d", flushCalled)
	}

	if len(results) != 2 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
}

var print bool = false

func BenchmarkBatchHarvester_int(b *testing.B) {
	cb := func(data []int) {
		if print {
			fmt.Printf("%d\n", len(data))
		}
	}
	h := NewBatchHarvester[int](
		cb,
		10,
		10*time.Second,
	)

	for n := 0; n < b.N; n++ {
		h.Add(n)
	}

	h.Stop()
}

func BenchmarkBatchHarvester_string(b *testing.B) {
	cb := func(data []string) {
		if print {
			fmt.Printf("%d\n", len(data))
		}
	}
	h := NewBatchHarvester[string](
		cb,
		10,
		10*time.Second,
	)

	for n := 0; n < b.N; n++ {
		h.Add(fmt.Sprintf("%d", n))
	}

	h.Stop()
}

type testStruct struct {
	i int
}

func BenchmarkBatchHarvester_struct(b *testing.B) {
	cb := func(data []*testStruct) {
		if print {
			fmt.Printf("%d\n", len(data))
		}
	}
	h := NewBatchHarvester[*testStruct](
		cb,
		10,
		10*time.Second,
	)

	for n := 0; n < b.N; n++ {
		h.Add(&testStruct{i: n})
	}

	h.Stop()
}

type testCopyStruct struct {
	s     string
	i     int
	inner *testStruct
}

func TestCopy(t *testing.T) {
	buffer := make([]testCopyStruct, 3)
	buffer[0] = testCopyStruct{s: "test0", i: 0, inner: &testStruct{i: 0}}
	buffer[1] = testCopyStruct{s: "test1", i: 1, inner: &testStruct{i: 1}}
	buffer[2] = testCopyStruct{s: "test2", i: 2, inner: &testStruct{i: 2}}

	data := make([]testCopyStruct, 3)
	for i := 0; i < 3; i++ {
		copy(data[i], buffer[i])
		data[i].s = fmt.Sprintf("new_test%d", i)
		// data[i].inner.i = i * 10
	}

	fmt.Printf("old=%v", buffer)
	fmt.Printf("new=%v", data)
}
