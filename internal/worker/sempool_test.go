package worker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSempool_Timeout(t *testing.T) {
	p := NewSemPool(1, 1*time.Millisecond)
	defer p.Stop(context.Background())
	ctx := context.Background()

	r := p.Dispatch(ctx, func() {
		time.Sleep(1 * time.Second)
	})
	if r != nil {
		t.Fatalf("expected nil, got %v", r)
	}

	r = p.Dispatch(ctx, func() {})
	if r == nil {
		t.Fatalf("expected error, got nil")
	}

	if r.Error() != "context deadline exceeded" {
		t.Fatalf("expected context deadline exceeded, got %v", r)
	}
}

func TestSempool_Stop(t *testing.T) {
	wg := &sync.WaitGroup{}
	c := make(chan struct{})
	counter := atomic.Int32{}

	p := NewSemPool(10, 1*time.Second)

	ctx := context.Background()

	wg.Add(2)
	_ = p.Dispatch(ctx, func() {
		defer wg.Done()
		<-c
		counter.Add(1)
	})

	_ = p.Dispatch(ctx, func() {
		defer wg.Done()
		<-c
		counter.Add(1)
	})

	p.Stop(ctx)

	c <- struct{}{}
	wg.Wait()

	fmt.Printf("counter: %d\n", counter.Load())
}
