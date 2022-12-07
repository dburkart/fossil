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

	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/proto"
	"github.com/dburkart/fossil/pkg/query"
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
	}
}

func (s *Server) ServeDatabase() {
	srv := NewMessageServer(s.log)
	mux := NewMapMux()

	// Wire up handlers
	mux.HandleState(proto.CommandUse, s.HandleUse)
	mux.Handle(proto.CommandVersion, s.HandleVersion)
	mux.Handle(proto.CommandQuery, s.HandleQuery)
	mux.Handle(proto.CommandAppend, s.HandleAppend)
	mux.Handle(proto.CommandStats, s.HandleStats)
	mux.Handle(proto.CommandList, s.HandleList)

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

func (s *Server) HandleUse(rw proto.ResponseWriter, c *conn, r *proto.Request) {
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
}

func (s *Server) HandleVersion(rw proto.ResponseWriter, r *proto.Request) {
	version := proto.VersionRequest{}
	err := proto.Unmarshal(r.Data(), &version)
	if err != nil {
		s.log.Error().Err(err).Msg("error unmarshaling")
		rw.WriteMessage(proto.MessageErrorUnmarshaling)
		return
	}
	s.log.Trace().Str("client-version", version.Version).Msg("got client version")
	// We don't currently reject any versions, so proceed to send our own version
	// announcement with an OK code.
	versionResponse := proto.VersionResponse{Code: 200}
	rw.WriteMessage(proto.NewMessageWithType(proto.CommandVersion, versionResponse))
}

func (s *Server) HandleAppend(rw proto.ResponseWriter, r *proto.Request) {
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
}

func (s *Server) HandleQuery(rw proto.ResponseWriter, r *proto.Request) {
	q := proto.QueryRequest{}

	err := proto.Unmarshal(r.Data(), &q)
	if err != nil {
		s.log.Error().Err(err).Msg("error unmarshaling")
		rw.WriteMessage(proto.MessageErrorUnmarshaling)
		return
	}

	stmt, err := query.Prepare(r.Database(), q.Query)
	if err != nil {
		rw.WriteMessage(proto.NewMessageWithType(proto.CommandError, proto.ErrResponse{Code: 504, Err: err}))
		return
	}
	result := stmt.Execute()

	resp := proto.QueryResponse{}
	resp.Results = result.Data

	_, err = rw.WriteMessage(proto.NewMessageWithType(proto.CommandQuery, resp))
	if err != nil {
		s.log.Error().Err(err).Msg("unable to write response")
		rw.WriteMessage(proto.MessageErrorUnmarshaling)
		return
	}
}

func (s *Server) HandleStats(rw proto.ResponseWriter, r *proto.Request) {
	// FIXME: This should be updated periodically in it's own runloop, not computed on request
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	resp := proto.StatsResponse{
		AllocHeap: m.Alloc,
		TotalMem:  m.Sys,
		Uptime:    time.Since(s.startupTime),
		Segments:  len(r.Database().Segments),
	}
	rw.WriteMessage(proto.NewMessageWithType(proto.CommandStats, resp))
}

func (s *Server) HandleList(rw proto.ResponseWriter, r *proto.Request) {
	resp := proto.ListResponse{
		DatabaseList: []string{},
	}
	for k := range s.dbMap {
		resp.DatabaseList = append(resp.DatabaseList, k)
	}
	rw.WriteMessage(proto.NewMessageWithType(proto.CommandStats, resp))
}
