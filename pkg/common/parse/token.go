/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package parse

type TokenType interface {
	ToString() string
}

type Token struct {
	Type     TokenType
	Lexeme   string
	Location [2]int
}
