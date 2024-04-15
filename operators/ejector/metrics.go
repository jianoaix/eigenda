package ejector

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	ejectorNamespace = "ejector"
)

var (
	// The "initiator" could be "periodic" or "requested" (invoked by client of the ejector).
	// The "status" indicates the result of the ejection.
	NumEjection = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: ejectorNamespace,
		Name:      "eigenda_ejection_total",
		Help:      "the total number of ejection attempts",
	}, []string{"status", "initiator"})

	// The "type" could be "eligible" or "ejected". They are recording the number of operators
	// that are eligible to eject and the number that are actually ejected, for each quorum.
	NumOperators = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: ejectorNamespace,
		Name:      "eigenda_operators_total",
		Help:      "the total number of operators to be ejected or actually ejected",
	}, []string{"quorum", "type"})
)
