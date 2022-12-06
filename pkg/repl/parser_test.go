/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package repl

import (
	"bytes"
	"testing"

	"github.com/dburkart/fossil/pkg/proto"
)

func TestParseREPLCommand(t *testing.T) {
	t.Run("use", func(t *testing.T) {
		msg := ParseREPLCommand([]byte("use default"))
		if msg.Command != proto.CommandUse {
			t.Fail()
		}
		if !bytes.Equal(msg.Data, []byte("default")) {
			t.Fail()
		}
	})
	t.Run("append no topic", func(t *testing.T) {
		cmp := proto.NewMessageWithType(proto.CommandAppend, proto.AppendRequest{Topic: "", Data: []byte("a")})
		msg := ParseREPLCommand([]byte("append a"))
		if msg.Command != proto.CommandAppend {
			t.Fail()
		}
		if !bytes.Equal(msg.Data, cmp.Data) {
			t.Fail()
		}
	})
	t.Run("append", func(t *testing.T) {
		cmp := proto.NewMessageWithType(proto.CommandAppend, proto.AppendRequest{Topic: "/", Data: []byte("a")})
		msg := ParseREPLCommand([]byte("append / a"))
		if msg.Command != proto.CommandAppend {
			t.Fail()
		}
		if !bytes.Equal(msg.Data, cmp.Data) {
			t.Fail()
		}
	})
	t.Run("append missing slash", func(t *testing.T) {
		cmp := proto.NewMessageWithType(proto.CommandAppend, proto.AppendRequest{Topic: "", Data: []byte("foo/bar/baz a")})
		msg := ParseREPLCommand([]byte("append foo/bar/baz a"))
		if msg.Command != proto.CommandAppend {
			t.Fail()
		}
		if !bytes.Equal(msg.Data, cmp.Data) {
			t.Fail()
		}
	})
	t.Run("query no query", func(t *testing.T) {
		cmp := proto.NewMessageWithType(proto.CommandQuery, proto.QueryRequest{Query: ""})
		msg := ParseREPLCommand([]byte("query"))
		if msg.Command != proto.CommandQuery {
			t.Fail()
		}
		if !bytes.Equal(msg.Data, cmp.Data) {
			t.Fail()
		}
	})
	t.Run("query", func(t *testing.T) {
		cmp := proto.NewMessageWithType(proto.CommandQuery, proto.QueryRequest{Query: "all"})
		msg := ParseREPLCommand([]byte("query all"))
		if msg.Command != proto.CommandQuery {
			t.Fail()
		}
		if !bytes.Equal(msg.Data, cmp.Data) {
			t.Fail()
		}
	})
}
