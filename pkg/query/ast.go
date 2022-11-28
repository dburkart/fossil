/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package query

import (
	"github.com/dburkart/fossil/pkg/database"
	"strings"
)

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
	f := n.GenerateFilter(d)

	if len(n.Children()) == 0 {
		if f == nil {
			return []database.Filter{}
		} else {
			return []database.Filter{f}
		}
	}

	var chain []database.Filter

	if f != nil {
		chain = append(chain, f)
	}

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

type TopicSelectorNode struct {
	BaseNode
}

func (q TopicSelectorNode) GenerateFilter(db *database.Database) database.Filter {
	topic, ok := q.Children()[0].(*TopicNode)
	if !ok {
		panic("Expected child to be of type *TopicNode")
	}
	topicName := topic.Value

	// Capture the desired topics in our closure
	var topicFilter = make(map[int]bool)

	// Since topics are hierarchical, we want any topic which has the desired prefix
	for key, element := range db.Topics {
		if strings.HasPrefix(key, topicName) {
			topicFilter[element] = true
		}
	}

	return func(data []database.Datum) []database.Datum {
		if data == nil {
			data = db.Retrieve(database.Query{Range: nil})
		}

		filtered := []database.Datum{}

		for _, val := range data {
			if _, ok := topicFilter[val.TopicID]; ok {
				filtered = append(filtered, val)
			}
		}

		return filtered
	}
}

type TopicNode struct {
	BaseNode
}
