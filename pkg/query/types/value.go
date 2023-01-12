/*
 * Copyright (c) 2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package types

import (
	"encoding/binary"
	"fmt"
	"github.com/dburkart/fossil/pkg/common/parse"
	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/query/scanner"
	"math"
	"strconv"
)

type Kind int

const (
	Unknown Kind = iota

	Boolean
	String
	Int
	Float
)

type Value interface {
	Kind() Kind
}

type (
	unknownVal struct{}
	booleanVal bool
	stringVal  string
	intVal     int64
	floatVal   float64
)

func (unknownVal) Kind() Kind { return Unknown }
func (booleanVal) Kind() Kind { return Boolean }
func (stringVal) Kind() Kind  { return String }
func (intVal) Kind() Kind     { return Int }
func (floatVal) Kind() Kind   { return Float }

func MakeUnknown() Value        { return unknownVal{} }
func MakeBoolean(b bool) Value  { return booleanVal(b) }
func MakeString(s string) Value { return stringVal(s) }
func MakeInt(i int64) Value     { return intVal(i) }
func MakeFloat(f float64) Value { return floatVal(f) }

func MakeFromEntry(entry database.Entry) Value {
	// FIXME: Handle more than just Types here
	switch entry.Schema {
	case "uint16", "int16":
		return MakeInt(int64(binary.LittleEndian.Uint16(entry.Data)))
	case "uint32", "int32":
		return MakeInt(int64(binary.LittleEndian.Uint32(entry.Data)))
	case "uint64", "int64":
		return MakeInt(int64(binary.LittleEndian.Uint64(entry.Data)))
	case "float32":
		return MakeFloat(float64(math.Float32frombits(binary.LittleEndian.Uint32(entry.Data))))
	case "float64":
		return MakeFloat(math.Float64frombits(binary.LittleEndian.Uint64(entry.Data)))
	case "boolean":
		return MakeBoolean(entry.Data[0] != 0)
	case "string":
		return MakeString(string(entry.Data))
	}

	return MakeUnknown()
}

func MakeFromToken(tok parse.Token) Value {
	switch tok.Type {
	case scanner.TOK_NUMBER:
		if x, err := strconv.ParseInt(tok.Lexeme, 0, 64); err == nil {
			return MakeInt(x)
		}
	case scanner.TOK_STRING:
		if s, err := strconv.Unquote(tok.Lexeme); err == nil {
			return MakeString(s)
		}
	}

	return MakeUnknown()
}

func IntVal(v Value) int64 {
	switch x := v.(type) {
	case intVal:
		return int64(x)
	default:
		panic("Not an int")
	}
}

func UnaryOp(operator parse.Token, operand Value) Value {
	switch operator.Type {
	case scanner.TOK_MINUS:
		switch operand := operand.(type) {
		case intVal:
			return MakeInt(-int64(operand))
		case floatVal:
			return MakeFloat(-float64(operand))
		default:
			return MakeUnknown()
		}
	case scanner.TOK_PLUS:
		switch operand := operand.(type) {
		case intVal, floatVal:
			return operand
		default:
			return MakeUnknown()
		}
	default:
		panic(fmt.Sprintf("Unknown operator %s", operator.Lexeme))
	}
}
