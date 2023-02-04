/*
 * Copyright (c) 2022, Gideon Williams <gideon@gideonw.com>
 * Copyright (c) 2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package server

import (
	"fmt"
	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/proto"
	"github.com/dburkart/fossil/pkg/query"
)

func VersionResponse(_ proto.VersionRequest) proto.Message {
	// We don't currently reject any versions, so respond with our own version
	// announcement with an OK code.
	versionResponse := proto.VersionResponse{Code: 200}
	return proto.NewMessageWithType(proto.CommandVersion, versionResponse)
}

func AppendResponse(a proto.AppendRequest, db *database.Database) proto.Message {
	err := db.Append(a.Data, a.Topic)
	if err != nil {
		return proto.NewMessageWithType(proto.CommandError, proto.ErrResponse{Code: 503, Err: err})
	} else {
		return proto.MessageOk
	}
}

func ListResponse(l proto.ListRequest, db *database.Database, dbMap map[string]*database.Database) proto.Message {
	resp := proto.ListResponse{
		ObjectList: []string{},
	}

	if l.Object == "databases" {
		if dbMap != nil {
			for k := range dbMap {
				resp.ObjectList = append(resp.ObjectList, k)
			}
		} else {
			resp.ObjectList = []string{db.Name}
		}
	} else if l.Object == "topics" {
		for _, v := range db.TopicLookup {
			resp.ObjectList = append(resp.ObjectList, v)
		}
	} else if l.Object == "schemas" {
		// Get our string object
		str := db.SchemaLookup[0]
		for idx, v := range db.TopicLookup {
			schema := db.SchemaLookup[idx]
			if schema != str {
				resp.ObjectList = append(resp.ObjectList, fmt.Sprintf("%s %s", v, schema.ToSchema()))
			}
		}
	}

	return proto.NewMessageWithType(proto.CommandList, resp)
}

func QueryResponse(q proto.QueryRequest, db *database.Database) proto.Message {
	stmt, err := query.Prepare(db, q.Query)
	if err != nil {
		return proto.NewMessageWithType(proto.CommandError, proto.ErrResponse{Code: 504, Err: err})
	}
	result := stmt.Execute()

	resp := proto.QueryResponse{}
	resp.Results = result.Data

	return proto.NewMessageWithType(proto.CommandQuery, resp)
}

func CreateResponse(c proto.CreateTopicRequest, db *database.Database) proto.Message {
	db.AddTopic(c.Topic, c.Schema)
	return proto.MessageOk
}
