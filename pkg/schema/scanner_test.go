/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package schema

import "testing"

func TestScannerMatchKey(t *testing.T) {
	s := Scanner{
		Input: "\"foobar\"",
	}

	n := s.MatchKey()

	if n != 7 {
		t.Fail()
	}

	s = Scanner{
		Input: "\"invalid",
	}

	n = s.MatchKey()
	if n != 0 {
		t.Fail()
	}

	s = Scanner{
		Input: "invalid",
	}

	n = s.MatchKey()
	if n != 0 {
		t.Fail()
	}

	s = Scanner{
		Input: "\"inv@lid\"",
	}

	n = s.MatchKey()
	if n != 0 {
		t.Fail()
	}
}

func TestScannerTypes(t *testing.T) {
	s := Scanner{Input: "boolean int8 int16 int32 int64 string"}

	expectedKeywordLexemes := []string{"boolean", "int8", "int16", "int32", "int64", "string"}

	for i := 0; i < len(expectedKeywordLexemes); i++ {
		tok := s.Emit()

		if tok.Type != TOK_TYPE {
			t.Error("wanted TOK_TYPE, got", tok.Type)
		}

		if tok.Lexeme != expectedKeywordLexemes[i] {
			t.Errorf("wanted '%s', got '%s'", expectedKeywordLexemes[i], tok.Lexeme)
		}
	}

}