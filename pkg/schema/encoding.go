/*
 * Copyright (c) 2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package schema

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

type SchemaType interface {
	[]byte | bool | string | int16 | int32 | int64 | uint16 |
		uint32 | uint64 | float32 | float64
}

// EncodeType takes a SchemaType and returns the byte slice representing the
// data in a format the database expects
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

func DecodeStringForSchema(input []byte, s Object) (string, error) {
	switch t := s.(type) {
	case *Type:
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
	case *Array:
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
	case *Composite:
		index := 0
		var pairs []string

		for i, key := range t.Keys {
			var err error
			var size int
			obj := t.Values[i]
			var repr string

			switch tt := obj.(type) {
			case *Type:
				switch tt.Name {
				case "string":
					size = int(binary.LittleEndian.Uint32(input[index : index+4]))
					index += 4
					repr = string(input[index : index+size])
				case "binary":
					size = int(binary.LittleEndian.Uint32(input[index : index+4]))
					index += 4
					repr, err = DecodeStringForSchema(input[index:index+size], obj)
					if err != nil {
						return "", err
					}
				default:
					size = tt.Size()
					repr, err = DecodeStringForSchema(input[index:index+size], obj)
					if err != nil {
						return "", err
					}
				}
			case *Array:
				size = tt.Size()
				repr, err = DecodeStringForSchema(input[index:index+size], obj)
				if err != nil {
					return "", err
				}
			}

			index += size
			pairs = append(pairs, fmt.Sprintf("%s: %s", key, repr))
		}

		return strings.Join(pairs, ", "), nil
	}

	return "", errors.New("unknown schema")
}

// EncodeStringForSchema takes an input string and a Object, and returns
// a byte slice representing that string.
func EncodeStringForSchema(input string, s Object) ([]byte, error) {
	var formatted []byte

	switch t := s.(type) {
	case *Type:
		switch t.Name {
		case "string":
			return []byte(input), nil
		case "binary":
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
	case *Array:
		var array []string
		array = strings.Split(input, ",")
		// Basic bounds checking
		if len(array) != t.Length {
			return nil, fmt.Errorf("schema expects %d elements, you provided %d", t.Length, len(array))
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
	case *Composite:
		var keys []string
		c := map[string]string{}
		pairs := strings.Split(input, ",")

		for _, p := range pairs {
			pair := strings.Split(p, ":")

			if len(pair) != 2 {
				return nil, errors.New("malformed composite literal")
			}

			s := strings.Trim(pair[0], " \t\n")
			key, err := strconv.Unquote(s)
			if err != nil {
				key = s
			}
			value := strings.Trim(pair[1], " \t\n")
			c[key] = value
			keys = append(keys, key)
		}

		sort.Strings(keys)

		for i, key := range keys {
			obj := t.Values[i]

			switch tt := obj.(type) {
			case *Type:
				switch tt.Name {
				case "string":
					formatted = append(formatted, binary.LittleEndian.AppendUint32([]byte{}, uint32(len(c[key])))...)
					b, err := EncodeStringForSchema(c[key], tt)
					if err != nil {
						return nil, err
					}
					formatted = append(formatted, b...)
				case "binary":
					formatted = append(formatted, binary.LittleEndian.AppendUint32([]byte{}, uint32(len(c[key])))...)
					b, err := EncodeStringForSchema(c[key], tt)
					if err != nil {
						return nil, err
					}
					formatted = append(formatted, b...)
				default:
					b, err := EncodeStringForSchema(c[key], tt)
					if err != nil {
						return nil, err
					}
					formatted = append(formatted, b...)
				}

			case *Array:
				b, err := EncodeStringForSchema(c[key], tt)
				if err != nil {
					return nil, err
				}
				formatted = append(formatted, b...)
			}
		}
	}

	return formatted, nil
}
