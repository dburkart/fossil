/*
 * Copyright (c) 2022, Gideon Williams <gideon@gideonw.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package server

import (
	"fmt"
	"net"
	"net/http"
	"strings"

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

	msgStream chan proto.Message

	collectors []collector.Collector
}

type commandHandler func(s *Server, message proto.Message) error

var commandMap = map[string]commandHandler{
	"APPEND": appendToDB,
}

func appendToDB(s *Server, message proto.Message) error {
	// TODO: Support topics
	s.database.Append(message.Data, "")
	return nil
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
		make(chan proto.Message),
		[]collector.Collector{},
	}
}

func (s *Server) ServeDatabase() {
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

		if handler, exists := commandMap[strings.ToUpper(m.Command)]; exists {
			err := handler(s, m)
			if err != nil {
				s.log.Error().Err(err)
			}
		}
	}
}

func (s *Server) ServeMetrics() {
	s.log.Info().Int("port", s.metricsPort).Msg("/metrics endpoint started")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(fmt.Sprintf(":%d", s.metricsPort), nil)
}
