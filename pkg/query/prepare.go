/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package query

import (
	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/query/parser"
	"github.com/dburkart/fossil/pkg/query/scanner"
)

func Prepare(d *database.Database, statement string) (database.Filters, error) {
	p := parser.Parser{
		scanner.Scanner{
			Input: statement,
		},
	}

	_, err := p.Parse()
	if err != nil {
		return nil, err
	}

	// Pre-validation
	//validations := []root.Visitor{
	//	types.NewTypeAnnotator(d),
	//}
	//
	//for _, validation := range validations {
	//	err = root.WalkTree(root, validation)
	//	if err != nil {
	//		return nil, err
	//	}
	//}

	// Walk the tree
	//filters := root.Walk(d)

	return nil, err
}
