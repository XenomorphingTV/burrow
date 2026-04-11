package runner

import (
	"runtime"
	"sync"
	"testing"
)

func TestPoolTryAcquireSucceedsWhenEmpty(t *testing.T) {
	p := NewPool(1)
	if !p.TryAcquire() {
		t.Fatal("TryAcquire should succeed on a fresh pool")
	}
}

func TestPoolTryAcquireFailsAtCapacity(t *testing.T) {
	p := NewPool(1)
	p.TryAcquire()
	if p.TryAcquire() {
		t.Error("TryAcquire should fail when pool is at capacity")
	}
}

func TestPoolTryAcquireSucceedsAfterRelease(t *testing.T) {
	p := NewPool(1)
	p.TryAcquire()
	p.Release()
	if !p.TryAcquire() {
		t.Error("TryAcquire should succeed after Release")
	}
}

func TestPoolAcquireRelease(t *testing.T) {
	p := NewPool(2)
	p.Acquire()
	p.Acquire()
	// Both slots taken; release one and re-acquire without blocking.
	p.Release()
	done := make(chan struct{})
	go func() {
		p.Acquire()
		close(done)
	}()
	<-done
}

func TestPoolZeroDefaultsToOne(t *testing.T) {
	for _, max := range []int{0, -1, -100} {
		p := NewPool(max)
		if !p.TryAcquire() {
			t.Errorf("NewPool(%d): first TryAcquire should succeed", max)
		}
		if p.TryAcquire() {
			t.Errorf("NewPool(%d): pool defaulted to cap>1, expected cap=1", max)
		}
	}
}

func TestPoolMaxParallelEnforced(t *testing.T) {
	const max = 3
	p := NewPool(max)

	var mu sync.Mutex
	running := 0
	peak := 0
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.Acquire()
			defer p.Release()

			mu.Lock()
			running++
			if running > peak {
				peak = running
			}
			mu.Unlock()

			// yield so goroutines overlap
			runtime.Gosched()

			mu.Lock()
			running--
			mu.Unlock()
		}()
	}
	wg.Wait()

	if peak > max {
		t.Errorf("peak concurrency %d exceeded max %d", peak, max)
	}
}
