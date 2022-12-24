/*
 * Copyright (c) 2022, Gideon Williams <gideon@gideonw.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package server

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsStore interface {
	Registry() *prometheus.Registry
	RegisterCollector(c prometheus.Collector)
	Handler() http.Handler

	// Collection
	IncClientConnection()
	IncRequests(db, cmd string)
	ObserveResponseNS(db, cmd string, t int64)
}

type metricsStore struct {
	registry          *prometheus.Registry
	ClientConnections prometheus.Counter
	Requests          *prometheus.CounterVec
	ResponseNS        *prometheus.HistogramVec
}

var (
	DatabaseLabel = "database"
	CommandLabel  = "cmd"
)

func NewMetricsStore() MetricsStore {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(
			collectors.WithGoCollectorRuntimeMetrics(collectors.MetricsAll),
		),
	)

	buckets := []float64{}
	for i := 1; i < 20; i++ {
		buckets = append(buckets, float64(2*i*int(time.Millisecond)))
	}

	factory := promauto.With(reg)
	return &metricsStore{
		registry: reg,
		ClientConnections: factory.NewCounter(prometheus.CounterOpts{
			Name: "fossil_client_connections",
			Help: "The total number of client connections",
		}),
		Requests: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "fossil_requests",
			Help: "Request counts for the fossil commands",
		}, []string{DatabaseLabel, CommandLabel}),
		ResponseNS: factory.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "fossil_response_ns",
			Help:    "Response times on commands made against a database",
			Buckets: buckets,
		}, []string{DatabaseLabel, CommandLabel}),
	}
}

func (ms *metricsStore) Registry() *prometheus.Registry {
	return ms.registry
}

func (ms *metricsStore) RegisterCollector(c prometheus.Collector) {
	ms.registry.MustRegister(c)
}

func (ms *metricsStore) Handler() http.Handler {
	return promhttp.HandlerFor(ms.Registry(), promhttp.HandlerOpts{Registry: ms.Registry()})
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
