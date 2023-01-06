/*
 * Copyright (c) 2023, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package repl

import (
	"encoding/csv"
	"encoding/json"
	"io"

	"github.com/dburkart/fossil/pkg/proto"
	"github.com/olekukonko/tablewriter"
)

type OutputWriter interface {
	Write(v proto.Printable)
}

type CSVWriter struct {
	w io.Writer
}

type TextWriter struct {
	w io.Writer
}

type JSONWriter struct {
	w io.Writer
}

func NewOutputWriter(w io.Writer, t string) OutputWriter {
	switch t {
	case "csv":
		return CSVWriter{
			w,
		}
	case "json":
		return JSONWriter{
			w,
		}
	}
	return TextWriter{
		w,
	}
}

func (w CSVWriter) Write(v proto.Printable) {
	wtr := csv.NewWriter(w.w)
	wtr.Write(v.Headers())
	wtr.WriteAll(v.Values())
}

func (w TextWriter) Write(v proto.Printable) {
	table := tablewriter.NewWriter(w.w)
	table.SetHeader(v.Headers())
	table.AppendBulk(v.Values())
	table.Render()
}

func (w JSONWriter) Write(v proto.Printable) {
	enc := json.NewEncoder(w.w)
	enc.Encode(v)
}
