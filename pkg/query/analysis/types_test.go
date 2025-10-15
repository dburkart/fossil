package analysis

import (
	"testing"

	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/query/ast"
	queryparser "github.com/dburkart/fossil/pkg/query/parser"
	"github.com/dburkart/fossil/pkg/query/scanner"
)

func TestTypeCheckerAllowsStringEquality(t *testing.T) {
	tempDir := t.TempDir()

	db, err := database.NewDatabase("test", tempDir)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	schema := "{\"key\":string,\"value\":int64,}"
	db.AddTopic("/dicts", schema)

	p := queryparser.Parser{Scanner: scanner.Scanner{Input: "all in /dicts | filter x -> x[key] == \"id\""}}
	root, err := p.Parse()
	if err != nil {
		t.Fatalf("failed to parse query: %v", err)
	}

	checker := MakeTypeChecker(db)
	ast.Walk(checker, root)

	if len(checker.Errors) != 0 {
		t.Fatalf("expected no type errors, got %v", checker.Errors)
	}
}
