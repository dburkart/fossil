/*
 * Copyright (c) 2022-2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package scanner

type TokenType int

const (
	TOK_INVALID TokenType = iota
	TOK_EOF
	TOK_NL

	TOK_IDENTIFIER
	TOK_KEYWORD
	TOK_INTEGER
	TOK_FLOAT
	TOK_STRING
	TOK_TOPIC
	TOK_COMMA
	TOK_COLON
	TOK_PIPE

	// Expressions
	TOK_EQ_EQ
	TOK_NOT_EQ
	TOK_LESS
	TOK_LESS_EQ
	TOK_GREATER
	TOK_GREATER_EQ
	TOK_PLUS
	TOK_MINUS
	TOK_SLASH
	TOK_STAR

	// Time
	TOK_WHENCE
	TOK_TIMESPAN

	TOK_PAREN_L
	TOK_PAREN_R
	TOK_BRACKET_L
	TOK_BRACKET_R

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
	case TOK_INTEGER:
		return "TOK_INTEGER"
	case TOK_FLOAT:
		return "TOK_FLOAT"
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
	case TOK_EQ_EQ:
		return "TOK_EQ_EQ"
	case TOK_NOT_EQ:
		return "TOK_NOT_EQ"
	case TOK_LESS:
		return "TOK_LESS"
	case TOK_LESS_EQ:
		return "TOK_LESS_EQ"
	case TOK_GREATER:
		return "TOK_GREATER"
	case TOK_GREATER_EQ:
		return "TOK_GREATER_EQ"
	case TOK_PLUS:
		return "TOK_PLUS"
	case TOK_MINUS:
		return "TOK_MINUS"
	case TOK_SLASH:
		return "TOK_SLASH"
	case TOK_STAR:
		return "TOK_STAR"
	case TOK_PAREN_L:
		return "TOK_PAREN_L"
	case TOK_PAREN_R:
		return "TOK_PAREN_R"
	case TOK_ARROW:
		return "TOK_ARROW"
	case TOK_PIPE:
		return "TOK_PIPE"
	}
	return "TOK_UNKNOWN"
}
