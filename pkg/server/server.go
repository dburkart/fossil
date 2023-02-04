/*
 * Copyright (c) 2022, Gideon Williams <gideon@gideonw.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package server

import (
	"fmt"
	"net/http"
	"path"
	"runtime"
	"time"

	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/proto"
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

	// Setup metrics store to add collectors to
	ms := NewMetricsStore()

	// take the db configs and build a map of databases name -> db
	dbMap := make(map[string]*database.Database)
	for k, v := range dbConfigs {
		log.Info().Str("name", v.Name).Str("directory", v.Directory).Msg("initializing database")
		dbLogger := log.With().Str("db", v.Name).Logger()
		db, err := database.NewDatabase(v.Name, path.Join(v.Directory, v.Name))
		if err != nil {
			dbLogger.Fatal().Err(err).Msg("error initializing database")
		}
		dbMap[k] = db
		ms.RegisterCollector(NewDBStatsCollector(db))
	}

	return Server{
		log,
		ms,
		time.Now(),
		dbMap,
		port,
		metricsPort,
	}
}

func (s *Server) accessLog(log zerolog.Logger, h MessageHandler) MessageHandler {
	return func(rw proto.ResponseWriter, r *proto.Request) {
		t := time.Now()
		defer func() {
			dur := time.Since(t).Nanoseconds()
			db := "unset"
			if r.Database() != nil {
				db = r.Database().Name
			}
			log.Info().Int64("ns", dur).Str("cmd", r.Command()).Str("db", db).Send()
			s.metrics.IncRequests(db, r.Command())
			s.metrics.ObserveResponseNS(db, r.Command(), dur)
		}()
		h(rw, r)
	}
}

func (s *Server) ServeDatabase() {
	srv := NewMessageServer(s.log, s.metrics)
	mux := NewMapMux()

	// Wire up handlers
	mux.HandleState(proto.CommandUse, s.HandleUse)
	mux.Handle(proto.CommandVersion, s.accessLog(s.log, s.HandleVersion))
	mux.Handle(proto.CommandQuery, s.accessLog(s.log, s.HandleQuery))
	mux.Handle(proto.CommandAppend, s.accessLog(s.log, s.HandleAppend))
	mux.Handle(proto.CommandStats, s.accessLog(s.log, s.HandleStats))
	mux.Handle(proto.CommandList, s.accessLog(s.log, s.HandleList))
	mux.Handle(proto.CommandCreate, s.accessLog(s.log, s.HandleCreate))

	err := srv.ListenAndServe(s.port, mux)
	if err != nil {
		s.log.Error().Err(err).Msg("error listening and serving")
	}
}

func (s *Server) ServeMetrics() {
	s.log.Info().Int("port", s.metricsPort).Msg("/metrics endpoint started")
	http.Handle("/metrics", s.metrics.Handler())
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
	rw.WriteMessage(VersionResponse(version))
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
	rw.WriteMessage(AppendResponse(a, r.Database()))
}

func (s *Server) HandleQuery(rw proto.ResponseWriter, r *proto.Request) {
	q := proto.QueryRequest{}

	err := proto.Unmarshal(r.Data(), &q)
	if err != nil {
		s.log.Error().Err(err).Msg("error unmarshaling")
		rw.WriteMessage(proto.MessageErrorUnmarshaling)
		return
	}

	_, err = rw.WriteMessage(QueryResponse(q, r.Database()))
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
		Topics:    r.Database().TopicCount,
	}
	rw.WriteMessage(proto.NewMessageWithType(proto.CommandStats, resp))
}

func (s *Server) HandleList(rw proto.ResponseWriter, r *proto.Request) {
	l := proto.ListRequest{}

	err := proto.Unmarshal(r.Data(), &l)
	if err != nil {
		s.log.Error().Err(err).Msg("error unmarshaling")
		rw.WriteMessage(proto.MessageErrorUnmarshaling)
		return
	}

	rw.WriteMessage(ListResponse(l, r.Database(), s.dbMap))
}

func (s *Server) HandleCreate(rw proto.ResponseWriter, r *proto.Request) {
	c := proto.CreateTopicRequest{}

	err := proto.Unmarshal(r.Data(), &c)
	if err != nil {
		s.log.Error().Err(err).Msg("error unmarshaling")
		rw.WriteMessage(proto.MessageErrorUnmarshaling)
		return
	}

	rw.WriteMessage(CreateResponse(c, r.Database()))
}
