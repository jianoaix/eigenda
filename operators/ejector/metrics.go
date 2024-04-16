package ejector

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	ejectorNamespace = "ejector"
)

type MetricsConfig struct {
	HTTPPort      string
	EnableMetrics bool
}

type Metrics struct {
	registry *prometheus.Registry
}

var (
	// The "requestor" could be "periodic" (internally initiated ejection) or "external"
	// (invoked by an external client of the ejector).
	// The "status" indicates the result of the ejection request.
	EjectionRequest = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: ejectorNamespace,
		Name:      "eigenda_ejection_request_total",
		Help:      "the total number of ejection requests",
	}, []string{"status", "requestor"})

	// The "state" could be "eligible" (at the moment of ejection requested) or "ejected", and the
	// "type" could be "number" or "stake".
	// These are recording the operators that are eligible to eject and that are actually ejected,
	// for each quorum.
	// By the "type" label it'll record both the number of these operators as well as the stake
	// they represent.
	Operators = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: ejectorNamespace,
		Name:      "eigenda_operators_total",
		Help:      "the total number of operators to be ejected or actually ejected",
	}, []string{"quorum", "state", "type"})
)
