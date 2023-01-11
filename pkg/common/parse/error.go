/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package parse

import (
	"fmt"
	"strings"
)

type SyntaxError struct {
	Location Location
	Message  string
}

func NewSyntaxError(t Token, m string) SyntaxError {
	return SyntaxError{Location: t.Location, Message: m}
}

func (s *SyntaxError) FormatError(input string) string {
	repeat := s.Location.End - s.Location.Start - 1
	if repeat < 0 {
		repeat = 0
	}

	errorString := "Syntax error found in query:\n"
	errorString += input
	errorString += fmt.Sprintf("\n%s^%s ", strings.Repeat(" ", s.Location.Start), strings.Repeat("~", repeat))
	errorString += fmt.Sprintf("%s\n", s.Message)
	return errorString
}
