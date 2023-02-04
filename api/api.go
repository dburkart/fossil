/*
 * Copyright (c) 2022-2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package fossil

import (
	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/proto"
	"net"
)

type Client interface {
	Close() error
	Send(proto.Message) (proto.Message, error)
	Append(string, []byte) error
	Query(string) (database.Entries, error)
}

// NewClient creates a new RemoteClient struct which can be used to interact with a
// remote fossil database. The client is thread safe, but only holds one
// connection at a time. For a client pool, use NewClientPool instead.
func NewClient(connstr string) (Client, error) {
	client, err := NewClientPool(connstr, 1)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// NewClientPool creates a new RemoteClient struct which holds a pool of net.Conn
// resources open to a remote fossil database. This is useful for sending large
// volumes of data to fossil.
func NewClientPool(connstr string, size uint) (Client, error) {
	var client RemoteClient
	var err error

	client.target, err = proto.ParseConnectionString(connstr)
	if err != nil {
		return nil, err
	}

	client.conn = make(chan net.Conn, size)

	for i := uint(0); i < size; i++ {
		c, err := net.Dial("tcp4", client.target.Address)
		if err != nil {
			return nil, err
		}
		_, err = connect(c, client.target.Database)
		if err != nil {
			return nil, err
		}
		client.conn <- c
	}

	return &client, nil
}
