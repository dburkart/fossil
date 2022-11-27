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
	s := lang.Scanner{Input: "   all in"}

	expectedKeywordLexemes := []string{"all", "in"}

	for i := 0; i < len(expectedKeywordLexemes); i++ {
		tok := s.Emit()

		if tok.Type != lang.TOK_KEYWORD {
			t.Error("wanted TOK_KEYWORD, got", tok.Type)
		}

		if tok.Lexeme != expectedKeywordLexemes[i] {
			t.Errorf("wanted '%s', got '%s'", expectedKeywordLexemes[i], tok.Lexeme)
		}
	}

}

func TestEmitIdentifier(t *testing.T) {
	s := lang.Scanner{Input: "variable a3 "}
	tok := s.Emit()

	if tok.Type != lang.TOK_IDENTIFIER {
		t.Error("wanted TOK_IDENTIFIER, got", tok.Type)
	}

	if tok.Lexeme != "variable" {
		t.Error("wanted 'variable', got", tok.Lexeme)
	}

	tok = s.Emit()

	if tok.Type != lang.TOK_IDENTIFIER {
		t.Error("wanted TOK_IDENTIFIER, got", tok.Type)
	}

	if tok.Lexeme != "a3" {
		t.Error("wanted 'a3', got", tok.Lexeme)
	}
}

func TestEmitTopic(t *testing.T) {
	s := lang.Scanner{Input: "/foo/bar/baz /"}
	expectedTopicLexemes := []string{"/foo/bar/baz", "/"}

	for i := 0; i < len(expectedTopicLexemes); i++ {
		tok := s.Emit()

		if tok.Type != lang.TOK_TOPIC {
			t.Error("wanted TOK_TOPIC, got", tok.Type)
		}

		if tok.Lexeme != expectedTopicLexemes[i] {
			t.Errorf("wanted '%s', got '%s'", expectedTopicLexemes[i], tok.Lexeme)
		}
	}
}
