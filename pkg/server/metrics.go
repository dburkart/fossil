package server

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type MetricsStore interface {
	IncClientConnection()
}

type metricsStore struct {
	ClientConnections prometheus.Counter
}

func NewMetricsStore() MetricsStore {
	return &metricsStore{
		ClientConnections: promauto.NewCounter(prometheus.CounterOpts{
			Name: "fossil_connections",
			Help: "The total number of client connections",
		}),
	}
}

func (ms *metricsStore) IncClientConnection() {
	ms.ClientConnections.Inc()
}
