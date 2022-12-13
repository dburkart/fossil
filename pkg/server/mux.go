/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package server

import (
	"io"
	"net"

	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/proto"
	"github.com/rs/zerolog"
)

type MessageMux interface {
	ServeMessage(c *conn, r *proto.Request)
	Handle(s string, f MessageHandler)
	HandleState(s string, f MessageStateHandler)
}

type MessageHandler func(proto.ResponseWriter, *proto.Request)
type MessageStateHandler func(proto.ResponseWriter, *conn, *proto.Request)

type MapMux struct {
	handlers      map[string]MessageHandler
	stateHandlers map[string]MessageStateHandler
}

func NewMapMux() MessageMux {
	return &MapMux{
		handlers:      make(map[string]MessageHandler),
		stateHandlers: make(map[string]MessageStateHandler),
	}
}

func (mm *MapMux) ServeMessage(c *conn, r *proto.Request) {
	sf, ok := mm.stateHandlers[r.Command()]
	if ok {
		sf(c.rw, c, r)
		return
	}

	f, ok := mm.handlers[r.Command()]
	if !ok {
		// NO OP for commands that do not exist
		c.rw.WriteMessage(proto.MessageErrorCommandNotFound)
		return
	}
	f(c.rw, r)
}

func (mm *MapMux) Handle(s string, f MessageHandler) {
	mm.handlers[s] = f
}

func (mm *MapMux) HandleState(s string, f MessageStateHandler) {
	mm.stateHandlers[s] = f
}

type MessageServer struct {
	log          zerolog.Logger
	metricsStore MetricsStore
}

func NewMessageServer(log zerolog.Logger, metricsStore MetricsStore) MessageServer {
	return MessageServer{
		log,
		metricsStore,
	}
}

func (ms *MessageServer) ListenAndServe(port int, mux MessageMux) error {
	sock, err := net.ListenTCP("tcp4", &net.TCPAddr{Port: port})
	if err != nil {
		ms.log.Error().Err(err).Int("port", port).Msg("unable to listen on port")
		return nil
	}
	ms.log.Info().Int("port", port).Msg("listening...")

	for {
		conn, err := sock.AcceptTCP()
		if err != nil {
			ms.log.Error().Err(err).Msg("unable to accept connection on collection socket")
		}

		c := newConn(ms.log, mux)
		go c.Handle(conn)
		ms.metricsStore.IncClientConnection()
	}
}

type conn struct {
	log zerolog.Logger
	c   *net.TCPConn
	rw  proto.ResponseWriter

	mux MessageMux

	// state
	dbName string
	db     *database.Database
}

func newConn(log zerolog.Logger, mux MessageMux) *conn {
	return &conn{
		log: log,
		mux: mux,
	}
}

func (c *conn) SetDatabase(name string, db *database.Database) {
	c.dbName = name
	c.db = db
}

func (c conn) DatabaseName() string {
	return c.dbName
}

func (c *conn) Handle(conn *net.TCPConn) {
	c.c = conn
	defer c.c.Close()

	c.rw = proto.NewResponseWriter(c.c)

	for {
		msg, err := proto.ReadMessageFull(c.c)
		if err == io.EOF {
			c.log.Info().Msg("client disconnected")
			return
		} else if err != nil {
			c.rw.WriteMessage(proto.MessageErrorMalformedMessage)
			c.log.Error().Err(err).Msg("error parsing message from []bytes")
			continue
		}
		c.log.Trace().Object("msg", msg).Msg("parsed message")
		go c.mux.ServeMessage(c, proto.NewRequest(msg, c.db))
	}
}
