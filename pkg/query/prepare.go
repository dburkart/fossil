/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package query

import (
	"errors"
	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/query/ast"
	"github.com/dburkart/fossil/pkg/query/parser"
	"github.com/dburkart/fossil/pkg/query/plan"
	"github.com/dburkart/fossil/pkg/query/scanner"
	"github.com/dburkart/fossil/pkg/query/validation"
)

func Prepare(d *database.Database, statement string) (database.Filters, error) {
	p := parser.Parser{
		scanner.Scanner{
			Input: statement,
		},
	}

	root, err := p.Parse()
	if err != nil {
		return nil, err
	}

	// Type checking
	typechecker := validation.MakeTypeAnnotator(d)
	ast.Walk(typechecker, root)

	if len(typechecker.Errors) > 0 {
		return nil, errors.New(typechecker.Errors[0].FormatError(statement))
	}

	// Walk the tree
	builder := plan.MetaDataFilterBuilder{DB: d}
	ast.Walk(&builder, root)

	return builder.Filters, err
}
