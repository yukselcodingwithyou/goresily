package bulkhead

import "errors"

// ErrFull is returned when the concurrency limit is exceeded.
var ErrFull = errors.New("bulkhead full")

// Bulkhead limits the number of concurrent executions.
type Bulkhead struct {
	sem chan struct{}
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
		defer func() { <-bh.sem }()
		return fn()
	default:
		return ErrFull
	}
}
