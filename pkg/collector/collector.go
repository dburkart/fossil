/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package collector

import (
	"bufio"
	"bytes"
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
	scanner := bufio.NewScanner(c.conn)
	for {
		scan := scanner.Scan()
		if !scan {
			if scanner.Err() != nil {
				c.log.Error().Err(scanner.Err()).Msg("error reading from the conn")
				continue
			}
			// io.EOF
			c.conn.Close()
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
	}
}
