/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */
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
			Name: "fossil_client_connections",
			Help: "The total number of client connections",
		}),
	}
}

func (ms *metricsStore) IncClientConnection() {
	ms.ClientConnections.Inc()
}
