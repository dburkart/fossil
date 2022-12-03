/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/dburkart/fossil/pkg/database"
	"github.com/rs/zerolog"
)

var (
	MessageOk                = OkResponse{Code: 200, Message: "Ok"}
	MessageOkDatabaseChanged = OkResponse{Code: 201, Message: "database changed"}
	MessageError             = ErrResponse{Code: 500}
	MessageErrorUnmarshaling = ErrResponse{Code: 506}
	MessageErrorUnknownDb    = ErrResponse{Code: 505}
)

type Message struct {
	Command string
	Data    []byte
}

// ParseMessage searches the byte slice for a message terminator and parses a message from the sequence of bytes
// it will return the number of bytes consumed
func ParseMessage(b []byte) (Message, error) {
	ret := Message{}

	ind := bytes.IndexByte(b, ' ')
	if ind == -1 {
		return ret, fmt.Errorf("malformed message")
	}
	ret.Command = strings.ToUpper(string(b[0:ind]))
	if ind < len(b) {
		ret.Data = b[ind+1:]
	}

	return ret, nil
}

func (m Message) MarshalZerologObject(e *zerolog.Event) {
	e.Str("command", m.Command).Bytes("data", m.Data)
}

func Marshal(t Marshaler) ([]byte, error) {
	return t.Marshal()
}

func Unmarshal(b []byte, t Unmarshaler) error {
	return t.Unmarshal(b)
}

type Marshaler interface {
	Marshal() ([]byte, error)
}

type Unmarshaler interface {
	Unmarshal([]byte) error
}

// var (
// 	_ WireMessage = QueryRequest{}
// 	_ WireMessage = QueryResponse{}
// )

type (
	ErrResponse struct {
		Code uint32
		Err  error
	}

	OkResponse struct {
		Code    uint32
		Message string
	}

	UseRequest struct {
		DbName string
	}

	AppendRequest struct {
		Topic string
		Data  []byte
	}

	QueryRequest struct {
		Query string
	}

	QueryResponse struct {
		Results database.Entries
	}
)

// UseRequest
// --------------------------

// Marshal ...
func (rq UseRequest) Marshal() ([]byte, error) {
	return []byte(rq.DbName), nil
}

// Unmarshal ...
func (rq *UseRequest) Unmarshal(b []byte) error {
	rq.DbName = string(b)

	return nil
}

// ErrResponse
// --------------------------

// Marshal ...
func (rq ErrResponse) Marshal() ([]byte, error) {
	b := []byte{}
	binary.LittleEndian.AppendUint32(b, rq.Code)
	buf := bytes.NewBuffer(b)
	err := buf.WriteByte(' ')
	if err != nil {
		return nil, err
	}
	if rq.Err != nil {
		_, err = buf.Write([]byte(rq.Err.Error()))
		if err != nil {
			return nil, err
		}
	} else {
		_, err = buf.Write([]byte("error"))
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// Unmarshal ...
func (rq *ErrResponse) Unmarshal(b []byte) error {
	buf := bytes.NewBuffer(b)
	err := binary.Read(buf, binary.LittleEndian, rq.Code)
	if err != nil {
		return err
	}
	space, err := buf.ReadByte()
	if err != nil {
		return err
	}
	if space != ' ' {
		return fmt.Errorf("expected space, got '%b'", space)
	}
	msg := []byte{}
	_, err = buf.Read(msg)
	if err != nil {
		return err
	}
	rq.Err = fmt.Errorf(string(msg))

	return nil
}

// OkResponse
// --------------------------

// Marshal ...
func (rq OkResponse) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	binary.LittleEndian.AppendUint32(buf.Bytes(), rq.Code)
	err := buf.WriteByte(' ')
	if err != nil {
		return nil, err
	}
	_, err = buf.Write([]byte(rq.Message))
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal ...
func (rq *OkResponse) Unmarshal(b []byte) error {
	buf := bytes.NewBuffer(b)
	err := binary.Read(buf, binary.LittleEndian, rq.Code)
	if err != nil {
		return err
	}
	space, err := buf.ReadByte()
	if err != nil {
		return err
	}
	if space != ' ' {
		return fmt.Errorf("expected space, got '%b'", space)
	}
	msg := []byte{}
	_, err = buf.Read(msg)
	if err != nil {
		return err
	}
	rq.Message = string(msg)

	return nil
}

// AppendRequest
// --------------------------

// Marshal ...
func (rq AppendRequest) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	_, err := buf.Write([]byte(rq.Topic))
	if err != nil {
		return nil, err
	}
	err = buf.WriteByte(' ')
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(rq.Data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal ...
func (rq *AppendRequest) Unmarshal(b []byte) error {
	buf := bytes.NewBuffer(b)
	topic, err := buf.ReadBytes(' ')
	if err != nil {
		return err
	}
	rq.Topic = string(topic[:len(topic)-1])

	_, err = buf.Read(rq.Data)
	if err != nil {
		return err
	}

	return nil
}

// QueryRequest
// --------------------------

// Marshal ...
func (rq QueryRequest) Marshal() ([]byte, error) {
	return []byte(rq.Query), nil
}

// Unmarshal ...
func (rq *QueryRequest) Unmarshal(b []byte) error {
	rq.Query = string(b)
	return nil
}

// QueryResponse
// --------------------------

// Marshal ...
func (rq QueryResponse) Marshal() ([]byte, error) {

	return nil, nil
}

// Unmarshal ...
func (rq *QueryResponse) Unmarshal(b []byte) error {

	return nil
}
