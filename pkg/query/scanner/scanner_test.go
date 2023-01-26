/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package scanner

import (
	"testing"
)

func TestMatchTimespan(t *testing.T) {
	s := Scanner{Input: "@hour"}
	width := s.MatchTimespan()

	if width != 5 {
		t.Errorf("@hour should have width of 5, not %d", width)
	}

	s.Input = "@bogus"
	width = s.MatchTimespan()

	if width != 0 {
		t.Error("@bogus should not have a width!")
	}
}

func TestMatchTimeWhence(t *testing.T) {
	s := Scanner{Input: "~now"}

	width := s.MatchTimeWhence()
	if width != len("~now") {
		t.Errorf("~now should have width %d, not %d", len("~now"), width)
	}

	s.Input = "~(2006-01-02T15:04:05-07:00)"
	width = s.MatchTimeWhence()
	if width != len("~(2006-01-02T15:04:05-07:00)") {
		t.Errorf("RFC3339 should have length %d, not %d", len("~(2006-01-02T15:04:05-07:00)"), width)
	}
}

func TestEmitNumber(t *testing.T) {
	s := Scanner{Input: "12345 hi"}

	tok := s.Emit()

	if tok.Type != TOK_INTEGER {
		t.Error("wanted TOK_INTEGER, got", tok.Type.ToString())
	}

	if tok.Lexeme != "12345" {
		t.Error("wanted 12345, got", tok.Lexeme)
	}
}

func TestEmitFloat(t *testing.T) {
	s := Scanner{Input: "1.5 .3 6.0 54"}

	wantTypes := []TokenType{TOK_FLOAT, TOK_FLOAT, TOK_FLOAT, TOK_INTEGER}
	wantLexemes := []string{"1.5", ".3", "6.0", "54"}

	for i := 0; i < len(wantTypes); i++ {
		tok := s.Emit()

		if tok.Type != wantTypes[i] {
			t.Error("wanted", wantTypes[i].ToString(), ", got", tok.Type.ToString())
		}

		if tok.Lexeme != wantLexemes[i] {
			t.Error("wanted", wantLexemes[i], ", got", tok.Lexeme)
		}
	}
}

func TestEmitKeyword(t *testing.T) {
	s := Scanner{Input: "   all in sample"}

	expectedKeywordLexemes := []string{"all", "in", "sample"}

	for i := 0; i < len(expectedKeywordLexemes); i++ {
		tok := s.Emit()

		if tok.Type != TOK_KEYWORD {
			t.Error("wanted TOK_KEYWORD, got", tok.Type.ToString())
		}

		if tok.Lexeme != expectedKeywordLexemes[i] {
			t.Errorf("wanted '%s', got '%s'", expectedKeywordLexemes[i], tok.Lexeme)
		}
	}

}

func TestEmitIdentifier(t *testing.T) {
	s := Scanner{Input: "variable a3 "}
	tok := s.Emit()

	if tok.Type != TOK_IDENTIFIER {
		t.Error("wanted TOK_IDENTIFIER, got", tok.Type.ToString())
	}

	if tok.Lexeme != "variable" {
		t.Error("wanted 'variable', got", tok.Lexeme)
	}

	tok = s.Emit()

	if tok.Type != TOK_IDENTIFIER {
		t.Error("wanted TOK_IDENTIFIER, got", tok.Type.ToString())
	}

	if tok.Lexeme != "a3" {
		t.Error("wanted 'a3', got", tok.Lexeme)
	}
}

func TestEmitTopic(t *testing.T) {
	s := Scanner{Input: "/foo/bar/baz / /c02f3a2a-2791-443b-a2e9-c5e29740b803/"}
	expectedTopicLexemes := []string{"/foo/bar/baz", "/", "/c02f3a2a-2791-443b-a2e9-c5e29740b803/"}

	for i := 0; i < len(expectedTopicLexemes); i++ {
		tok := s.Emit()

		if tok.Type != TOK_TOPIC && tok.Type != TOK_SLASH {
			t.Error("wanted TOK_TOPIC, got", tok.Type.ToString())
		}

		if tok.Lexeme != expectedTopicLexemes[i] {
			t.Errorf("wanted '%s', got '%s'", expectedTopicLexemes[i], tok.Lexeme)
		}
	}
}
