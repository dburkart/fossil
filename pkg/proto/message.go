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
	"io"
	"strings"
	"time"

	"github.com/dburkart/fossil/pkg/database"
	"github.com/rs/zerolog"
)

var (
	MessageOk                    = NewMessageWithType(CommandOk, OkResponse{Code: 200, Message: "Ok"})
	MessageOkDatabaseChanged     = NewMessageWithType(CommandOk, OkResponse{Code: 201, Message: "database changed"})
	MessageError                 = NewMessageWithType(CommandError, ErrResponse{Code: 500})
	MessageErrorCommandNotFound  = NewMessageWithType(CommandError, ErrResponse{Code: 501, Err: fmt.Errorf("command not found")})
	MessageErrorMalformedMessage = NewMessageWithType(CommandError, ErrResponse{Code: 502, Err: fmt.Errorf("malformed message")})
	MessageErrorUnmarshaling     = NewMessageWithType(CommandError, ErrResponse{Code: 506, Err: fmt.Errorf("error unmarshaling")})
	MessageErrorUnknownDb        = NewMessageWithType(CommandError, ErrResponse{Code: 505})
)

type Message struct {
	Command string
	Data    []byte
}

func NewMessage(cmd string, data []byte) Message {
	return Message{
		cmd,
		data,
	}
}

func NewMessageWithType(cmd string, t Marshaler) Message {
	d, err := t.Marshal()
	if err != nil {
		panic(err)
	}
	return Message{
		cmd,
		d,
	}
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

func (m Message) Marshal() ([]byte, error) {
	b := new(bytes.Buffer)
	b.Write([]byte(m.Command))
	b.WriteByte(' ')
	b.Write(m.Data)
	b.WriteByte('\n')
	return b.Bytes(), nil
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
	StatsRequest struct {
		Database string
	}
	StatsResponse struct {
		AllocHeap uint64
		TotalMem  uint64
		Uptime    time.Duration
		Segments  int
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
	buf := bytes.NewBuffer(binary.LittleEndian.AppendUint32([]byte{}, rq.Code))
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
	err := binary.Read(buf, binary.LittleEndian, &rq.Code)
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
	msg, err := io.ReadAll(buf)
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
	b := []byte{}
	buf := bytes.NewBuffer(binary.LittleEndian.AppendUint32(b, rq.Code))
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
	err := binary.Read(buf, binary.LittleEndian, &rq.Code)
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
	msg, err := io.ReadAll(buf)
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

	data, err := io.ReadAll(buf)
	if err != nil {
		return err
	}
	rq.Data = data

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
	b := []byte{}
	buf := bytes.NewBuffer(binary.LittleEndian.AppendUint32(b, uint32(len(rq.Results))))
	buf.WriteByte(' ')
	for i := range rq.Results {
		ent := rq.Results[i].ToString()
		l := binary.LittleEndian.AppendUint32([]byte{}, uint32(len(ent)))
		buf.Write(l)
		buf.WriteString(ent)
	}

	return buf.Bytes(), nil
}

// Unmarshal ...
func (rq *QueryResponse) Unmarshal(b []byte) error {
	var count uint32 = 0
	buf := bytes.NewBuffer(b)
	err := binary.Read(buf, binary.LittleEndian, &count)
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
	var i uint32
	for i = 0; i < count; i++ {
		var l uint32
		err := binary.Read(buf, binary.LittleEndian, &l)
		if err != nil {
			return err
		}
		line := make([]byte, l)
		n, err := buf.Read(line)
		if err != nil {
			return err
		}
		if uint32(n) != l {
			return fmt.Errorf("error entry len not the right len %d != %d", n, l)
		}
		ent, err := database.ParseEntry(string(line))
		if err != nil {
			return err
		}
		rq.Results = append(rq.Results, ent)
	}
	return nil
}

// StatsRequest
// --------------------------

// Marshal ...
func (rq StatsRequest) Marshal() ([]byte, error) {
	return []byte(rq.Database), nil
}

// Unmarshal ...
func (rq *StatsRequest) Unmarshal(b []byte) error {
	rq.Database = string(b)

	return nil
}

// StatsResponse
// --------------------------

// Marshal ...
func (rq StatsResponse) Marshal() ([]byte, error) {
	b := binary.LittleEndian.AppendUint64([]byte{}, rq.AllocHeap)
	b = binary.LittleEndian.AppendUint64(b, rq.TotalMem)
	b = binary.LittleEndian.AppendUint64(b, uint64(rq.Segments))
	buf := bytes.NewBuffer(b)
	buf.WriteString(rq.Uptime.String())
	return buf.Bytes(), nil
}

// Unmarshal ...
func (rq *StatsResponse) Unmarshal(b []byte) error {
	buf := bytes.NewBuffer(b)
	err := binary.Read(buf, binary.LittleEndian, &rq.AllocHeap)
	if err != nil {
		return err
	}
	err = binary.Read(buf, binary.LittleEndian, &rq.TotalMem)
	if err != nil {
		return err
	}
	var segs uint64
	err = binary.Read(buf, binary.LittleEndian, &segs)
	if err != nil {
		return err
	}
	rq.Segments = int(segs)
	up, err := buf.ReadBytes('\n')
	if err != nil {
		return err
	}
	dur := strings.Trim(string(up), "\n")
	d, err := time.ParseDuration(dur)
	if err != nil {
		return err
	}
	rq.Uptime = d

	return nil
}
