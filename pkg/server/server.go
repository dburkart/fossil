/*
 * Copyright (c) 2022, Gideon Williams <gideon@gideonw.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/dburkart/fossil/pkg/collector"
	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/proto"
	"github.com/dburkart/fossil/pkg/query"
	"github.com/dustin/go-humanize"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

type Server struct {
	log         zerolog.Logger
	metrics     MetricsStore
	startupTime time.Time

	dbMap       map[string]*database.Database
	port        int
	metricsPort int

	collectors []collector.Collector
}

type DatabaseConfig struct {
	Name      string
	Directory string
}

func New(log zerolog.Logger, dbConfigs map[string]DatabaseConfig, port, metricsPort int) Server {
	// TODO: We need a filesystem lock to ensure we don't double run a server on the same database
	// https://pkg.go.dev/io/fs#FileMode ModeExclusive

	// take the db configs and build a map of databases name -> db
	dbMap := make(map[string]*database.Database)
	for k, v := range dbConfigs {
		log.Info().Str("name", v.Name).Str("directory", v.Directory).Msg("initializing database")
		dbMap[k] = database.NewDatabase(v.Directory)
	}

	return Server{
		log,
		NewMetricsStore(),
		time.Now(),
		dbMap,
		port,
		metricsPort,
		[]collector.Collector{},
	}
}

func (s *Server) ServeDatabase() {
	srv := NewMessageServer(s.log)
	mux := NewMapMux()

	mux.Handle(proto.CommandQuery, func(w io.Writer, msg proto.Message) {
		db := "default"
		stmt := query.Prepare(s.dbMap[db], string(msg.Data))
		result := stmt.Execute()

		ret := new(bytes.Buffer)
		for _, val := range result.Data {
			n, err := ret.WriteString(val.ToString() + "\n")
			if err != nil {
				s.log.Error().Err(err).Msg("unable to write to results buffer")
			}
			s.log.Trace().Int("wrote", n).Msg("wrote to results buffer")
		}

		if ret.Len() == 0 {
			w.Write([]byte("No Results\n"))
			return
		}

		n, err := w.Write(append(ret.Bytes(), '\n'))
		if err != nil {
			s.log.Error().Err(err).Msg("unable to write response")
		}
		s.log.Trace().Int("wrote", n).Msg("wrote response")
	})

	mux.Handle(proto.CommandAppend, func(w io.Writer, msg proto.Message) {
		s.dbMap["default"].Append(msg.Data, "")
		w.Write([]byte("Ok!\n"))
	})

	mux.Handle(proto.CommandStats, func(w io.Writer, msg proto.Message) {
		s.log.Info().Msg("INFO command")
		// FIXME: This should be updated periodically in it's own runloop, not computed on request
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		w.Write([]byte(fmt.Sprintf(
			"Allocated Heap: %v\nTotal Memory: %v\nUptime: %s\nSegments: %d\n",
			humanize.Bytes(m.Alloc),
			humanize.Bytes(m.Sys),
			time.Now().Sub(s.startupTime).String(),
			len(s.dbMap["default"].Segments),
		)))
	})

	err := srv.ListenAndServe(s.port, mux)
	if err != nil {
		s.log.Error().Err(err).Msg("error listening and serving")
	}
}

func (s *Server) ServeMetrics() {
	s.log.Info().Int("port", s.metricsPort).Msg("/metrics endpoint started")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(fmt.Sprintf(":%d", s.metricsPort), nil)
}
