package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// ErrOpen is returned when the breaker is open and calls are blocked.
var ErrOpen = errors.New("circuit breaker is open")

// State represents the breaker state.
type State int

const (
	// Closed allows calls to pass through.
	Closed State = iota
	// Open rejects calls.
	Open
	// HalfOpen allows a limited number of test calls.
	HalfOpen
)

func (s State) String() string {
	switch s {
	case Closed:
		return "Closed"
	case Open:
		return "Open"
	case HalfOpen:
		return "HalfOpen"
	default:
		return "Unknown"
	}
}

// CircuitBreaker controls access to an operation based on failures.
type CircuitBreaker struct {
	maxFailures   int
	window        time.Duration
	timeout       time.Duration
	trialRequests int
	trialDuration time.Duration
	mu            sync.Mutex
	failureTimes  []time.Time
	trialCount    int
	trialStart    time.Time
	state         State
	onStateChange func(State)
}

// Builder constructs a CircuitBreaker.
type Builder struct {
	maxFailures   int
	window        time.Duration
	timeout       time.Duration
	trialRequests int
	trialDuration time.Duration
	onStateChange func(State)
}

// NewBuilder returns a builder with default values.
func NewBuilder() *Builder {
	return &Builder{
		maxFailures: 3,
		timeout:     time.Second,
	}
}

// MaxFailures sets the number of failures before opening the breaker.
func (b *Builder) MaxFailures(n int) *Builder {
	b.maxFailures = n
	return b
}

// Timeout sets the duration the breaker remains open.
func (b *Builder) Timeout(d time.Duration) *Builder {
	b.timeout = d
	return b
}

// Window sets the time window for counting failures.
func (b *Builder) Window(d time.Duration) *Builder {
	b.window = d
	return b
}

// TrialRequests sets the number of calls allowed in half-open state before closing.
func (b *Builder) TrialRequests(n int) *Builder {
	b.trialRequests = n
	return b
}

// TrialDuration sets the time allowed for half-open calls before re-opening.
func (b *Builder) TrialDuration(d time.Duration) *Builder {
	b.trialDuration = d
	return b
}

// OnStateChange registers a callback for state changes.
func (b *Builder) OnStateChange(fn func(State)) *Builder {
	b.onStateChange = fn
	return b
}

// Build creates the CircuitBreaker.
func (b *Builder) Build() *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:   b.maxFailures,
		window:        b.window,
		timeout:       b.timeout,
		trialRequests: b.trialRequests,
		trialDuration: b.trialDuration,
		state:         Closed,
		onStateChange: b.onStateChange,
	}
}

func (cb *CircuitBreaker) setState(s State) {
	if cb.state != s {
		cb.state = s
		if cb.onStateChange != nil {
			cb.onStateChange(s)
		}
	}
}

// Execute runs fn if the breaker is closed.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()
	state := cb.state
	// in half-open check time window
	if state == HalfOpen && cb.trialDuration > 0 && time.Since(cb.trialStart) > cb.trialDuration {
		cb.setState(Open)
		cb.startOpenTimer()
		cb.mu.Unlock()
		return ErrOpen
	}
	cb.mu.Unlock()

	if state == Open {
		return ErrOpen
	}

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()
	switch state {
	case Closed:
		if err != nil {
			cb.recordFailure()
			if len(cb.failureTimes) >= cb.maxFailures {
				cb.resetFailures()
				cb.setState(Open)
				cb.startOpenTimer()
			}
			return err
		}
		cb.resetFailures()
		return nil
	case HalfOpen:
		cb.trialCount++
		if err != nil {
			cb.setState(Open)
			cb.startOpenTimer()
			return err
		}
		if cb.trialRequests == 0 || cb.trialCount >= cb.trialRequests {
			cb.setState(Closed)
			cb.resetFailures()
		}
		return err
	default:
		return err
	}
}

func (cb *CircuitBreaker) recordFailure() {
	now := time.Now()
	cb.failureTimes = append(cb.failureTimes, now)
	if cb.window > 0 {
		cutoff := now.Add(-cb.window)
		i := 0
		for ; i < len(cb.failureTimes); i++ {
			if cb.failureTimes[i].After(cutoff) {
				break
			}
		}
		if i > 0 {
			cb.failureTimes = cb.failureTimes[i:]
		}
	}
}

func (cb *CircuitBreaker) resetFailures() {
	cb.failureTimes = nil
	cb.trialCount = 0
}

func (cb *CircuitBreaker) startOpenTimer() {
	time.AfterFunc(cb.timeout, func() {
		cb.mu.Lock()
		defer cb.mu.Unlock()
		cb.setState(HalfOpen)
		cb.trialStart = time.Now()
		cb.trialCount = 0
	})
}
