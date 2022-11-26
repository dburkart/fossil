package lang_test

import (
	"github.com/dburkart/fossil/pkg/lang"
	"testing"
)

func TestEmitNumber(t *testing.T) {
	s := lang.Scanner{Input: "12345 hi"}

	tok := s.Emit()

	if tok.Type != lang.TOK_NUMBER {
		t.Error("wanted TOK_NUMBER, got", tok.Type)
	}

	if tok.Lexeme != "12345" {
		t.Error("wanted 12345, got", tok.Lexeme)
	}
}

func TestEmitKeyword(t *testing.T) {
	s := lang.Scanner{Input: "  all of"}
	tok := s.Emit()

	if tok.Type != lang.TOK_KEYWORD {
		t.Error("wanted TOK_IDENTIFIER, got", tok.Type)
	}

	if tok.Lexeme != "all" {
		t.Error("wanted all, got", tok.Lexeme)
	}
}
