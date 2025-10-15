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

// splitTopLevel splits an input string by the provided separator while ignoring
// separators that appear within quoted strings or nested brackets / braces /
// parentheses. This allows us to safely parse literals such as composite
// members that may themselves contain comma-separated data (e.g. arrays).
func splitTopLevel(input string, sep rune) ([]string, error) {
	var (
		parts        []string
		current      strings.Builder
		inQuote      bool
		escaped      bool
		depthParen   int
		depthBracket int
		depthBrace   int
	)

	flush := func() {
		part := strings.TrimSpace(current.String())
		parts = append(parts, part)
		current.Reset()
	}

	for _, r := range input {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}

		switch r {
		case '\\':
			if inQuote {
				escaped = true
			}
		case '"':
			inQuote = !inQuote
		case '(':
			if !inQuote {
				depthParen++
			}
		case ')':
			if !inQuote {
				if depthParen == 0 {
					return nil, errors.New("unmatched closing parenthesis")
				}
				depthParen--
			}
		case '[':
			if !inQuote {
				depthBracket++
			}
		case ']':
			if !inQuote {
				if depthBracket == 0 {
					return nil, errors.New("unmatched closing bracket")
				}
				depthBracket--
			}
		case '{':
			if !inQuote {
				depthBrace++
			}
		case '}':
			if !inQuote {
				if depthBrace == 0 {
					return nil, errors.New("unmatched closing brace")
				}
				depthBrace--
			}
		case sep:
			if !inQuote && depthParen == 0 && depthBracket == 0 && depthBrace == 0 {
				flush()
				continue
			}
		}

		current.WriteRune(r)
	}

	if escaped {
		return nil, errors.New("dangling escape character in literal")
	}

	if depthParen != 0 || depthBracket != 0 || depthBrace != 0 || inQuote {
		return nil, errors.New("unterminated literal")
	}

	flush()

	return parts, nil
}

func findTopLevelColon(input string) (int, error) {
	var (
		inQuote      bool
		escaped      bool
		depthParen   int
		depthBracket int
		depthBrace   int
	)

	for idx, r := range input {
		if escaped {
			escaped = false
			continue
		}

		switch r {
		case '\\':
			if inQuote {
				escaped = true
			}
		case '"':
			inQuote = !inQuote
		case '(':
			if !inQuote {
				depthParen++
			}
		case ')':
			if !inQuote {
				if depthParen == 0 {
					return -1, errors.New("unmatched closing parenthesis")
				}
				depthParen--
			}
		case '[':
			if !inQuote {
				depthBracket++
			}
		case ']':
			if !inQuote {
				if depthBracket == 0 {
					return -1, errors.New("unmatched closing bracket")
				}
				depthBracket--
			}
		case '{':
			if !inQuote {
				depthBrace++
			}
		case '}':
			if !inQuote {
				if depthBrace == 0 {
					return -1, errors.New("unmatched closing brace")
				}
				depthBrace--
			}
		case ':':
			if !inQuote && depthParen == 0 && depthBracket == 0 && depthBrace == 0 {
				return idx, nil
			}
		}
	}

	return -1, errors.New("malformed composite literal")
}

func consumeValueForObject(input string, obj Object) (value string, rest string, err error) {
	tokens, err := splitTopLevel(input, ',')
	if err != nil {
		return "", "", err
	}

	switch tt := obj.(type) {
	case *Type:
		if len(tokens) == 0 || (len(tokens) == 1 && tokens[0] == "") {
			return "", "", errors.New("malformed composite literal")
		}
		value = tokens[0]
		restTokens := tokens[1:]
		for _, token := range restTokens {
			if token == "" {
				return "", "", errors.New("malformed composite literal")
			}
		}
		if len(restTokens) > 0 {
			rest = strings.Join(restTokens, ",")
		}
	case *Array:
		if len(tokens) < tt.Length {
			return "", "", fmt.Errorf("schema expects %d elements, you provided %d", tt.Length, len(tokens))
		}
		value = strings.Join(tokens[:tt.Length], ", ")
		restTokens := tokens[tt.Length:]
		for _, token := range restTokens {
			if token == "" {
				return "", "", errors.New("malformed composite literal")
			}
		}
		if len(restTokens) > 0 {
			rest = strings.Join(restTokens, ",")
		}
	default:
		return "", "", errors.New("unsupported composite member type")
	}

	return value, rest, nil
}

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
		array, err := splitTopLevel(input, ',')
		if err != nil {
			return nil, err
		}
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
		remainder := strings.TrimSpace(input)
		if remainder == "" {
			return nil, errors.New("malformed composite literal")
		}

		for len(remainder) > 0 {
			remainderBefore := remainder
			colonIdx, err := findTopLevelColon(remainder)
			if err != nil {
				return nil, err
			}

			rawKey := strings.TrimSpace(remainder[:colonIdx])
			key, err := strconv.Unquote(rawKey)
			if err != nil {
				key = rawKey
			}

			obj := t.SchemaForKey(key)
			if _, ok := obj.(Unknown); ok {
				return nil, fmt.Errorf("unknown key '%s' in composite literal", key)
			}

			valueStart := strings.TrimSpace(remainder[colonIdx+1:])
			valueLiteral, rest, err := consumeValueForObject(valueStart, obj)
			if err != nil {
				return nil, err
			}

			c[key] = valueLiteral
			keys = append(keys, key)

			remainder = strings.TrimSpace(rest)
			if strings.HasPrefix(remainder, ",") {
				remainder = strings.TrimSpace(remainder[1:])
			}
			if remainder == remainderBefore {
				return nil, errors.New("malformed composite literal")
			}
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
