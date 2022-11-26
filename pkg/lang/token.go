/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package lang

type TokenType int

const (
	TOK_INVALID = iota
	TOK_EOF
	TOK_NL

	TOK_IDENTIFIER
	TOK_KEYWORD
	TOK_NUMBER
	TOK_STRING
	TOK_TOPIC
	TOK_COMMA

	TOK_PAREN_L
	TOK_PAREN_R

	TOK_ARROW
)

type Token struct {
	Type   TokenType
	Lexeme string
}
