/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package parser

import (
	"errors"
	"fmt"
	"time"
	"unicode"
	"unicode/utf8"
)

var numberFormats = [...]string{
	time.RFC3339,
	time.RFC3339Nano,
	time.RFC822,
	time.RFC822Z,
	time.Layout,
	"2006/01/02",
	"02/01/2006",
}

var letterFormats = [...]string{
	"Jan 02, 2006",
	time.RFC850,
	time.UnixDate,
	time.RFC1123,
	time.RFC1123Z,
	time.Stamp,
}

func ParseVagueDateTime(some string) (time.Time, error) {
	first, _ := utf8.DecodeRuneInString(some)
	var theFmt string
	var found time.Time

	switch {
	case unicode.IsDigit(first):
		for _, theFmt = range numberFormats {
			tm, err := time.Parse(theFmt, some)
			if err == nil {
				found = tm
				break
			}
		}
	default:
		for _, theFmt := range letterFormats {
			tm, err := time.Parse(theFmt, some)
			if err == nil {
				found = tm
				break
			}
		}
	}

	if found.IsZero() {
		return found, errors.New(fmt.Sprintf("Specified time '%s' did not match a known timestamp", some))
	}

	return found, nil
}
