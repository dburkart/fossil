/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package query

import (
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

// MatchIdentifier returns the length of the next token, assuming it is an
// identifier.
//
// Grammar:
//
//	identifier      = 1*(ALPHA / DIGIT)
func (s *Scanner) MatchIdentifier() int {
	i := s.Pos
	r, width := utf8.DecodeRuneInString(s.Input[i:])
	size := 0

	for unicode.IsDigit(r) || unicode.IsLetter(r) {
		size += width
		i += width
		r, width = utf8.DecodeRuneInString(s.Input[i:])
	}

	return size
}

// MatchTopic returns the length of the next token, assuming it is a topic
// string.
//
// Grammar:
//
//	topic           = "/" 1*(ALPHA / DIGIT / "/")
func (s *Scanner) MatchTopic() int {
	i := s.Pos
	r, width := utf8.DecodeRuneInString(s.Input[i:])
	size := 0

	for unicode.IsDigit(r) || unicode.IsLetter(r) || r == '/' {
		size += width
		i += width
		r, width = utf8.DecodeRuneInString(s.Input[i:])
	}

	return size
}

// MatchNumber returns the length of the next token, assuming it is a
// number
//
// Grammar:
//
//	number          = 1*DIGIT
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
func (s *Scanner) Emit() Token {
	var t Token

	oldStart := s.Start

	for {
		r, width := utf8.DecodeRuneInString(s.Input[s.Pos:])
		s.Start = s.Pos
		found := true
		skip := 0

		identifierFallthrough := func() {
			t.Type = TOK_IDENTIFIER
			skip = s.MatchIdentifier()
		}

		switch {
		case r == '\n':
			t.Type = TOK_NL
			skip = width
		case unicode.IsSpace(r):
			skip = width
			found = false
		case r == '(':
			t.Type = TOK_PAREN_L
			skip = width
		case r == ')':
			t.Type = TOK_PAREN_R
			skip = width
		case r == '/':
			t.Type = TOK_TOPIC
			skip = s.MatchTopic()
		case unicode.IsDigit(r):
			t.Type = TOK_NUMBER
			skip = s.MatchNumber()
		case r == 'a':
			if strings.HasPrefix(s.Input[s.Pos:], "all") {
				t.Type = TOK_KEYWORD
				skip = len("all")
				break
			}
			identifierFallthrough()
		case r == 'i':
			if strings.HasPrefix(s.Input[s.Pos:], "in") {
				t.Type = TOK_KEYWORD
				skip = len("in")
				break
			}
			identifierFallthrough()
		case r == 's':
			if strings.HasPrefix(s.Input[s.Pos:], "sample") {
				t.Type = TOK_KEYWORD
				skip = len("sample")
				break
			}
			identifierFallthrough()
		case unicode.IsLetter(r):
			identifierFallthrough()
		}

		s.Pos = s.Start + skip
		if found {
			break
		}
	}

	t.Lexeme = s.Input[s.Start:s.Pos]
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
