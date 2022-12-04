/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package query

import "github.com/dburkart/fossil/pkg/database"

func Prepare(d *database.Database, statement string) (database.Filters, error) {
	p := Parser{
		Scanner{
			Input: statement,
		},
	}

	ast, err := p.Parse()
	if err != nil {
		return []database.Filter{}, err
	}

	// Walk the tree
	filters := ast.Walk(d)

	return filters, err
}
