/*
 * Copyright (c) 2022-2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package fossil

import (
	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/proto"
)

type Client interface {
	Open(proto.ConnectionString, uint) error
	Close() error
	Send(proto.Message) (proto.Message, error)
	Append(string, []byte) error
	Query(string) (database.Entries, error)
}

// NewClient creates a new Client struct which can be used to interact with a
// remote fossil database. The client is thread safe, but only holds one
// connection at a time. For a client pool, use NewClientPool instead.
func NewClient(connstr string) (Client, error) {
	client, err := NewClientPool(connstr, 1)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// NewClientPool creates a new Client struct which holds a pool of net.Conn
// resources open to a remote fossil database. This is useful for sending large
// volumes of data to fossil.
func NewClientPool(connstr string, size uint) (Client, error) {
	var client Client
	var err error

	target, err := proto.ParseConnectionString(connstr)
	if err != nil {
		return nil, err
	}

	if target.Local == true {
		client = &LocalClient{}
	} else {
		client = &RemoteClient{}
	}

	err = client.Open(target, size)
	if err != nil {
		return nil, err
	}

	return client, nil
}
