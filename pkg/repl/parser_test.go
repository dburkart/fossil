/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package repl

import (
	"bytes"
	"github.com/dburkart/fossil/pkg/schema"
	"testing"

	"github.com/dburkart/fossil/pkg/proto"
)

func TestParseREPLCommand(t *testing.T) {
	t.Run("use", func(t *testing.T) {
		msg, err := ParseREPLCommand([]byte("use default"), map[string]schema.Object{})
		if err != nil {
			t.Fail()
		}
		if msg.Command() != proto.CommandUse {
			t.Fail()
		}
		if !bytes.Equal(msg.Data(), []byte("default")) {
			t.Fail()
		}
	})
	t.Run("append no topic", func(t *testing.T) {
		cmp := proto.NewMessageWithType(proto.CommandAppend, proto.AppendRequest{Topic: "", Data: []byte("a")})
		msg, err := ParseREPLCommand([]byte("append a"), map[string]schema.Object{})
		if err != nil {
			t.Fail()
		}
		if msg.Command() != proto.CommandAppend {
			t.Fail()
		}
		if !bytes.Equal(msg.Data(), cmp.Data()) {
			t.Fail()
		}
	})
	t.Run("append", func(t *testing.T) {
		cmp := proto.NewMessageWithType(proto.CommandAppend, proto.AppendRequest{Topic: "/", Data: []byte("a")})
		msg, err := ParseREPLCommand([]byte("append / a"), map[string]schema.Object{})
		if err != nil {
			t.Fail()
		}
		if msg.Command() != proto.CommandAppend {
			t.Fail()
		}
		if !bytes.Equal(msg.Data(), cmp.Data()) {
			t.Fail()
		}
	})
	t.Run("append missing slash", func(t *testing.T) {
		cmp := proto.NewMessageWithType(proto.CommandAppend, proto.AppendRequest{Topic: "", Data: []byte("foo/bar/baz a")})
		msg, err := ParseREPLCommand([]byte("append foo/bar/baz a"), map[string]schema.Object{})
		if err != nil {
			t.Fail()
		}
		if msg.Command() != proto.CommandAppend {
			t.Fail()
		}
		if !bytes.Equal(msg.Data(), cmp.Data()) {
			t.Fail()
		}
	})
	t.Run("append no args", func(t *testing.T) {
		_, err := ParseREPLCommand([]byte("append"), map[string]schema.Object{})
		if err == nil {
			t.Fail()
		}
	})
	t.Run("query no query", func(t *testing.T) {
		cmp := proto.NewMessageWithType(proto.CommandQuery, proto.QueryRequest{Query: ""})
		msg, err := ParseREPLCommand([]byte("query"), map[string]schema.Object{})
		if err != nil {
			t.Fail()
		}
		if msg.Command() != proto.CommandQuery {
			t.Fail()
		}
		if !bytes.Equal(msg.Data(), cmp.Data()) {
			t.Fail()
		}
	})
	t.Run("query", func(t *testing.T) {
		cmp := proto.NewMessageWithType(proto.CommandQuery, proto.QueryRequest{Query: "all"})
		msg, err := ParseREPLCommand([]byte("query all"), map[string]schema.Object{})
		if err != nil {
			t.Fail()
		}
		if msg.Command() != proto.CommandQuery {
			t.Fail()
		}
		if !bytes.Equal(msg.Data(), cmp.Data()) {
			t.Fail()
		}
	})
}
