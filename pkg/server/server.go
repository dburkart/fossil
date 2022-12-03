/*
 * Copyright (c) 2022, Gideon Williams <gideon@gideonw.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package server

import (
	"fmt"
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
		dbLogger := log.With().Str("db", v.Name).Logger()
		db, err := database.NewDatabase(dbLogger, v.Name, v.Directory)
		if err != nil {
			dbLogger.Fatal().Err(err).Msg("error initializing database")
		}
		dbMap[k] = db
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

	mux.HandleState(proto.CommandUse, func(rw proto.ResponseWriter, c *conn, r *proto.Request) {
		use := proto.UseRequest{}
		err := proto.Unmarshal(r.Data(), &use)
		if err != nil {
			s.log.Error().Err(err).Msg("error unmarshaling")
			rw.WriteMessage(proto.MessageErrorUnmarshaling)
			return
		}
		db, ok := s.dbMap[use.DbName]
		if !ok {
			s.log.Error().Err(err).Str("dbName", use.DbName).Msg("error unknown db")
			rw.WriteMessage(proto.MessageErrorUnknownDb)
			return
		}
		c.SetDatabase(use.DbName, db)

		rw.WriteMessage(proto.MessageOkDatabaseChanged)
	})

	mux.Handle(proto.CommandQuery, func(rw proto.ResponseWriter, r *proto.Request) {
		q := proto.QueryRequest{}

		err := proto.Unmarshal(r.Data(), &q)
		if err != nil {
			s.log.Error().Err(err).Msg("error unmarshaling")
			rw.WriteMessage(proto.MessageErrorUnmarshaling)
			return
		}

		stmt := query.Prepare(r.Database(), q.Query)
		result := stmt.Execute()

		resp := proto.QueryResponse{}
		resp.Results = result.Data

		_, err = rw.WriteMessage(resp)
		if err != nil {
			s.log.Error().Err(err).Msg("unable to write response")
			rw.WriteMessage(proto.MessageErrorUnmarshaling)
			return
		}
	})

	mux.Handle(proto.CommandAppend, func(rw proto.ResponseWriter, r *proto.Request) {
		a := proto.AppendRequest{}
		err := proto.Unmarshal(r.Data(), &a)
		if err != nil {
			s.log.Error().Err(err).Msg("error unmarshaling")
			rw.WriteMessage(proto.MessageErrorUnmarshaling)
			return
		}

		s.log.Trace().Str("topic", a.Topic).Msg("append")
		r.Database().Append(a.Data, a.Topic)
		rw.WriteMessage(proto.MessageOk)
	})

	mux.Handle(proto.CommandStats, func(rw proto.ResponseWriter, r *proto.Request) {
		s.log.Info().Msg("INFO command")
		// FIXME: This should be updated periodically in it's own runloop, not computed on request
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		rw.Write([]byte(fmt.Sprintf(
			"Allocated Heap: %v\nTotal Memory: %v\nUptime: %s\nSegments: %d\n",
			humanize.Bytes(m.Alloc),
			humanize.Bytes(m.Sys),
			time.Now().Sub(s.startupTime).String(),
			len(r.Database().Segments),
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
