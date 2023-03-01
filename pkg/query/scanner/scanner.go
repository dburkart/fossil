/*
 * Copyright (c) 2022-2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package scanner

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

// MatchIdentifier returns the length of the next token, assuming it is an
// identifier.
//
// Grammar:
//
//	identifier      = 1*(ALPHA / DIGIT / '_' / '-')
func (s *Scanner) MatchIdentifier() int {
	i := s.Pos
	r, width := utf8.DecodeRuneInString(s.Input[i:])
	size := 0

	for unicode.IsDigit(r) || unicode.IsLetter(r) || r == '-' || r == '_' {
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

	for unicode.IsDigit(r) || unicode.IsLetter(r) || r == '/' ||
		r == '-' || r == '_' || r == '#' || r == '.' || r == '@' {
		size += width
		i += width
		r, width = utf8.DecodeRuneInString(s.Input[i:])
	}

	return size
}

// MatchInteger returns the length of the next token, assuming it is a
// number
//
// Grammar:
//
//	integer          = 1*DIGIT
func (s *Scanner) MatchInteger() int {
	r, width := utf8.DecodeRuneInString(s.Input[s.Pos:])
	size := 0

	for i := s.Pos; unicode.IsDigit(r); {
		size += width
		i += width
		r, width = utf8.DecodeRuneInString(s.Input[i:])
	}

	return size
}

// MatchFloat returns the length of the next token, assuming it is a
// floating point number
//
// Grammar:
//
//  float           = *DIGIT "." 1*DIGIT
func (s *Scanner) MatchFloat() int {
	r, width := utf8.DecodeRuneInString(s.Input[s.Pos:])
	lsize := 0
	rsize := 0

	for i := s.Pos; unicode.IsDigit(r); {
		lsize += width
		i += width
		r, width = utf8.DecodeRuneInString(s.Input[i:])
	}

	if r != '.' {
		return 0
	}

	r, width = utf8.DecodeRuneInString(s.Input[s.Pos+lsize+1:])

	for i := s.Pos + lsize + 1; unicode.IsDigit(r); {
		rsize += width
		i += width
		r, width = utf8.DecodeRuneInString(s.Input[i:])
	}

	if rsize == 0 {
		return 0
	}

	return lsize + rsize + 1
}

// MatchString returns the length of the next token, assuming it is a
// string
//
// Grammar:
//
//	string          = DQUOTE *ALPHANUM DQUOTE / SQUOTE *ALPHANUM SQUOTE
func (s *Scanner) MatchString() int {
	r, width := utf8.DecodeRuneInString(s.Input[s.Pos:])
	size := 0

	quote := r
	r, width = utf8.DecodeRuneInString(s.Input[s.Pos+1:])
	for r != quote {
		if r == utf8.RuneError {
			return 0
		}
		size += width
		r, width = utf8.DecodeRuneInString(s.Input[s.Pos+size+1:])
	}

	// Include quote runes
	return size + 2
}

// MatchTimeWhence returns the length of the next token, assuming it is
// a time-whence
//
// Grammar:
//
//	time-whence     = "~now" / "~" RFC3339
func (s *Scanner) MatchTimeWhence() int {
	r, _ := utf8.DecodeRuneInString(s.Input[s.Pos:])

	if r != '~' {
		return 0
	}

	pos := s.Pos + 1
	r, _ = utf8.DecodeRuneInString(s.Input[pos:])

	// If the next rune is an open parenthesis, we will assume that we need to match
	// an RFC3339 timestamp
	if r == '(' {
		pos = pos + 1
		r, _ = utf8.DecodeRuneInString(s.Input[pos:])

		// Find the next boundary
		var end int
		for end = pos; rune(s.Input[end]) != ')'; end++ {
			if end == len(s.Input)-1 {
				break
			}
		}

		// Add back one for '~', and another to include "end"
		return end - pos + 3
	}

	if strings.HasPrefix(s.Input[pos:], "now") {
		return len("~now")
	}

	return 0
}

// MatchTimespan returns the length of the next token, assuming it is a
// timespan
//
// Grammar:
//
//	timespan        = "@second" / "@minute" / "@hour" / "@day" / "@week" / "@month" / "@year"
func (s *Scanner) MatchTimespan() int {
	r, _ := utf8.DecodeRuneInString(s.Input[s.Pos:])

	if r != '@' {
		return 0
	}

	pos := s.Pos + 1
	r, _ = utf8.DecodeRuneInString(s.Input[pos:])

	switch r {
	case 'd':
		if strings.HasPrefix(s.Input[pos:], "day") {
			return len("@day")
		}
	case 'h':
		if strings.HasPrefix(s.Input[pos:], "hour") {
			return len("@hour")
		}
	case 'm':
		if strings.HasPrefix(s.Input[pos:], "month") {
			return len("@month")
		}

		if strings.HasPrefix(s.Input[pos:], "minute") {
			return len("@minute")
		}
	case 's':
		if strings.HasPrefix(s.Input[pos:], "second") {
			return len("@second")
		}
	case 'w':
		if strings.HasPrefix(s.Input[pos:], "week") {
			return len("@week")
		}
	case 'y':
		if strings.HasPrefix(s.Input[pos:], "year") {
			return len("@year")
		}
	}

	return 0
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
		case r == ':':
			t.Type = TOK_COLON
			skip = width
		case r == '|':
			t.Type = TOK_PIPE
			skip = width
		case r == '(':
			t.Type = TOK_PAREN_L
			skip = width
		case r == ')':
			t.Type = TOK_PAREN_R
			skip = width
		case r == '[':
			t.Type = TOK_BRACKET_L
			skip = width
		case r == ']':
			t.Type = TOK_BRACKET_R
			skip = width
		case r == '=':
			if strings.HasPrefix(s.Input[s.Pos:], "==") {
				t.Type = TOK_EQ_EQ
				skip = len("==")
				break
			}
			t.Type = TOK_INVALID
			skip = s.SkipToBoundary(isDelimiter)
		case r == '!':
			if strings.HasPrefix(s.Input[s.Pos:], "!=") {
				t.Type = TOK_NOT_EQ
				skip = len("!=")
				break
			}
			t.Type = TOK_INVALID
			skip = s.SkipToBoundary(isDelimiter)
		case r == '<':
			if strings.HasPrefix(s.Input[s.Pos:], "<=") {
				t.Type = TOK_LESS_EQ
				skip = len("<=")
				break
			}
			t.Type = TOK_LESS
			skip = width
		case r == '>':
			if strings.HasPrefix(s.Input[s.Pos:], ">=") {
				t.Type = TOK_GREATER_EQ
				skip = len(">=")
				break
			}
			t.Type = TOK_GREATER
			skip = width
		case r == '*':
			t.Type = TOK_STAR
			skip = width
		case r == '+':
			t.Type = TOK_PLUS
			skip = width
		case r == '-':
			if strings.HasPrefix(s.Input[s.Pos:], "->") {
				t.Type = TOK_ARROW
				skip = len("->")
				break
			}
			t.Type = TOK_MINUS
			skip = width
		case r == ',':
			t.Type = TOK_COMMA
			skip = width
		case r == '/':
			next, _ := utf8.DecodeRuneInString(s.Input[s.Pos+1:])
			if isDelimiter(next) || !unicode.IsLetter(next) {
				t.Type = TOK_SLASH
				skip = width
				break
			}
			t.Type = TOK_TOPIC
			skip = s.MatchTopic()
		case r == '~':
			skip = s.MatchTimeWhence()
			if skip > 0 {
				t.Type = TOK_WHENCE
			} else {
				t.Type = TOK_INVALID
				skip = s.SkipToBoundary(isNonTimeDelimiter)
			}
		case r == '@':
			skip = s.MatchTimespan()
			if skip > 0 {
				t.Type = TOK_TIMESPAN
			} else {
				t.Type = TOK_INVALID
				skip = s.SkipToBoundary(isDelimiter)
			}
		case r == '\'' || r == '"':
			skip = s.MatchString()
			if skip > 0 {
				t.Type = TOK_STRING
			} else {
				t.Type = TOK_INVALID
				skip = s.SkipToBoundary(isDelimiter)
			}
		case r == '.':
			skip = s.MatchFloat()
			if skip > 0 {
				t.Type = TOK_FLOAT
			} else {
				t.Type = TOK_INVALID
				skip = s.SkipToBoundary(isDelimiter)
			}
		case unicode.IsDigit(r):
			skip = s.MatchFloat()
			if skip > 0 {
				t.Type = TOK_FLOAT
			} else {
				skip = s.MatchInteger()
				t.Type = TOK_INTEGER
			}
		case r == 'a':
			if strings.HasPrefix(s.Input[s.Pos:], "all") {
				t.Type = TOK_KEYWORD
				skip = len("all")
				break
			}
			identifierFallthrough()
		case r == 'b':
			if strings.HasPrefix(s.Input[s.Pos:], "before") {
				t.Type = TOK_KEYWORD
				skip = len("before")
				break
			}

			if strings.HasPrefix(s.Input[s.Pos:], "between") {
				t.Type = TOK_KEYWORD
				skip = len("between")
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
			if strings.HasPrefix(s.Input[s.Pos:], "since") {
				t.Type = TOK_KEYWORD
				skip = len("since")
				break
			}
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
	return unicode.IsSpace(r) || r == '(' || r == ')' || r == ',' || r == '-'
}

func isNonTimeDelimiter(r rune) bool {
	return unicode.IsSpace(r) || r == '(' || r == ')' || r == ','
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
