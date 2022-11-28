/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package server

import (
	"bufio"
	"bytes"
	"io"
	"net"

	"github.com/dburkart/fossil/pkg/proto"
	"github.com/rs/zerolog"
)

type MessageMux interface {
	ServeMessage(w io.Writer, msg proto.Message)
	Handle(s string, f HandleMessage)
}

type HandleMessage func(io.Writer, proto.Message)

type MapMux struct {
	handlers map[string]HandleMessage
}

func NewMapMux() MessageMux {
	return &MapMux{
		handlers: make(map[string]HandleMessage),
	}
}

func (mm *MapMux) ServeMessage(w io.Writer, msg proto.Message) {
	f, ok := mm.handlers[msg.Command]
	if !ok {
		// NO OP for commands that do not exist
		return
	}
	f(w, msg)
}

func (mm *MapMux) Handle(s string, f HandleMessage) {
	mm.handlers[s] = f
}

type MessageServer struct {
	log zerolog.Logger
}

func NewMessageServer(log zerolog.Logger) MessageServer {
	return MessageServer{
		log,
	}
}

func (ms *MessageServer) ListenAndServe(port int, mux MessageMux) error {
	sock, err := net.ListenTCP("tcp4", &net.TCPAddr{Port: port})
	if err != nil {
		ms.log.Error().Err(err).Int("port", port).Msg("unable to listen on collection port")
		return nil
	}
	ms.log.Info().Int("collection-port", port).Msg("listening for metrics")

	for {
		conn, err := sock.AcceptTCP()
		if err != nil {
			ms.log.Error().Err(err).Msg("unable to accept connection on collection socket")
		}

		c := newConn(ms.log, mux)
		go c.Handle(conn)
	}
}

type conn struct {
	log zerolog.Logger
	c   *net.TCPConn

	mux MessageMux
}

func newConn(log zerolog.Logger, mux MessageMux) *conn {
	return &conn{
		log: log,
		mux: mux,
	}
}

func (c *conn) Handle(conn *net.TCPConn) {
	c.c = conn

	// connection error states
	scanner := bufio.NewScanner(c.c)
	for {
		scan := scanner.Scan()
		if !scan {
			if scanner.Err() != nil {
				c.log.Error().Err(scanner.Err()).Msg("error reading from the conn")
				continue
			}
			// io.EOF
			c.c.Close()
			return
		}

		line := scanner.Bytes()
		c.log.Info().Int("read", len(line)).Msg("read from conn")
		buf := bytes.NewBuffer(line)
		msg, err := proto.ParseMessage(buf.Bytes())
		if err != nil {
			c.log.Trace().Bytes("buf", line).Send()
			c.log.Error().Err(err).Msg("error parsing message from buffer")
			continue
		}
		c.log.Info().Object("msg", msg).Msg("parsed message")

		rBuf := new(bytes.Buffer)
		wr := bufio.NewWriter(rBuf)

		go c.mux.ServeMessage(wr, msg)
	}
}
