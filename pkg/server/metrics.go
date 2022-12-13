/*
 * Copyright (c) 2022, Gideon Williams <gideon@gideonw.com>
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
	IncRequests(db, cmd string)
	ObserveResponseNS(db, cmd string, t int64)
}

type metricsStore struct {
	ClientConnections prometheus.Counter
	Requests          *prometheus.CounterVec
	ResponseNS        *prometheus.HistogramVec
}

var (
	DatabaseLabel = "database"
	CommandLabel  = "cmd"
)

func NewMetricsStore() MetricsStore {
	return &metricsStore{
		ClientConnections: promauto.NewCounter(prometheus.CounterOpts{
			Name: "fossil_client_connections",
			Help: "The total number of client connections",
		}),
		Requests: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "fossil_requests",
			Help: "Request counts for the fossil commands",
		}, []string{DatabaseLabel, CommandLabel}),
		ResponseNS: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name: "fossil_response_ns",
			Help: "Response times on commands made against a database",
		}, []string{DatabaseLabel, CommandLabel}),
	}
}

func (ms *metricsStore) IncClientConnection() {
	ms.ClientConnections.Inc()
}

func (ms *metricsStore) IncRequests(db, cmd string) {
	ms.Requests.With(prometheus.Labels{CommandLabel: cmd, DatabaseLabel: db}).Inc()
}

func (ms *metricsStore) ObserveResponseNS(db, cmd string, t int64) {
	ms.ResponseNS.
		With(prometheus.Labels{CommandLabel: cmd, DatabaseLabel: db}).
		Observe(float64(t))
}
