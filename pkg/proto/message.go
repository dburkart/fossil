/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package proto

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dburkart/fossil/pkg/database"
	"github.com/rs/zerolog"
)

var (
	Version                      = "v1.0.0"
	MessageOk                    = NewMessageWithType(CommandOk, OkResponse{Code: 200, Message: "Ok"})
	MessageOkDatabaseChanged     = NewMessageWithType(CommandOk, OkResponse{Code: 201, Message: "database changed"})
	MessageError                 = NewMessageWithType(CommandError, ErrResponse{Code: 500})
	MessageErrorCommandNotFound  = NewMessageWithType(CommandError, ErrResponse{Code: 501, Err: fmt.Errorf("command not found")})
	MessageErrorMalformedMessage = NewMessageWithType(CommandError, ErrResponse{Code: 502, Err: fmt.Errorf("malformed message")})
	MessageErrorUnmarshaling     = NewMessageWithType(CommandError, ErrResponse{Code: 506, Err: fmt.Errorf("error unmarshaling")})
	MessageErrorUnknownDb        = NewMessageWithType(CommandError, ErrResponse{Code: 505})

	lenWidth     = 4
	commandWidth = 8
)

func ReadMessageFull(r io.Reader) (Message, error) {
	msg := Message{}
	err := msg.Unmarshal(r)
	if err != nil {
		return msg, err
	}
	return msg, nil
}

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

func (m Message) Marshal() ([]byte, error) {
	b := make([]byte, lenWidth+commandWidth+len(m.Data))
	binary.LittleEndian.PutUint32(b, uint32(commandWidth+len(m.Data)))
	copy(b[lenWidth:], []byte(m.Command))
	copy(b[commandWidth+lenWidth:], m.Data)

	return b, nil
}

func (m *Message) Unmarshal(r io.Reader) error {
	lengthPrefix := make([]byte, lenWidth)
	_, err := io.ReadFull(r, lengthPrefix)
	if err != nil {
		return err
	}
	length := binary.LittleEndian.Uint32(lengthPrefix)
	b := make([]byte, length)
	n, err := io.ReadFull(r, b)
	if err != nil {
		return fmt.Errorf("unable to read response\n\t'%s'", string(b))
	}
	if n <= 8 {
		return errors.New("message format incorrect")
	}

	// Parse message
	m.Command = strings.ToUpper(strings.Trim(string(b[:commandWidth]), "\u0000"))
	m.Data = b[commandWidth:]

	return nil
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
	VersionRequest struct {
		Version string
	}
	VersionResponse struct {
		Code    uint32
		Version string
	}

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

// VersionRequest
// --------------------------

// Marshal a VersionRequest. We don't actually use the specified version, and
// instead rely on the Version variable above
func (v VersionRequest) Marshal() ([]byte, error) {
	return []byte(Version), nil
}

// Unmarshal ...
func (v *VersionRequest) Unmarshal(b []byte) error {
	v.Version = string(b)

	return nil
}

// VersionResponse
// --------------------------

// Marshal a VersionResponse. As with VersionRequest, we override the version
// specified in the supplied VersionResponse.
func (v VersionResponse) Marshal() ([]byte, error) {
	buf := bytes.NewBuffer(binary.LittleEndian.AppendUint32([]byte{}, v.Code))
	_, err := buf.Write([]byte(Version))
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Unmarshal ...
func (v *VersionResponse) Unmarshal(b []byte) error {
	buf := bytes.NewBuffer(b)
	err := binary.Read(buf, binary.LittleEndian, &v.Code)
	if err != nil {
		return err
	}

	version, err := io.ReadAll(buf)
	if err != nil {
		return err
	}

	v.Version = string(version)
	return nil
}

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

	if rq.Err != nil {
		_, err := buf.Write([]byte(rq.Err.Error()))
		if err != nil {
			return nil, err
		}
	} else {
		_, err := buf.Write([]byte("error"))
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
	buf := bytes.NewBuffer(binary.LittleEndian.AppendUint32([]byte{}, rq.Code))

	_, err := buf.WriteString(rq.Message)
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
	buf := bytes.NewBuffer(binary.LittleEndian.AppendUint32([]byte{}, uint32(len(rq.Topic))))
	_, err := buf.Write([]byte(rq.Topic))
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
	lengthPrefix := make([]byte, lenWidth)
	n, err := io.ReadFull(buf, lengthPrefix)
	if err != nil {
		return err
	}
	length := binary.LittleEndian.Uint32(lengthPrefix)
	topic := make([]byte, length)
	m, err := io.ReadFull(buf, topic)
	if err != nil {
		return err
	}
	if length == 0 {
		rq.Topic = "/"
	} else {
		rq.Topic = string(topic[:length])
	}

	rq.Data = b[n+m:]

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
	up, err := io.ReadAll(buf)
	if err != nil {
		return err
	}
	d, err := time.ParseDuration(string(up))
	if err != nil {
		return err
	}
	rq.Uptime = d

	return nil
}
