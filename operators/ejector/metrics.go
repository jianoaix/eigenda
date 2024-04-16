package ejector

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	ejectorNamespace = "ejector"
)

type MetricsConfig struct {
	MetricsPort   string
	EnableMetrics bool
}

type Metrics struct {
	registry *prometheus.Registry

	EjectionRequest *prometheus.CounterVec
	Operators       *prometheus.CounterVec
}

func NewMetrics() *Metrics {
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	reg.MustRegister(collectors.NewGoCollector())

	metrics := &Metrics{
		// The "requestor" could be:
		// - "periodic": internally initiated ejection; or
		// - "external": invoked by an external client of the ejector
		// The "status" indicates the final processing result of the ejection request.
		EjectionRequests: promauto.With(reg).NewCounterVec(
			prometheus.CounterOpts{
				Namespace: ejectorNamespace,
				Name:      "eigenda_ejection_requests_total",
				Help:      "the total number of ejection requests",
			},
			[]string{"status", "requestor"},
		),

		// The "state" could be:
		// - "requested": operator is requested for ejection; or
		// - "ejected": operator is actually ejected
		// The  "type" could be "number" or "stake", for the number of operators as well as the
		// stake they represent.
		Operators: promauto.With(reg).NewCounterVec(
			prometheus.CounterOpts{
				Namespace: ejectorNamespace,
				Name:      "eigenda_operators_total",
				Help:      "the total number of operators to be ejected or actually ejected",
			}, []string{"quorum", "state", "type"},
		),
	}

	return metric
}
