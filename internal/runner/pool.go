package runner

// Pool is a semaphore-based concurrency limiter.
type Pool struct {
	sem chan struct{}
}

// NewPool returns a Pool that limits concurrent task execution to max slots.
func NewPool(max int) *Pool {
	if max <= 0 {
		max = 1
	}
	sem := make(chan struct{}, max)
	for i := 0; i < max; i++ {
		sem <- struct{}{}
	}
	return &Pool{sem: sem}
}

// Acquire blocks until a slot is available.
func (p *Pool) Acquire() {
	<-p.sem
}

// TryAcquire attempts to acquire a slot without blocking.
// Returns true if successful, false if pool is at capacity.
func (p *Pool) TryAcquire() bool {
	select {
	case <-p.sem:
		return true
	default:
		return false
	}
}

// Release returns a slot to the pool.
func (p *Pool) Release() {
	p.sem <- struct{}{}
}
