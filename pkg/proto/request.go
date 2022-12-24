/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package proto

import (
	"github.com/dburkart/fossil/pkg/database"
)

type Request struct {
	msg Message
	db  *database.Database
}

// NewRequest creates a new request from the line message and the current
// client state
func NewRequest(msg Message, db *database.Database) *Request {
	return &Request{
		msg: msg,
		db:  db,
	}
}

// Database retrieves the current database handle
func (r *Request) Database() *database.Database {
	return r.db
}

// Command retrieves the command from the request
func (r *Request) Command() string {
	return r.msg.Command()
}

// Data retrieves the data portion of the line message
func (r *Request) Data() []byte {
	return r.msg.Data()
}
