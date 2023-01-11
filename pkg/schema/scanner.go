/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package schema

import (
	"github.com/dburkart/fossil/pkg/common/parse"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Scanner struct {
	Input     string
	Start     int
	Pos       int
	RuneWidth int
	LastWidth int
}

// MatchKey returns the length of the schema key
//
// Grammar:
//
//	key = DQUOTE 1*( ALPHA / DIGIT / "_" / "-" ) DQUOTE
func (s *Scanner) MatchKey() int {
	pos := s.Pos
	r, width := utf8.DecodeRuneInString(s.Input[pos:])

	if r != '"' && r != '\'' {
		return 0
	}

	matchRune := r

	pos += width
	r, width = utf8.DecodeRuneInString(s.Input[pos:])

	for r != matchRune {
		if pos >= len(s.Input) {
			return 0
		}

		if !unicode.IsLetter(r) && !unicode.IsDigit(r) &&
			r != '_' && r != '-' {
			return 0
		}

		pos += width
		r, width = utf8.DecodeRuneInString(s.Input[pos:])
	}

	pos += width

	return pos - s.Pos
}

func (s *Scanner) MatchNumber() int {
	r, width := utf8.DecodeRuneInString(s.Input[s.Pos:])
	size := 0

	for i := s.Pos; unicode.IsDigit(r); {
		size += width
		i += width
		r, width = utf8.DecodeRuneInString(s.Input[i:])
	}

	return size
}

// Emit the next Token found on Scanner.Input
func (s *Scanner) Emit() parse.Token {
	var t parse.Token

	oldStart := s.Start

	for {
		r, width := utf8.DecodeRuneInString(s.Input[s.Pos:])
		s.Start = s.Pos
		found := true
		skip := 0

		switch {
		case unicode.IsSpace(r):
			skip = width
			found = false
		case r == '{':
			t.Type = TOK_CURLY_O
			skip = width
		case r == '}':
			t.Type = TOK_CURLY_X
			skip = width
		case r == '[':
			t.Type = TOK_BRACKET_O
			skip = width
		case r == ']':
			t.Type = TOK_BRACKET_X
			skip = width
		case r == ':':
			t.Type = TOK_COLON
			skip = width
		case r == ',':
			t.Type = TOK_COMMA
			skip = width
		case unicode.IsDigit(r):
			skip = s.MatchNumber()
			if skip == 0 {
				t.Type = TOK_INVALID
				skip = s.SkipToBoundary(isDelimiter)
			}
			t.Type = TOK_NUMBER
		case r == '"' || r == '\'':
			skip = s.MatchKey()
			if skip == 0 {
				t.Type = TOK_INVALID
				skip = s.SkipToBoundary(isDelimiter)
			}
			t.Type = TOK_KEY
		case r == 'b':
			if strings.HasPrefix(s.Input[s.Pos:], "binary") {
				t.Type = TOK_TYPE
				skip = len("binary")
				break
			}
			if strings.HasPrefix(s.Input[s.Pos:], "boolean") {
				t.Type = TOK_TYPE
				skip = len("boolean")
				break
			}
			t.Type = TOK_INVALID
			skip = s.SkipToBoundary(isDelimiter)
		case r == 'f':
			if strings.HasPrefix(s.Input[s.Pos:], "float") {
				if strings.HasPrefix(s.Input[s.Pos+5:], "32") ||
					strings.HasPrefix(s.Input[s.Pos+5:], "64") {
					t.Type = TOK_TYPE
					skip = len("float32")
					break
				}
			}
			t.Type = TOK_INVALID
			skip = s.SkipToBoundary(isDelimiter)
		case r == 'i':
			if strings.HasPrefix(s.Input[s.Pos:], "int") {
				if strings.HasPrefix(s.Input[s.Pos+3:], "8") {
					t.Type = TOK_TYPE
					skip = len("int8")
					break
				}

				if strings.HasPrefix(s.Input[s.Pos+3:], "16") ||
					strings.HasPrefix(s.Input[s.Pos+3:], "32") ||
					strings.HasPrefix(s.Input[s.Pos+3:], "64") {
					t.Type = TOK_TYPE
					skip = len("int16")
					break
				}
			}
			t.Type = TOK_INVALID
			skip = s.SkipToBoundary(isDelimiter)
		case r == 's':
			if strings.HasPrefix(s.Input[s.Pos:], "string") {
				t.Type = TOK_TYPE
				skip = len("string")
				break
			}
			t.Type = TOK_INVALID
			skip = s.SkipToBoundary(isDelimiter)
		case r == 'u':
			if strings.HasPrefix(s.Input[s.Pos:], "uint") {
				if strings.HasPrefix(s.Input[s.Pos+4:], "8") {
					t.Type = TOK_TYPE
					skip = len("uint8")
					break
				}

				if strings.HasPrefix(s.Input[s.Pos+4:], "16") ||
					strings.HasPrefix(s.Input[s.Pos+4:], "32") ||
					strings.HasPrefix(s.Input[s.Pos+4:], "64") {
					t.Type = TOK_TYPE
					skip = len("uint16")
					break
				}
			}
		}

		s.Pos = s.Start + skip
		if found {
			break
		}
	}

	t.Lexeme = s.Input[s.Start:s.Pos]
	t.Location = parse.Location{Start: s.Start, End: s.Pos}
	s.Start = s.Pos

	s.LastWidth = s.Start - oldStart

	return t
}

// Rewind the last read token
func (s *Scanner) Rewind() {
	s.Start -= s.LastWidth
	s.Pos = s.Start
	s.LastWidth = 0
}

type boundaryFunc func(rune) bool

func isDelimiter(r rune) bool {
	return unicode.IsSpace(r) || r == ':' || r == ',' || r == '"' || r == '}'
}

// SkipToBoundary returns the number of bytes until the next delimiter.
// This is useful for skipping over invalid tokens.
func (s *Scanner) SkipToBoundary(boundary boundaryFunc) int {
	r, width := utf8.DecodeRuneInString(s.Input[s.Pos:])
	size := 0

	for !boundary(r) && s.Pos+size < len(s.Input) {
		size += width
		r, width = utf8.DecodeRuneInString(s.Input[s.Pos+size:])
	}

	return size
}
