/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package query

import "github.com/dburkart/fossil/pkg/database"

type ASTNode interface {
	Children() []ASTNode
	GenerateFilter(*database.Database) database.Filter
	Walk(*database.Database) []database.Filter
}

type BaseNode struct {
	Value    string
	children []ASTNode
}

func (b *BaseNode) Children() []ASTNode {
	return b.children
}

func (b *BaseNode) GenerateFilter(_ *database.Database) database.Filter {
	return nil
}

func (b *BaseNode) AddChild(child ASTNode) {
	b.children = append(b.children, child)
}

func (b *BaseNode) descend(d *database.Database, n ASTNode) []database.Filter {
	if len(n.Children()) == 0 {
		f := n.GenerateFilter(d)
		if f == nil {
			return []database.Filter{}
		} else {
			return []database.Filter{f}
		}
	}

	var chain []database.Filter

	for _, child := range n.Children() {
		chain = append(chain, b.descend(d, child)...)
	}

	return chain
}

func (b *BaseNode) Walk(d *database.Database) []database.Filter {
	return b.descend(d, b)
}

type QueryNode struct {
	BaseNode
}

func (q QueryNode) GenerateFilter(_ *database.Database) database.Filter {
	return nil
}

type QuantifierNode struct {
	BaseNode
}

func (q QuantifierNode) GenerateFilter(db *database.Database) database.Filter {
	return func(data []database.Datum) []database.Datum {
		if data == nil {
			data = db.Retrieve(database.Query{Quantifier: q.Value, Range: nil})
		}

		switch q.Value {
		case "all":
			return data
		}
		// TODO: What's the right thing to return here? Maybe we should panic?
		return []database.Datum{}
	}
}
