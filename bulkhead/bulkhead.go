package bulkhead

import (
	"errors"
	"sync/atomic"
)

// ErrFull is returned when the concurrency limit is exceeded.
var ErrFull = errors.New("bulkhead full")

// Bulkhead limits the number of concurrent executions.
type Bulkhead struct {
	sem         chan struct{}
	execCount   uint64
	rejectCount uint64
}

// Metrics holds counters for bulkhead activity.
type Metrics struct {
	Executed uint64
	Rejected uint64
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
	return &Bulkhead{sem: make(chan struct{}, b.limit)}
}

// Execute runs fn if the limit has not been reached.
func (bh *Bulkhead) Execute(fn func() error) error {
	select {
	case bh.sem <- struct{}{}:
		atomic.AddUint64(&bh.execCount, 1)
		defer func() { <-bh.sem }()
		return fn()
	default:
		atomic.AddUint64(&bh.rejectCount, 1)
		return ErrFull
	}
}

// Metrics returns current counters for the bulkhead.
func (bh *Bulkhead) Metrics() Metrics {
	return Metrics{
		Executed: atomic.LoadUint64(&bh.execCount),
		Rejected: atomic.LoadUint64(&bh.rejectCount),
	}
}
