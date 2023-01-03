/*
 * Copyright (c) 2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package fossil

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/dburkart/fossil/pkg/schema"
	"math"
	"strconv"
	"strings"
)

type SchemaType interface {
	[]byte | bool | string | int16 | int32 | int64 | uint16 |
		uint32 | uint64 | float32 | float64
}

func EncodeType[T SchemaType](v T) ([]byte, error) {
	var formatted []byte

	switch t := any(v).(type) {
	case []byte:
		return t, nil
	case bool:
		var b uint8
		b = 0
		if t {
			b = 1
		}
		formatted = append(formatted, b)
		return formatted, nil
	case string:
		return []byte(t), nil
	case int16:
		return binary.LittleEndian.AppendUint16(formatted, uint16(t)), nil
	case int32:
		return binary.LittleEndian.AppendUint32(formatted, uint32(t)), nil
	case int64:
		return binary.LittleEndian.AppendUint64(formatted, uint64(t)), nil
	case uint16:
		return binary.LittleEndian.AppendUint16(formatted, t), nil
	case uint32:
		return binary.LittleEndian.AppendUint32(formatted, t), nil
	case uint64:
		return binary.LittleEndian.AppendUint64(formatted, t), nil
	case float32:
		return binary.LittleEndian.AppendUint32(formatted, math.Float32bits(t)), nil
	case float64:
		return binary.LittleEndian.AppendUint64(formatted, math.Float64bits(t)), nil
	}

	panic("We should not get here")
}

func DecodeStringForSchema(input []byte, s schema.Object) (string, error) {
	switch t := s.(type) {
	case *schema.Type:
		switch t.Name {
		case "string":
			return string(input), nil
		case "binary":
			return fmt.Sprintf("...%d bytes...", len(input)), nil
		case "boolean":
			if input[0] == 0 {
				return "false", nil
			}
			return "true", nil
		case "uint8":
			return fmt.Sprintf("%d", input[0]), nil
		case "uint16":
			return fmt.Sprintf("%d", binary.LittleEndian.Uint16(input)), nil
		case "uint32":
			return fmt.Sprintf("%d", binary.LittleEndian.Uint32(input)), nil
		case "uint64":
			return fmt.Sprintf("%d", binary.LittleEndian.Uint64(input)), nil
		case "int16":
			return fmt.Sprintf("%d", int16(binary.LittleEndian.Uint16(input))), nil
		case "int32":
			return fmt.Sprintf("%d", int32(binary.LittleEndian.Uint32(input))), nil
		case "int64":
			return fmt.Sprintf("%d", int64(binary.LittleEndian.Uint64(input))), nil
		case "float32":
			return fmt.Sprintf("%f", math.Float32frombits(binary.LittleEndian.Uint32(input))), nil
		case "float64":
			return fmt.Sprintf("%f", math.Float64frombits(binary.LittleEndian.Uint64(input))), nil
		}
	case *schema.Array:
		var output string

		for i := 0; i < t.Length; i++ {
			width := t.Type.Size()
			e, err := DecodeStringForSchema(input[i*width:(i+1)*width], &t.Type)
			if err != nil {
				return "", err
			}
			output += e
			if i < t.Length-1 {
				output += ", "
			}
		}

		return output, nil
	case *schema.Composite:
		// FIXME: Implement
	}

	return "", errors.New("unknown schema")
}

// EncodeStringForSchema takes an input string and a schema.Object, and returns
// a byte slice representing that string.
func EncodeStringForSchema(input string, s schema.Object) ([]byte, error) {
	var formatted []byte

	switch t := s.(type) {
	case *schema.Type:
		switch t.Name {
		case "string":
			return []byte(input), nil
		case "boolean":
			var b uint8
			b = 1
			if input == "false" {
				b = 0
			}
			formatted = append(formatted, b)
			return formatted, nil
		case "int16":
			i, err := strconv.ParseInt(input, 10, 16)
			if err != nil {
				return nil, err
			}
			return EncodeType(int16(i))
		case "int32":
			i, err := strconv.ParseInt(input, 10, 32)
			if err != nil {
				return nil, err
			}
			return EncodeType(int32(i))
		case "int64":
			i, err := strconv.ParseInt(input, 10, 64)
			if err != nil {
				return nil, err
			}
			return EncodeType(i)
		case "uint16":
			i, err := strconv.ParseUint(input, 10, 16)
			if err != nil {
				return nil, err
			}
			return EncodeType(uint16(i))
		case "uint32":
			i, err := strconv.ParseUint(input, 10, 32)
			if err != nil {
				return nil, err
			}
			return EncodeType(uint32(i))
		case "uint64":
			i, err := strconv.ParseUint(input, 10, 64)
			if err != nil {
				return nil, err
			}
			return EncodeType(i)
		case "float32":
			f, err := strconv.ParseFloat(input, 32)
			if err != nil {
				return nil, err
			}
			return EncodeType(float32(f))
		case "float64":
			f, err := strconv.ParseFloat(input, 64)
			if err != nil {
				return nil, err
			}
			return EncodeType(f)
		}
	case *schema.Array:
		var array []string
		array = strings.Split(input, ",")
		// Basic bounds checking
		if len(array) != t.Length {
			return nil, errors.New(fmt.Sprintf("schema expects %d elements, you provided %d", t.Length, len(array)))
		}
		// For each value in the array, pack it into formatted
		for _, v := range array {
			b, err := EncodeStringForSchema(strings.Trim(v, " \t"), &t.Type)
			if err != nil {
				return nil, err
			}
			formatted = append(formatted, b...)
		}

		return formatted, nil
	case *schema.Composite:
		// FIXME: Implement
	}

	return formatted, nil
}
