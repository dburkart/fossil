/*
 * Copyright (c) 2022, Gideon Williams <gideon@gideonw.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package server

import (
	"fmt"
	"github.com/dburkart/fossil/pkg/database"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

type Server struct {
	log     zerolog.Logger
	metrics MetricsStore

	database       *database.Database
	collectionPort int
	databasePort   int
	metricsPort    int
}

func New(log zerolog.Logger, path string, collectionPort, databasePort, metricsPort int) Server {
	// TODO: We need a filesystem lock to ensure we don't double run a server on the same database
	db := database.NewDatabase(path)

	return Server{
		log,
		NewMetricsStore(),
		db,
		collectionPort,
		databasePort,
		metricsPort,
	}
}

func (s Server) ServeDatabase() {
	s.log.Info().Int("database-port", s.databasePort).Msg("listening for client connections")
	s.log.Info().Int("collection-port", s.collectionPort).Msg("listening for metrics")
}

func (s Server) ServeMetrics() {
	s.log.Info().Int("port", s.metricsPort).Msg("/metrics endpoint started")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(fmt.Sprintf(":%d", s.metricsPort), nil)
}
