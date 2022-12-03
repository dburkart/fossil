/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package proto

import (
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
