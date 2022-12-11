/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package schema

type TokenType int

const (
	TOK_INVALID = iota
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

type Token struct {
	Type     TokenType
	Lexeme   string
	Location [2]int
}
