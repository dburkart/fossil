/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */
package server

import (
	"fmt"
	"net"
	"net/http"

	"github.com/dburkart/fossil/pkg/collector"
	"github.com/dburkart/fossil/pkg/proto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

type Server struct {
	log     zerolog.Logger
	metrics MetricsStore

	collectionPort int
	databasePort   int
	metricsPort    int

	msgStream chan proto.Message

	collectors []collector.Collector
}

func New(log zerolog.Logger, collectionPort, databasePort, metricsPort int) Server {
	return Server{
		log,
		NewMetricsStore(),
		collectionPort,
		databasePort,
		metricsPort,
		make(chan proto.Message),
		[]collector.Collector{},
	}
}

func (s Server) ServeDatabase() {
	s.log.Info().Int("database-port", s.databasePort).Msg("listening for client connections")

	go s.listenCollection()

	s.processMessages()
}

func (s *Server) listenCollection() {
	sock, err := net.ListenTCP("tcp4", &net.TCPAddr{Port: s.collectionPort})
	if err != nil {
		s.log.Error().Err(err).Int("port", s.collectionPort).Msg("unable to listen on collection port")
		return
	}
	s.log.Info().Int("collection-port", s.collectionPort).Msg("listening for metrics")

	for {
		conn, err := sock.AcceptTCP()
		if err != nil {
			s.log.Error().Err(err).Msg("unable to accept connection on collection socket")
		}

		col := collector.New(s.log, conn, s.msgStream)
		go col.Handle()
	}
}

func (s *Server) processMessages() {
	for m := range s.msgStream {
		s.log.Info().Str("command", m.Command).Msg("handle")
	}
}

func (s Server) ServeMetrics() {
	s.log.Info().Int("port", s.metricsPort).Msg("/metrics endpoint started")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(fmt.Sprintf(":%d", s.metricsPort), nil)
}
