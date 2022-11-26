/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package lang

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
}

func (s *Scanner) MatchNumber() int {
	i := s.Pos
	r, width := utf8.DecodeRuneInString(s.Input[i:])
	size := 0

	for i := s.Pos; r >= '0' && r <= '9'; {
		size += width
		i += width
		r, width = utf8.DecodeRuneInString(s.Input[i:])
	}

	return size
}

func (s *Scanner) Emit() Token {
	var t Token

	for {
		r, width := utf8.DecodeRuneInString(s.Input[s.Pos:])
		s.Start = s.Pos
		found := true
		skip := 0

		switch {
		case r == '\n':
			t.Type = TOK_NL
			skip = width
		case unicode.IsSpace(r):
			skip = width
			found = false
		case r >= '0' && r <= '9':
			t.Type = TOK_NUMBER
			skip = s.MatchNumber()
		case r == 'a':
			if strings.HasPrefix(s.Input[s.Pos:], "all") {
				t.Type = TOK_KEYWORD
				skip = len("all")
				break
			}
		}

		s.Pos = s.Start + skip
		if found {
			break
		}
	}

	t.Lexeme = s.Input[s.Start:s.Pos]
	s.Start = s.Pos

	return t
}