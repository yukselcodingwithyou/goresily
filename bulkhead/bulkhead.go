package bulkhead

import (
	"errors"
	"sync/atomic"
)

// ErrFull is returned when the concurrency limit is exceeded.
var ErrFull = errors.New("bulkhead full")

// Bulkhead limits the number of concurrent executions.
type Bulkhead struct {
	sem     chan struct{}
	limit   int
	current int64 // atomic counter for current usage
}

// Builder constructs a Bulkhead.
type Builder struct {
	limit int
}

// NewBuilder returns a builder with default values.
func NewBuilder() *Builder {
	return &Builder{limit: 1}
}

// Limit sets the maximum number of concurrent executions.
func (b *Builder) Limit(n int) *Builder {
	b.limit = n
	return b
}

// Build creates the Bulkhead.
func (b *Builder) Build() *Bulkhead {
	return &Bulkhead{
		sem:   make(chan struct{}, b.limit),
		limit: b.limit,
	}
}

// Execute runs fn if the limit has not been reached.
func (bh *Bulkhead) Execute(fn func() error) error {
	select {
	case bh.sem <- struct{}{}:
		atomic.AddInt64(&bh.current, 1)
		defer func() { 
			<-bh.sem 
			atomic.AddInt64(&bh.current, -1)
		}()
		return fn()
	default:
		return ErrFull
	}
}

// CurrentUsage returns the current number of executing requests
func (bh *Bulkhead) CurrentUsage() int {
	return int(atomic.LoadInt64(&bh.current))
}

// Limit returns the maximum capacity
func (bh *Bulkhead) Limit() int {
	return bh.limit
}
