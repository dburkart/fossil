/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package schema

type TokenType int

const (
	TOK_INVALID TokenType = iota
	TOK_EOF

	TOK_TYPE
	TOK_COLON
	TOK_COMMA
	TOK_KEY
	TOK_NUMBER

	TOK_BRACKET_O
	TOK_BRACKET_X

	TOK_CURLY_O
	TOK_CURLY_X
)

func (t TokenType) ToString() string {
	switch t {
	case TOK_INVALID:
		return "TOK_INVALID"
	case TOK_EOF:
		return "TOK_EOF"
	case TOK_TYPE:
		return "TOK_TYPE"
	case TOK_COLON:
		return "TOK_COLON"
	case TOK_COMMA:
		return "TOK_COMMA"
	case TOK_KEY:
		return "TOK_KEY"
	case TOK_NUMBER:
		return "TOK_NUMBER"
	case TOK_BRACKET_O:
		return "TOK_BRACKET_O"
	case TOK_BRACKET_X:
		return "TOK_BRACKET_X"
	case TOK_CURLY_O:
		return "TOK_CURLY_O"
	case TOK_CURLY_X:
		return "TOK_CURLY_X"
	}
	return "TOK_UNKNOWN"
}
