/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package repl

import (
	"bytes"
	"errors"
	"strings"

	"github.com/dburkart/fossil/pkg/proto"
	"github.com/dburkart/fossil/pkg/schema"
)

// ParseREPLCommand parses input from the command line
//
// This function assumes there is no '\n'
func ParseREPLCommand(b []byte, schemas map[string]schema.Object) (proto.Message, error) {
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

		if len(data) == 0 {
			return nil, errors.New("malformed append request: expected data after append keyword")
		}

		// check for space after topic, no space means the data starts with /
		spaceInd := bytes.IndexByte(data, ' ')
		if data[0] == '/' && spaceInd != -1 {
			req.Topic = string(data[:spaceInd])
			s, ok := schemas[req.Topic]
			if ok {
				d, err := schema.EncodeStringForSchema(string(data[spaceInd+1:]), s)
				if err != nil {
					return nil, err
				}
				req.Data = d
			} else {
				req.Data = data[spaceInd+1:]
			}
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
	case proto.CommandList:
		req := proto.ListRequest{}

		req.Object = string(data)

		msg = proto.NewMessageWithType(proto.CommandList, req)
	case proto.CommandCreate:
		req := proto.CreateTopicRequest{}

		if !strings.HasPrefix(string(data), "topic") &&
			!strings.HasPrefix(string(data), "TOPIC") {
			return nil, errors.New("malformed create request: expected topic keyword after create")
		}

		begin := bytes.IndexByte(data, ' ') + 1
		spaceInd := bytes.IndexByte(data[begin:], ' ') + begin

		// No schema
		if spaceInd == -1 {
			req.Topic = string(data[begin:])
			req.Schema = ""
		} else {
			req.Topic = string(data[begin:spaceInd])
			req.Schema = string(data[spaceInd+1:])
		}

		msg = proto.NewMessageWithType(proto.CommandCreate, req)
	default:
		msg = proto.NewMessage(command, b)
	}

	return msg, nil
}
