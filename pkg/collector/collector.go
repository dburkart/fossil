/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */
package collector

import (
	"net"

	"github.com/dburkart/fossil/pkg/proto"
	"github.com/rs/zerolog"
)

type Collector struct {
	log zerolog.Logger

	conn   *net.TCPConn
	stream chan proto.Message
}

func New(log zerolog.Logger, c *net.TCPConn, stream chan proto.Message) Collector {
	return Collector{log, c, stream}
}

func (c *Collector) Handle() {
	for {
		reader := proto.NewMessageReader()
		n, err := reader.ReadFrom(c.conn)
		if err != nil {
			c.log.Error().Err(err).Msg("error reading from the conn")
		}

		c.log.Info().Int64("read", n).Msg("read from conn")
		for _, m := range reader.PopMessages() {
			c.log.Info().Str("command", m.Command).Msg("read message")
		}
	}
}
