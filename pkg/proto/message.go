/*
 * Copyright (c) 2022-2023, Gideon Williams gideon@gideonw.com
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
	"github.com/dburkart/fossil/pkg/schema"
	"github.com/dustin/go-humanize"
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
	MessageErrorUnknownDb        = NewMessageWithType(CommandList, ListRequest{})

	MessageList = NewMessageWithType(CommandError, ErrResponse{Code: 505})

	lenWidth     = 4
	commandWidth = 8
)

func ReadMessageFull(r io.Reader) (Message, error) {
	msg := &lineMessage{}
	err := msg.Unmarshal(r)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

type Message interface {
	Marshal() ([]byte, error)
	Unmarshal(r io.Reader) error
	Command() string
	Data() []byte
	MarshalZerologObject(e *zerolog.Event)
}

type lineMessage struct {
	command string
	data    []byte
}

func NewMessage(cmd string, data []byte) Message {
	return &lineMessage{
		cmd,
		data,
	}
}

func NewMessageWithType(cmd string, t Marshaler) Message {
	d, err := t.Marshal()
	if err != nil {
		panic(err)
	}
	return &lineMessage{
		cmd,
		d,
	}
}

func (m lineMessage) Marshal() ([]byte, error) {
	b := make([]byte, lenWidth+commandWidth+len(m.data))
	binary.BigEndian.PutUint32(b, uint32(commandWidth+len(m.data)))
	copy(b[lenWidth:], []byte(m.command))
	copy(b[commandWidth+lenWidth:], m.data)

	return b, nil
}

func (m *lineMessage) Unmarshal(r io.Reader) error {
	lengthPrefix := make([]byte, lenWidth)
	_, err := io.ReadFull(r, lengthPrefix)
	if err != nil {
		return err
	}
	length := binary.BigEndian.Uint32(lengthPrefix)
	if length > 100*humanize.MiByte {
		return errors.New("message too large")
	}
	buf := make([]byte, length)
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return fmt.Errorf("unable to read response\n\t'%s'", string(buf))
	}
	if n < 8 {
		return errors.New("message format incorrect")
	}

	// Parse message
	m.command = strings.ToUpper(strings.Trim(string(buf[:commandWidth]), "\u0000"))
	m.data = buf[commandWidth:]

	return nil
}
func (m lineMessage) Command() string {
	return m.command
}

func (m lineMessage) Data() []byte {
	return m.data
}

func (m lineMessage) MarshalZerologObject(e *zerolog.Event) {
	e.Str("command", m.command).Bytes("data", m.data)
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

type Printable interface {
	Headers() []string
	Values() [][]string
}

type (
	VersionRequest struct {
		Version string
	}
	VersionResponse struct {
		Code    uint32 `json:"code"`
		Version string `json:"version"`
	}

	ErrResponse struct {
		Code uint32 `json:"code"`
		Err  error  `json:"error"`
	}

	OkResponse struct {
		Code    uint32 `json:"code"`
		Message string `json:"message"`
	}

	UseRequest struct {
		DbName string
	}

	ListRequest struct {
		Object string
	}

	ListResponse struct {
		ObjectList []string `json:"results"`
	}

	StatsRequest struct {
		Database string
	}

	StatsResponse struct {
		AllocHeap uint64        `json:"alloc_heap"`
		TotalMem  uint64        `json:"total_mem"`
		Uptime    time.Duration `json:"uptime"`
		Segments  int           `json:"segments"`
		Topics    int           `json:"topics"`
	}

	AppendRequest struct {
		Topic string
		Data  []byte
	}

	QueryRequest struct {
		Query string
	}

	QueryResponse struct {
		Results database.Entries `json:"results"`
	}

	CreateTopicRequest struct {
		Topic  string
		Schema string
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
	buf := bytes.NewBuffer(binary.BigEndian.AppendUint32([]byte{}, v.Code))
	_, err := buf.Write([]byte(Version))
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Unmarshal ...
func (v *VersionResponse) Unmarshal(b []byte) error {
	buf := bytes.NewBuffer(b)
	err := binary.Read(buf, binary.BigEndian, &v.Code)
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

func (v VersionResponse) Headers() []string {
	return []string{"code", "version"}
}

func (v VersionResponse) Values() [][]string {
	return [][]string{[]string{fmt.Sprintf("%d", v.Code), v.Version}}
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
	buf := bytes.NewBuffer(binary.BigEndian.AppendUint32([]byte{}, rq.Code))

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
	err := binary.Read(buf, binary.BigEndian, &rq.Code)
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

func (v ErrResponse) Headers() []string {
	return []string{"code", "error"}
}

func (v ErrResponse) Values() [][]string {
	return [][]string{[]string{fmt.Sprintf("%d", v.Code), v.Err.Error()}}
}

// OkResponse
// --------------------------

// Marshal ...
func (rq OkResponse) Marshal() ([]byte, error) {
	buf := bytes.NewBuffer(binary.BigEndian.AppendUint32([]byte{}, rq.Code))

	_, err := buf.WriteString(rq.Message)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal ...
func (rq *OkResponse) Unmarshal(b []byte) error {
	buf := bytes.NewBuffer(b)
	err := binary.Read(buf, binary.BigEndian, &rq.Code)
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

func (v OkResponse) Headers() []string {
	return []string{"code", "message"}
}

func (v OkResponse) Values() [][]string {
	return [][]string{[]string{fmt.Sprintf("%d", v.Code), v.Message}}
}

// AppendRequest
// --------------------------

// Marshal ...
func (rq AppendRequest) Marshal() ([]byte, error) {
	buf := bytes.NewBuffer(binary.BigEndian.AppendUint32([]byte{}, uint32(len(rq.Topic))))
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
	length := binary.BigEndian.Uint32(lengthPrefix)
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
	buf := bytes.NewBuffer(binary.BigEndian.AppendUint32(b, uint32(len(rq.Results))))
	for i := range rq.Results {
		ent := rq.Results[i].ToString()
		l := binary.BigEndian.AppendUint32([]byte{}, uint32(len(ent)))
		buf.Write(l)
		buf.WriteString(ent)
	}

	return buf.Bytes(), nil
}

// Unmarshal ...
func (rq *QueryResponse) Unmarshal(b []byte) error {
	var count uint32 = 0
	buf := bytes.NewBuffer(b)
	err := binary.Read(buf, binary.BigEndian, &count)
	if err != nil {
		return err
	}
	var i uint32
	for i = 0; i < count; i++ {
		var l uint32
		err := binary.Read(buf, binary.BigEndian, &l)
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

func (v QueryResponse) Headers() []string {
	return []string{"time", "topic", "schema", "data"}
}

func (v QueryResponse) Values() [][]string {
	res := [][]string{}
	for _, val := range v.Results {
		obj, err := schema.Parse(val.Schema)
		if err != nil {
			continue
		}
		str, err := schema.DecodeStringForSchema(val.Data, obj)
		if err != nil {
			continue
		}
		res = append(res, []string{
			val.Time.Format(time.RFC3339Nano),
			val.Topic,
			val.Schema,
			str,
		})
	}

	return res
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
	b := binary.BigEndian.AppendUint64([]byte{}, rq.AllocHeap)
	b = binary.BigEndian.AppendUint64(b, rq.TotalMem)
	b = binary.BigEndian.AppendUint64(b, uint64(rq.Segments))
	b = binary.BigEndian.AppendUint64(b, uint64(rq.Topics))
	buf := bytes.NewBuffer(b)
	buf.WriteString(rq.Uptime.String())
	return buf.Bytes(), nil
}

// Unmarshal ...
func (rq *StatsResponse) Unmarshal(b []byte) error {
	buf := bytes.NewBuffer(b)
	err := binary.Read(buf, binary.BigEndian, &rq.AllocHeap)
	if err != nil {
		return err
	}
	err = binary.Read(buf, binary.BigEndian, &rq.TotalMem)
	if err != nil {
		return err
	}
	var segs uint64
	err = binary.Read(buf, binary.BigEndian, &segs)
	if err != nil {
		return err
	}
	rq.Segments = int(segs)
	var topics uint64
	err = binary.Read(buf, binary.BigEndian, &topics)
	if err != nil {
		return err
	}
	rq.Topics = int(topics)
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

func (v StatsResponse) Headers() []string {
	return []string{"alloc_heap", "total_mem", "uptime", "segments", "topics"}
}

func (v StatsResponse) Values() [][]string {
	return [][]string{
		[]string{
			humanize.Bytes(v.AllocHeap),
			humanize.Bytes(v.TotalMem),
			v.Uptime.String(),
			fmt.Sprintf("%d", v.Segments),
			fmt.Sprintf("%d", v.Topics),
		},
	}
}

// ListRequest
// --------------------------

// Marshal ...
func (rq ListRequest) Marshal() ([]byte, error) {
	if rq.Object == "" {
		return []byte{}, nil
	}
	buf := bytes.NewBufferString(rq.Object)
	return buf.Bytes(), nil
}

// Unmarshal ...
func (rq *ListRequest) Unmarshal(b []byte) error {
	obj := string(b)
	obj = strings.TrimSpace(obj)
	if len(obj) == 0 {
		rq.Object = "databases"
	} else {
		rq.Object = obj
	}
	return nil
}

// ListResponse
// --------------------------

// Marshal ...
func (rq ListResponse) Marshal() ([]byte, error) {
	b := []byte{}
	buf := bytes.NewBuffer(binary.BigEndian.AppendUint32(b, uint32(len(rq.ObjectList))))
	for i := range rq.ObjectList {
		l := binary.BigEndian.AppendUint32([]byte{}, uint32(len(rq.ObjectList[i])))
		buf.Write(l)
		buf.WriteString(rq.ObjectList[i])
	}
	return buf.Bytes(), nil
}

// Unmarshal ...
func (rq *ListResponse) Unmarshal(b []byte) error {
	var count uint32 = 0
	buf := bytes.NewBuffer(b)
	err := binary.Read(buf, binary.BigEndian, &count)
	if err != nil {
		return err
	}
	rq.ObjectList = []string{}
	var i uint32
	for i = 0; i < count; i++ {
		var l uint32
		err := binary.Read(buf, binary.BigEndian, &l)
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
		rq.ObjectList = append(rq.ObjectList, string(line))
	}
	return nil
}

func (v ListResponse) Headers() []string {
	return []string{"result"}
}

func (v ListResponse) Values() [][]string {
	res := [][]string{}
	for i := range v.ObjectList {
		res = append(res, []string{v.ObjectList[i]})
	}

	return res
}

// CreateTopicRequest
//-------------------------

// Marshal ...
func (rq CreateTopicRequest) Marshal() ([]byte, error) {
	buf := bytes.NewBuffer(binary.BigEndian.AppendUint32([]byte{}, uint32(len(rq.Topic))))
	_, err := buf.Write([]byte(rq.Topic))
	if err != nil {
		return nil, err
	}
	_, err = buf.Write([]byte(rq.Schema))
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal ...
func (rq *CreateTopicRequest) Unmarshal(b []byte) error {
	buf := bytes.NewBuffer(b)
	lengthPrefix := make([]byte, lenWidth)
	n, err := io.ReadFull(buf, lengthPrefix)
	if err != nil {
		return err
	}
	length := binary.BigEndian.Uint32(lengthPrefix)
	topic := make([]byte, length)
	m, err := io.ReadFull(buf, topic)
	if err != nil {
		return err
	}
	rq.Topic = string(topic)
	rq.Schema = string(b[n+m:])
	if rq.Schema == "" {
		rq.Schema = "string"
	}
	return nil
}
