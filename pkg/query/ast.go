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
}

type BaseNode struct {
	Value    string
	children []ASTNode
}

func (b *BaseNode) Children() []ASTNode {
	return b.children
}

func (b *BaseNode) AddChild(child ASTNode) {
	b.children = append(b.children, child)
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

func (q QuantifierNode) GenerateFilter(*database.Database) database.Filter {
	return func(data []database.Datum) []database.Datum {
		switch q.Value {
		case "all":
			return data
		}
		// TODO: What's the right thing to return here? Maybe we should panic?
		return []database.Datum{}
	}
}
