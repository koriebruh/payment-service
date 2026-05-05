package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	EventPublishedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_published_total",
			Help: "Total number of events published",
		},
		[]string{"topic", "event_type", "status"},
	)

	EventConsumedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_consumed_total",
			Help: "Total number of events consumed",
		},
		[]string{"topic", "event_type", "status"},
	)

	ActiveGoroutines = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_goroutines",
			Help: "Number of active goroutines",
		},
	)
)
