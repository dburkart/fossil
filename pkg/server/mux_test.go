/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package server

import (
	"testing"

	"github.com/dburkart/fossil/pkg/proto"
)

var resCmd string

func stub1(rw proto.ResponseWriter, c *conn, msg *proto.Request) {
	resCmd = msg.Command()
}

func stub2(rw proto.ResponseWriter, msg *proto.Request) {
	resCmd = msg.Command()
}

func unmarshalAppend(rw proto.ResponseWriter, msg *proto.Request) {
	resCmd = msg.Command()

	req := proto.AppendRequest{}
	err := req.Unmarshal(msg.Data())
	if err != nil {
		return
	}

	resCmd = req.Topic
}

func unmarshalQuery(rw proto.ResponseWriter, msg *proto.Request) {
	resCmd = msg.Command()

	req := proto.QueryRequest{}
	err := req.Unmarshal(msg.Data())
	if err != nil {
		return
	}

	resCmd = req.Query
}

func unmarshalUse(rw proto.ResponseWriter, msg *proto.Request) {
	resCmd = msg.Command()

	req := proto.UseRequest{}
	err := req.Unmarshal(msg.Data())
	if err != nil {
		return
	}

	resCmd = req.DbName
}

func BenchmarkAllMessageTypes(b *testing.B) {
	mux := NewMapMux()

	mux.HandleState(proto.CommandUse, stub1)
	mux.Handle(proto.CommandQuery, stub2)
	mux.Handle(proto.CommandAppend, stub2)
	mux.Handle(proto.CommandStats, stub2)

	tests := []*proto.Request{
		proto.NewRequest(proto.NewMessageWithType(proto.CommandUse, proto.UseRequest{DbName: "default"}), nil),
		proto.NewRequest(proto.NewMessageWithType(proto.CommandAppend, proto.AppendRequest{Topic: "/", Data: []byte("y2k")}), nil),
		proto.NewRequest(proto.NewMessageWithType(proto.CommandQuery, proto.QueryRequest{Query: "all"}), nil),
		proto.NewRequest(proto.NewMessageWithType(proto.CommandStats, proto.StatsRequest{Database: "default"}), nil),
	}

	c := &conn{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux.ServeMessage(c, tests[i%len(tests)])
	}
}

func BenchmarkAppendMessageUnmarshal(b *testing.B) {
	mux := NewMapMux()

	mux.Handle(proto.CommandAppend, unmarshalAppend)

	tests := []*proto.Request{
		proto.NewRequest(proto.NewMessageWithType(proto.CommandAppend, proto.AppendRequest{Topic: "/", Data: []byte("y2k")}), nil),
		proto.NewRequest(proto.NewMessageWithType(proto.CommandAppend, proto.AppendRequest{Topic: "/", Data: []byte("y2k")}), nil),
		proto.NewRequest(proto.NewMessageWithType(proto.CommandAppend, proto.AppendRequest{Topic: "/", Data: []byte("y2k")}), nil),
		proto.NewRequest(proto.NewMessageWithType(proto.CommandAppend, proto.AppendRequest{Topic: "/", Data: []byte("y2k")}), nil),
	}

	c := &conn{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux.ServeMessage(c, tests[i%len(tests)])
	}
}

func BenchmarkQueryMessageUnmarshal(b *testing.B) {
	mux := NewMapMux()

	mux.Handle(proto.CommandQuery, unmarshalQuery)

	tests := []*proto.Request{
		proto.NewRequest(proto.NewMessageWithType(proto.CommandQuery, proto.QueryRequest{Query: "all"}), nil),
		proto.NewRequest(proto.NewMessageWithType(proto.CommandQuery, proto.QueryRequest{Query: "all"}), nil),
		proto.NewRequest(proto.NewMessageWithType(proto.CommandQuery, proto.QueryRequest{Query: "all"}), nil),
		proto.NewRequest(proto.NewMessageWithType(proto.CommandQuery, proto.QueryRequest{Query: "all"}), nil),
	}

	c := &conn{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux.ServeMessage(c, tests[i%len(tests)])
	}
}

func BenchmarkUseMessageUnmarshal(b *testing.B) {
	mux := NewMapMux()

	mux.Handle(proto.CommandUse, unmarshalUse)

	tests := []*proto.Request{
		proto.NewRequest(proto.NewMessageWithType(proto.CommandUse, proto.UseRequest{DbName: "default"}), nil),
		proto.NewRequest(proto.NewMessageWithType(proto.CommandUse, proto.UseRequest{DbName: "default"}), nil),
		proto.NewRequest(proto.NewMessageWithType(proto.CommandUse, proto.UseRequest{DbName: "default"}), nil),
		proto.NewRequest(proto.NewMessageWithType(proto.CommandUse, proto.UseRequest{DbName: "default"}), nil),
	}

	c := &conn{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux.ServeMessage(c, tests[i%len(tests)])
	}
}
