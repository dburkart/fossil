/*
 * Copyright (c) 2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package fossil

import (
	"errors"
	"fmt"
	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/proto"
	"github.com/dburkart/fossil/pkg/server"
)

type LocalClient struct {
	target proto.ConnectionString
	db     *database.Database
}

func (client *LocalClient) Open(target proto.ConnectionString, _ uint) error {
	var err error

	client.target = target
	client.db, err = database.NewDatabase(target.Address, target.Database)
	if err != nil {
		return err
	}

	return nil
}

func (client *LocalClient) Close() error {
	return nil
}

func (client *LocalClient) Send(message proto.Message) (proto.Message, error) {
	switch message.Command() {
	case proto.CommandVersion:
		var versionReq proto.VersionRequest
		err := proto.Unmarshal(message.Data(), &versionReq)
		if err != nil {
			return proto.MessageErrorUnmarshaling, nil
		}
		return server.VersionResponse(versionReq), nil
	case proto.CommandList:
		var listReq proto.ListRequest
		err := proto.Unmarshal(message.Data(), &listReq)
		if err != nil {
			return proto.MessageErrorUnmarshaling, nil
		}
		return server.ListResponse(listReq, client.db, nil), nil
	case proto.CommandAppend:
		var appendReq proto.AppendRequest
		err := proto.Unmarshal(message.Data(), &appendReq)
		if err != nil {
			return proto.MessageErrorUnmarshaling, nil
		}
		return server.AppendResponse(appendReq, client.db), nil
	case proto.CommandQuery:
		var queryReq proto.QueryRequest
		err := proto.Unmarshal(message.Data(), &queryReq)
		if err != nil {
			return proto.MessageErrorUnmarshaling, nil
		}
		return server.QueryResponse(queryReq, client.db), nil
	case proto.CommandCreate:
		var createReq proto.CreateTopicRequest
		err := proto.Unmarshal(message.Data(), &createReq)
		if err != nil {
			return proto.MessageErrorUnmarshaling, nil
		}
		return server.CreateResponse(createReq, client.db), nil
	case proto.CommandStats:
		return proto.NewMessageWithType(
			proto.CommandError,
			proto.ErrResponse{Code: 404, Err: errors.New("Stats request not supported in local mode")},
		), nil
	case proto.CommandUse:
		return proto.NewMessageWithType(
			proto.CommandError,
			proto.ErrResponse{Code: 404, Err: errors.New("Use request not supported in local mode")},
		), nil
	default:
		return proto.NewMessageWithType(
			proto.CommandError,
			proto.ErrResponse{Code: 404, Err: errors.New(fmt.Sprintf("Unknown command: %s", message.Command()))},
		), nil
	}
}

func (client *LocalClient) Append(topic string, data []byte) error {
	appendMsg := proto.NewMessageWithType(proto.CommandAppend,
		proto.AppendRequest{
			Topic: topic,
			Data:  data,
		})

	resp, err := client.Send(appendMsg)
	if err != nil {
		return err
	}

	ok := proto.OkResponse{}
	err = ok.Unmarshal(resp.Data())
	if err != nil {
		return err
	}

	return nil
}

func (client *LocalClient) Query(q string) (database.Entries, error) {
	queryMsg := proto.NewMessageWithType(proto.CommandQuery,
		proto.QueryRequest{
			Query: q,
		})

	resp, err := client.Send(queryMsg)
	if err != nil {
		return nil, err
	}

	queryResponse := proto.QueryResponse{}
	err = queryResponse.Unmarshal(resp.Data())
	if err != nil {
		return nil, err
	}

	return queryResponse.Results, nil
}
