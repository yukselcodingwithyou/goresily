package prometheus

// Labels represents a set of label name/value pairs.
type Labels map[string]string

// CounterOpts describes a counter metric.
type CounterOpts struct {
	Name        string
	Help        string
	ConstLabels Labels
}

// Collector is a placeholder for a Prometheus collector.
type Collector interface{}

// Registerer registers Prometheus collectors.
type Registerer interface {
	Register(Collector) error
	MustRegister(...Collector)
	Unregister(Collector) bool
}

// NewCounterFunc returns a no-op collector for counter functions.
func NewCounterFunc(opts CounterOpts, fn func() float64) Collector { return struct{}{} }

// DefaultRegisterer is a no-op registerer used when none is supplied.
var DefaultRegisterer Registerer = noopRegisterer{}

type noopRegisterer struct{}

func (noopRegisterer) Register(Collector) error  { return nil }
func (noopRegisterer) MustRegister(...Collector) {}
func (noopRegisterer) Unregister(Collector) bool { return true }
