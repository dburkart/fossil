/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package proto

import (
	"bytes"
	"fmt"

	"github.com/rs/zerolog"
)

type Message struct {
	Command string
	Data    []byte
}

// ParseMessage searches the byte slice for a message terminator and parses a message from the sequence of bytes
// it will return the number of bytes consumed
func ParseMessage(b []byte) (Message, error) {
	ret := Message{}

	ind := bytes.IndexByte(b, ' ')
	if ind == -1 {
		return ret, fmt.Errorf("malformed message")
	}
	ret.Command = string(b[0:ind])
	if ind < len(b) {
		ret.Data = b[ind+1:]
	}

	return ret, nil
}

func (m Message) MarshalZerologObject(e *zerolog.Event) {
	e.Str("command", m.Command).Bytes("data", m.Data)
}
