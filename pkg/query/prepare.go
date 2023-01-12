/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package query

import (
	"github.com/dburkart/fossil/pkg/database"
	ast2 "github.com/dburkart/fossil/pkg/query/ast"
	"github.com/dburkart/fossil/pkg/query/parser"
	"github.com/dburkart/fossil/pkg/query/scanner"
	"github.com/dburkart/fossil/pkg/query/types"
)

func Prepare(d *database.Database, statement string) (database.Filters, error) {
	p := parser.Parser{
		scanner.Scanner{
			Input: statement,
		},
	}

	ast, err := p.Parse()
	if err != nil {
		return nil, err
	}

	// Pre-validation
	validations := []ast2.Visitor{
		types.NewTypeAnnotator(d),
	}

	for _, validation := range validations {
		err = ast2.WalkTree(ast, validation)
		if err != nil {
			return nil, err
		}
	}

	// Walk the tree
	filters := ast.Walk(d)

	return filters, err
}
