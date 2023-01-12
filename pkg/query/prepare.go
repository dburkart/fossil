/*
 * Copyright (c) 2022-2023, Dana Burkart <dana.burkart@gmail.com>
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

type Query struct {
	Filters database.Filters
}

func (q *Query) Execute() database.Result {
	result := q.Filters.Execute()
	return result
}

func Prepare(d *database.Database, statement string) (Query, error) {
	p := parser.Parser{
		scanner.Scanner{
			Input: statement,
		},
	}

	root, err := p.Parse()
	if err != nil {
		return Query{}, err
	}

	// Type checking
	checker := validation.MakeTypeAnnotator(d)
	ast.Walk(checker, root)

	if len(checker.Errors) > 0 {
		// FIXME: Handle multiple errors
		return Query{}, errors.New(checker.Errors[0].FormatError(statement))
	}

	// Build metadata filters
	builder := plan.MetaDataFilterBuilder{DB: d}
	ast.Walk(&builder, root)

	return Query{Filters: builder.Filters}, err
}
