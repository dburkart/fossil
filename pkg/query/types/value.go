/*
 * Copyright (c) 2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package types

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/dburkart/fossil/pkg/common/parse"
	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/query/scanner"
	"github.com/dburkart/fossil/pkg/schema"
)

type Kind int

const (
	Unknown Kind = iota

	Boolean
	String
	Int
	Float
	Tuple
	Composite
)

type Value interface {
	Kind() Kind
}

type (
	unknownVal   struct{}
	booleanVal   bool
	stringVal    string
	intVal       int64
	floatVal     float64
	tupleVal     []Value
	compositeVal map[string]Value
)

func (unknownVal) Kind() Kind   { return Unknown }
func (booleanVal) Kind() Kind   { return Boolean }
func (stringVal) Kind() Kind    { return String }
func (intVal) Kind() Kind       { return Int }
func (floatVal) Kind() Kind     { return Float }
func (tupleVal) Kind() Kind     { return Tuple }
func (compositeVal) Kind() Kind { return Composite }

func MakeUnknown() Value                     { return unknownVal{} }
func MakeBoolean(b bool) Value               { return booleanVal(b) }
func MakeString(s string) Value              { return stringVal(s) }
func MakeInt(i int64) Value                  { return intVal(i) }
func MakeFloat(f float64) Value              { return floatVal(f) }
func MakeTuple(t []Value) Value              { return tupleVal(t) }
func MakeComposite(m map[string]Value) Value { return compositeVal(m) }

func MakeFromSchemaType(b []byte, t schema.Type) Value {
	switch t.Name {
	case "uint16", "int16":
		return MakeInt(int64(binary.LittleEndian.Uint16(b)))
	case "uint32", "int32":
		return MakeInt(int64(binary.LittleEndian.Uint32(b)))
	case "uint64", "int64":
		return MakeInt(int64(binary.LittleEndian.Uint64(b)))
	case "float32":
		return MakeFloat(float64(math.Float32frombits(binary.LittleEndian.Uint32(b))))
	case "float64":
		return MakeFloat(math.Float64frombits(binary.LittleEndian.Uint64(b)))
	case "boolean":
		return MakeBoolean(b[0] != 0)
	case "string":
		return MakeString(string(b))
	default:
		panic("Unknown type!")
	}
}

func MakeFromSchemaArray(b []byte, a schema.Array) Value {
	var values []Value

	for i := 0; i < a.Length; i++ {
		values = append(values, MakeFromSchemaType(b[i*a.Type.Size():], a.Type))
	}

	return MakeTuple(values)
}

func MakeFromEntry(entry database.Entry) Value {
	object, err := schema.Parse(entry.Schema)
	if err != nil {
		panic(err)
	}

	// FIXME: Handle composite types
	switch t := object.(type) {
	case *schema.Type:
		return MakeFromSchemaType(entry.Data, *t)
	case *schema.Array:
		return MakeFromSchemaArray(entry.Data, *t)
	case *schema.Composite:
		value := map[string]Value{}

		index := 0

		// Iterate through our keys, pulling out values to store in our map
		for i, key := range t.Keys {
			// Depending on the object type, it will take up variable amounts of space
			var size int
			obj := t.Values[i]

			switch tt := obj.(type) {
			case *schema.Type:
				size = tt.Size()

				// We are variable length, so read in the length of the field
				if tt.Name == "string" || tt.Name == "binary" {
					// Read in 4 bytes specifying size, skip it by adding to index
					size = int(binary.LittleEndian.Uint32(entry.Data[index : index+4]))
					index += 4
				}

				value[key] = MakeFromSchemaType(entry.Data[index:index+size], *tt)
			case *schema.Array:
				size = tt.Size()

				if tt.Type.Name == "string" || tt.Type.Name == "binary" {
					// Read in 4 bytes specifying size, skip it by adding to index
					size = int(binary.LittleEndian.Uint32(entry.Data[index : index+4]))
					index += 4
				}

				value[key] = MakeFromSchemaArray(entry.Data[index:index+size], *tt)
			}

			index += size
		}
		return MakeComposite(value)
	}

	return MakeUnknown()
}

func EntryFromValue(v Value) (database.Entry, error) {
	var err error
	entry := database.Entry{Data: []byte{}}

	switch v := v.(type) {
	case intVal:
		entry.Data = binary.LittleEndian.AppendUint64(entry.Data, uint64(v))
		entry.Schema = "int64"
	case floatVal:
		entry.Data = binary.LittleEndian.AppendUint64(entry.Data, math.Float64bits(float64(v)))
		entry.Schema = "float64"
	case stringVal:
		entry.Data = []byte(v)
		entry.Schema = "string"
	case booleanVal:
		if v {
			entry.Data = []byte{1}
		} else {
			entry.Data = []byte{0}
		}
		entry.Schema = "boolean"
	case tupleVal:
		// First, we assert that all values have the same sub-value type.
		// We also ensure that it's a valid "array" type
		var lastType Value
		var buffer bytes.Buffer
		var t schema.Type
		var ok bool
		for _, ix := range v {
			if lastType == nil {
				lastType = ix
			}

			switch ix.(type) {
			case intVal:
				_, ok = lastType.(intVal)
				t = schema.Type{Name: "int64"}
				b, err := schema.EncodeType(IntVal(ix))
				if err != nil {
					return entry, err
				}
				buffer.Write(b)
			case floatVal:
				_, ok = lastType.(floatVal)
				t = schema.Type{Name: "float64"}
				b, err := schema.EncodeType(FloatVal(ix))
				if err != nil {
					return entry, err
				}
				buffer.Write(b)
			case booleanVal:
				_, ok = lastType.(booleanVal)
				t = schema.Type{Name: "boolean"}
				b, err := schema.EncodeType(BooleanVal(ix))
				if err != nil {
					return entry, err
				}
				buffer.Write(b)
			default:
				ok = false
			}

			if !ok {
				err = errors.New("could not convert heterogeneous tuple to array")
				break
			}

			entry.Schema = schema.Array{Type: t, Length: len(v)}.ToSchema()
			entry.Data = buffer.Bytes()
		}

		if !ok {
			break
		}
	case compositeVal:
		var buffer bytes.Buffer
		composite := schema.Composite{}
		sortedKeys := make([]string, len(v))

		i := 0
		for k := range v {
			sortedKeys[i] = k
			i++
		}

		sort.Strings(sortedKeys)

		for _, key := range sortedKeys {
			var t schema.Object
			value := v[key]

			switch value.(type) {
			case intVal:
				t = schema.Type{Name: "int64"}
				b, err := schema.EncodeType(IntVal(value))
				if err != nil {
					return entry, err
				}
				buffer.Write(b)
			case floatVal:
				t = schema.Type{Name: "float64"}
				b, err := schema.EncodeType(FloatVal(value))
				if err != nil {
					return entry, err
				}
				buffer.Write(b)
			case booleanVal:
				t = schema.Type{Name: "boolean"}
				b, err := schema.EncodeType(BooleanVal(value))
				if err != nil {
					return entry, err
				}
				buffer.Write(b)
			case stringVal:
				s := StringVal(value)
				t = schema.Type{Name: "string"}
				b, err := schema.EncodeType(s)
				if err != nil {
					return entry, err
				}
				// First, write out the string length
				buffer.Write(binary.LittleEndian.AppendUint32([]byte{}, uint32(len(s))))
				buffer.Write(b)
			}

			composite.Keys = append(composite.Keys, key)
			composite.Values = append(composite.Values, t)
		}

		entry.Schema = composite.ToSchema()
		entry.Data = buffer.Bytes()
	}
	return entry, err
}

func MakeFromToken(tok parse.Token) Value {
	switch tok.Type {
	case scanner.TOK_INTEGER:
		if x, err := strconv.ParseInt(tok.Lexeme, 0, 64); err == nil {
			return MakeInt(x)
		}
	case scanner.TOK_FLOAT:
		if x, err := strconv.ParseFloat(tok.Lexeme, 64); err == nil {
			return MakeFloat(x)
		}
	case scanner.TOK_STRING:
		if s, err := strconv.Unquote(tok.Lexeme); err == nil {
			return MakeString(s)
		}
	}

	return MakeUnknown()
}

func StringVal(v Value) string {
	switch x := v.(type) {
	case stringVal:
		return string(x)
	case intVal:
		return strconv.FormatInt(IntVal(x), 10)
	case booleanVal:
		boolStr := "true"
		if !BooleanVal(x) {
			boolStr = "false"
		}
		return boolStr
	case floatVal:
		return strconv.FormatFloat(FloatVal(x), 'e', 3, 64)
	default:
		panic("Could not convert string")
	}
}

func BooleanVal(v Value) bool {
	switch x := v.(type) {
	case booleanVal:
		return bool(x)
	case intVal:
		return x != 0
	case floatVal:
		return x != 0.0
	default:
		panic("Not a bool")
	}
}

func IntVal(v Value) int64 {
	switch x := v.(type) {
	case intVal:
		return int64(x)
	default:
		panic("Not an int")
	}
}

func FloatVal(v Value) float64 {
	switch x := v.(type) {
	case intVal:
		return float64(x)
	case floatVal:
		return float64(x)
	case booleanVal:
		if x {
			return 1.0
		} else {
			return 0.0
		}
	default:
		panic("Not a float")
	}
}

func CompositeVal(v Value) map[string]Value {
	switch x := v.(type) {
	case compositeVal:
		return x
	default:
		panic("Not a composite")
	}
}

func TupleVal(v Value) []Value {
	switch x := v.(type) {
	case tupleVal:
		return x
	default:
		panic("Not a tuple")
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

func BinaryOp(left Value, operator parse.Token, right Value) Value {
	left, right = upcast(left, right)

	switch left := left.(type) {
	case unknownVal:
		return left

	case intVal:
		right := right.(intVal)
		switch operator.Type {
		// Comparisons
		case scanner.TOK_LESS:
			return MakeBoolean(left < right)
		case scanner.TOK_LESS_EQ:
			return MakeBoolean(left <= right)
		case scanner.TOK_EQ_EQ:
			return MakeBoolean(left == right)
		case scanner.TOK_GREATER:
			return MakeBoolean(left > right)
		case scanner.TOK_GREATER_EQ:
			return MakeBoolean(left >= right)

		// Arithmetic
		case scanner.TOK_MINUS:
			return left - right
		case scanner.TOK_PLUS:
			return left + right
		case scanner.TOK_STAR:
			return left * right
		case scanner.TOK_SLASH:
			return MakeFloat(float64(left) / float64(right))
		}
	case floatVal:
		right := right.(floatVal)
		switch operator.Type {
		// Comparisons
		case scanner.TOK_LESS:
			return MakeBoolean(left < right)
		case scanner.TOK_LESS_EQ:
			return MakeBoolean(left <= right)
		case scanner.TOK_EQ_EQ:
			return MakeBoolean(left == right)
		case scanner.TOK_GREATER:
			return MakeBoolean(left > right)
		case scanner.TOK_GREATER_EQ:
			return MakeBoolean(left >= right)

		// Arithmetic
		case scanner.TOK_MINUS:
			return left - right
		case scanner.TOK_PLUS:
			return left + right
		case scanner.TOK_STAR:
			return left * right
		case scanner.TOK_SLASH:
			return left / right
		}
	}

	panic(fmt.Sprintf("Unsupported comparison %s", operator.Lexeme))
}

func complexity(v Value) int {
	switch v.(type) {
	case unknownVal:
		return 0
	case booleanVal:
		return 1
	case *stringVal:
		return 2
	case intVal:
		return 3
	case floatVal:
		return 4
	}
	panic("Unknown type")
}

func upcast(a, b Value) (Value, Value) {
	switch ca, cb := complexity(a), complexity(b); {
	case ca < cb:
		a, b = upcastInternal(a, b)
	case ca > cb:
		b, a = upcastInternal(b, a)
	}
	return a, b
}

func upcastInternal(a, b Value) (Value, Value) {
	switch b.(type) {
	case intVal:
		return a, b
	case floatVal:
		switch aa := a.(type) {
		case intVal:
			return MakeFloat(float64(aa)), b
		}
		return a, b
	}

	panic("Could not upcast")
}
