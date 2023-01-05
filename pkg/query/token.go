/*
 * Copyright (c) 2022-2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package query

type TokenType int

const (
	TOK_INVALID TokenType = iota
	TOK_EOF
	TOK_NL

	TOK_IDENTIFIER
	TOK_KEYWORD
	TOK_NUMBER
	TOK_STRING
	TOK_TOPIC
	TOK_COMMA
	TOK_COLON

	// Arithmetic
	TOK_PLUS
	TOK_MINUS
	TOK_STAR

	// Time
	TOK_WHENCE
	TOK_TIMESPAN

	TOK_PAREN_L
	TOK_PAREN_R

	TOK_ARROW
)

func (t TokenType) ToString() string {
	switch t {
	case TOK_INVALID:
		return "TOK_INVALID"
	case TOK_EOF:
		return "TOK_EOF"
	case TOK_NL:
		return "TOK_NL"
	case TOK_IDENTIFIER:
		return "TOK_IDENTIFIER"
	case TOK_KEYWORD:
		return "TOK_KEYWORD"
	case TOK_NUMBER:
		return "TOK_NUMBER"
	case TOK_STRING:
		return "TOK_STRING"
	case TOK_TOPIC:
		return "TOK_TOPIC"
	case TOK_WHENCE:
		return "TOK_WHENCE"
	case TOK_TIMESPAN:
		return "TOK_TIMESPAN"
	case TOK_COMMA:
		return "TOK_COMMA"
	case TOK_COLON:
		return "TOK_COLON"
	case TOK_PLUS:
		return "TOK_PLUS"
	case TOK_MINUS:
		return "TOK_MINUS"
	case TOK_STAR:
		return "TOK_STAR"
	case TOK_PAREN_L:
		return "TOK_PAREN_L"
	case TOK_PAREN_R:
		return "TOK_PAREN_R"
	case TOK_ARROW:
		return "TOK_ARROW"
	}
	return "TOK_UNKNOWN"
}
