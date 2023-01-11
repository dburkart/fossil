/*
 * Copyright (c) 2022-2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package parse

type TokenType interface {
	ToString() string
}

type Location struct {
	Start int
	End   int
}

type Token struct {
	Type     TokenType
	Lexeme   string
	Location Location
}
