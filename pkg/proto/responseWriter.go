/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package proto

import (
	"encoding/binary"
	"io"
)

var ()

type ResponseWriter struct {
	io.Writer
	w io.Writer
}

// NewResponseWriter ...
func NewResponseWriter(w io.Writer) ResponseWriter {
	return ResponseWriter{
		w: w,
	}
}

func (rw ResponseWriter) Write(b []byte) (int, error) {
	return rw.w.Write(b)
}

func (rw ResponseWriter) WriteMessage(t Marshaler) (int, error) {
	b, err := t.Marshal()
	if err != nil {
		return 0, err
	}
	lengthPrefix := make([]byte, 4)
	binary.LittleEndian.PutUint32(lengthPrefix, uint32(len(b)))
	n, err := rw.w.Write(lengthPrefix)
	if err != nil {
		return n, err
	}
	m, err := rw.w.Write(b)
	return m + n, err
}
