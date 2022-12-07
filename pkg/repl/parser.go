/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package repl

import (
	"bytes"
	"strings"

	"github.com/dburkart/fossil/pkg/proto"
)

// ParseREPLCommand parses input from the command line
//
// This function assumes there is no '\n'
func ParseREPLCommand(b []byte) proto.Message {
	// Get the command
	var msg proto.Message
	var cmd []byte
	var data []byte

	// all commands have a space after them, if not then they are command only
	// like QUIT
	ind := bytes.IndexByte(b, ' ')
	if ind == -1 || len(b) == ind {
		cmd = b
	} else {
		cmd = b[0:ind]
		data = b[ind+1:]
	}

	// Marshal message based on the command
	command := strings.ToUpper(string(cmd))
	switch command {
	case proto.CommandVersion:
		msg = proto.NewMessageWithType(proto.CommandVersion, proto.VersionRequest{})
	case proto.CommandAppend:
		req := proto.AppendRequest{}

		// check for space after topic, no space means the data starts with /
		spaceInd := bytes.IndexByte(data, ' ')
		if data[0] == '/' && spaceInd != -1 {
			req.Topic = string(data[:spaceInd])
			req.Data = data[spaceInd+1:]
		} else {
			req.Data = data[:]
		}
		msg = proto.NewMessageWithType(proto.CommandAppend, req)
	case proto.CommandUse:
		req := proto.UseRequest{}

		req.DbName = string(data)

		msg = proto.NewMessageWithType(proto.CommandUse, req)

	case proto.CommandStats:
		req := proto.StatsRequest{}

		req.Database = string(data)

		msg = proto.NewMessageWithType(proto.CommandStats, req)
	case proto.CommandQuery:
		req := proto.QueryRequest{}

		req.Query = string(data)

		msg = proto.NewMessageWithType(proto.CommandQuery, req)
	default:
		msg = proto.NewMessage(command, b)
	}

	return msg
}
