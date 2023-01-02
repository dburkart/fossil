/*
 * Copyright (c) 2022-2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package fossil

import (
	"io"
	"math"
	"net"
	"syscall"
	"time"

	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/proto"
	"github.com/pkg/errors"
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
	client, err := NewClientPool(connstr, 1)
	if err != nil {
		return Client{}, err
	}

	return client, nil
}

// NewClientPool creates a new Client struct which holds a pool of net.Conn
// resources open to a remote fossil database. This is useful for sending large
// volumes of data to fossil.
func NewClientPool(connstr string, size uint) (Client, error) {
	var client Client
	var err error

	client.target, err = proto.ParseConnectionString(connstr)
	if err != nil {
		return Client{}, err
	}

	client.conn = make(chan net.Conn, size)

	for i := uint(0); i < size; i++ {
		c, err := net.Dial("tcp4", client.target.Address)
		if err != nil {
			return Client{}, err
		}
		_, err = connect(c, client.target.Database)
		if err != nil {
			return Client{}, err
		}
		client.conn <- c
	}

	return client, nil
}

// FIXME: Refactor this into a common Use() API
func connect(c net.Conn, dbName string) (proto.OkResponse, error) {
	// First, send a version advertisement
	versionMsg := proto.NewMessageWithType(proto.CommandVersion, proto.VersionRequest{})
	b, _ := versionMsg.Marshal()
	c.Write(b)
	m, err := proto.ReadMessageFull(c)
	if err != nil {
		return proto.OkResponse{}, errors.Wrap(err, "unable to parse server version response")
	}
	version := proto.VersionResponse{}
	err = version.Unmarshal(m.Data())
	if err != nil {
		return proto.OkResponse{}, errors.Wrap(err, "unable to unmarshal version response")
	}
	if version.Code != 200 {
		return proto.OkResponse{}, errors.New("server rejected client version")
	}
	// We don't have any version logic yet

	// Send the server use message
	useMsg := proto.NewMessageWithType(proto.CommandUse, proto.UseRequest{DbName: dbName})
	b, _ = useMsg.Marshal()
	c.Write(b)
	m, err = proto.ReadMessageFull(c)
	if err != nil {
		return proto.OkResponse{}, errors.Wrap(err, "unable to parse server use response")
	}
	ok := proto.OkResponse{}
	err = ok.Unmarshal(m.Data())
	if err != nil {
		return proto.OkResponse{}, errors.Wrap(err, "unable to unmarshal ok response")
	}

	return ok, nil
}

func (c *Client) reconnectWithBackoff() (net.Conn, error) {
	var conn net.Conn
	var err error

	// Try for a total of 6 seconds
	for i := 0; i < 3; i++ {
		delay := time.Duration(math.Exp2(float64(i)))
		time.Sleep(delay * time.Second)
		conn, err = net.Dial("tcp4", c.target.Address)

		if err == nil {
			_, err = connect(conn, c.target.Database)
			if err != nil {
				conn.Close()
				continue
			}
			break
		}
	}

	return conn, err
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
	data, err := m.Marshal()
	if err != nil {
		return nil, err
	}

	conn := <-c.conn
	defer func() {
		c.conn <- conn
	}()

retry:
	_, err = conn.Write(data)
	if err != nil {
		// Handle peer reset with reconnect logic
		if errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE) {
			conn, err = c.reconnectWithBackoff()
			if err != nil {
				return nil, err
			}
			// We use a goto here because we need to retry sending our message,
			// however, if we recursively call Send() we'll end up with a
			// duplicated net.Conn in our connection pool.
			goto retry
		} else {
			return nil, err
		}
	}

	resp, err := proto.ReadMessageFull(conn)
	if err != nil {
		if errors.Is(err, io.EOF) {
			conn, err = c.reconnectWithBackoff()
			if err != nil {
				return nil, err
			}
			// We use a goto here because we need to retry sending our message,
			// however, if we recursively call Send() we'll end up with a
			// duplicated net.Conn in our connection pool.
			goto retry
		}
		return nil, err
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
	err = ok.Unmarshal(resp.Data())
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
	err = queryResponse.Unmarshal(resp.Data())
	if err != nil {
		return nil, err
	}

	return queryResponse.Results, nil
}
