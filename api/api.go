/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package fossil

import (
	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/proto"
	"github.com/pkg/errors"
	"net"
)

// A Client holds the data needed to interact with a fossil database.
type Client struct {
	target proto.ConnectionString
	conn   chan net.Conn
}

// NewClient creates a new Client struct which can be used to interact with a
// remote fossil database. The client is thread safe, but only holds one
// connection at a time. For a client pool, use NewClientPool instead.
func NewClient(connstr string) (Client, error) {
	var client Client

	client.target = proto.ParseConnectionString(connstr)
	c, err := net.Dial("tcp4", client.target.Address)
	if err != nil {
		return Client{}, err
	}
	// Before we do anything else, send over a "USE" command to switch DBs
	_, err = connect(c, client.target.Database)
	if err != nil {
		return Client{}, err
	}
	client.conn = make(chan net.Conn, 1)
	client.conn <- c

	return client, nil
}

// FIXME: Refactor this into a common Use() API
func connect(c net.Conn, dbName string) (proto.OkResponse, error) {
	// Always send a use first
	useMsg := proto.NewMessageWithType(proto.CommandUse, proto.UseRequest{DbName: dbName})
	b, _ := useMsg.Marshal()
	c.Write(b)
	m, err := proto.ReadMessageFull(c)
	if err != nil {
		return proto.OkResponse{}, errors.Wrap(err, "unable to parse server use response")
	}
	ok := proto.OkResponse{}
	err = ok.Unmarshal(m.Data)
	if err != nil {
		return proto.OkResponse{}, errors.Wrap(err, "unable to unmarshal ok response")
	}

	return ok, nil
}

func (c *Client) Close() error {
	for i := 0; i < len(c.conn); i++ {
		conn := <-c.conn
		err := conn.Close()
		if err != nil {
			return err
		}
	}
	c.conn = nil
	return nil
}

// Send a general message to the fossil server.
func (c *Client) Send(m proto.Message) (proto.Message, error) {
	conn := <-c.conn
	defer func() {
		c.conn <- conn
	}()

	data, err := m.Marshal()
	if err != nil {
		return proto.Message{}, err
	}

	_, err = conn.Write(data)
	if err != nil {
		return proto.Message{}, err
	}

	resp, err := proto.ReadMessageFull(conn)
	if err != nil {
		return proto.Message{}, err
	}
	return resp, nil
}

// Append data to the specified topic.
func (c *Client) Append(topic string, data []byte) error {
	appendMsg := proto.NewMessageWithType(proto.CommandAppend,
		proto.AppendRequest{
			Topic: topic,
			Data:  data,
		})

	resp, err := c.Send(appendMsg)
	if err != nil {
		return err
	}

	ok := proto.OkResponse{}
	err = ok.Unmarshal(resp.Data)
	if err != nil {
		return err
	}

	return nil
}

// Query the database for some time-series data.
func (c *Client) Query(q string) (database.Entries, error) {
	queryMsg := proto.NewMessageWithType(proto.CommandQuery,
		proto.QueryRequest{
			Query: q,
		})

	resp, err := c.Send(queryMsg)
	if err != nil {
		return nil, err
	}

	queryResponse := proto.QueryResponse{}
	err = queryResponse.Unmarshal(resp.Data)
	if err != nil {
		return nil, err
	}

	return queryResponse.Results, nil
}
