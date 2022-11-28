/*
 * Copyright (c) 2022, Gideon Williams <gideon@gideonw.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package server

import (
	"fmt"
	"io"
	"net/http"

	"github.com/dburkart/fossil/pkg/collector"
	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/proto"
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

	collectors []collector.Collector
}

func New(log zerolog.Logger, path string, collectionPort, databasePort, metricsPort int) Server {
	// TODO: We need a filesystem lock to ensure we don't double run a server on the same database
	// https://pkg.go.dev/io/fs#FileMode ModeExclusive
	db := database.NewDatabase(path)

	return Server{
		log,
		NewMetricsStore(),
		db,
		collectionPort,
		databasePort,
		metricsPort,
		[]collector.Collector{},
	}
}

func (s *Server) ServeDatabase() {
	s.log.Info().Int("database-port", s.databasePort).Msg("listening for client connections")

	srv := NewMessageServer(s.log)
	mux := NewMapMux()

	mux.Handle(proto.CommandAppend, func(w io.Writer, msg proto.Message) {
		s.database.Append(msg.Data, "")
		w.Write([]byte("Ok!\n"))
	})

	mux.Handle(proto.CommandInfo, func(w io.Writer, msg proto.Message) {
		s.log.Info().Msg("INFO command")
		w.Write([]byte("hello world\n"))
	})

	err := srv.ListenAndServe(s.collectionPort, mux)
	if err != nil {
		s.log.Error().Err(err).Msg("error listening and serving")
	}
}

func (s *Server) ServeMetrics() {
	s.log.Info().Int("port", s.metricsPort).Msg("/metrics endpoint started")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(fmt.Sprintf(":%d", s.metricsPort), nil)
}
