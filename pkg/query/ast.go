/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package query

import (
	"github.com/dburkart/fossil/pkg/database"
	"strings"
	"time"
)

type ASTNode interface {
	Children() []ASTNode
	GenerateFilter(*database.Database) database.Filter
	Walk(*database.Database) []database.Filter
}

type (
	BaseNode struct {
		Value    string
		children []ASTNode
	}

	QueryNode struct {
		BaseNode
	}

	QuantifierNode struct {
		BaseNode
	}

	TopicSelectorNode struct {
		BaseNode
	}

	TopicNode struct {
		BaseNode
	}

	TimePredicateNode struct {
		BaseNode
	}

	TimeExpressionNode struct {
		BaseNode
	}

	TimeWhenceNode struct {
		BaseNode
	}

	TimespanNode struct {
		BaseNode
	}
)

//-- BaseNode

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

//-- QueryNode

func (q QueryNode) GenerateFilter(_ *database.Database) database.Filter {
	return nil
}

//-- QuantifierNode

func (q QuantifierNode) GenerateFilter(db *database.Database) database.Filter {
	return func(data database.Entries) database.Entries {
		if data == nil {
			data = db.Retrieve(database.Query{Quantifier: q.Value, Range: nil})
		}

		switch q.Value {
		case "all":
			return data
		case "sample":
			timespan, ok := q.Children()[0].(*TimespanNode)
			if !ok {
				panic("Expected child to be of type *TimespanNode")
			}

			sampleDuration := timespan.Duration()
			nextTime := data[0].Time
			filtered := database.Entries{}

			for _, val := range data {
				if val.Time.After(nextTime) || val.Time.Equal(nextTime) {
					filtered = append(filtered, val)
					nextTime = val.Time.Add(sampleDuration)
				}
			}

			return filtered

		}
		// TODO: What's the right thing to return here? Maybe we should panic?
		return database.Entries{}
	}
}

//-- TopicSelectorNode

func (q TopicSelectorNode) GenerateFilter(db *database.Database) database.Filter {
	topic, ok := q.Children()[0].(*TopicNode)
	if !ok {
		panic("Expected child to be of type *TopicNode")
	}
	topicName := topic.Value

	// Capture the desired topics in our closure
	var topicFilter = make(map[string]bool)

	// Since topics are hierarchical, we want any topic which has the desired prefix
	for key := range db.Topics {
		if strings.HasPrefix(key, topicName) {
			topicFilter[key] = true
		}
	}

	return func(data database.Entries) database.Entries {
		if data == nil {
			data = db.Retrieve(database.Query{Range: nil})
		}

		filtered := database.Entries{}

		for _, val := range data {
			if _, ok := topicFilter[val.Topic]; ok {
				filtered = append(filtered, val)
			}
		}

		return filtered
	}
}

//-- TimePredicateNode

func (t TimePredicateNode) GenerateFilter(db *database.Database) database.Filter {
	var startTime, endTime time.Time
	var err error

	switch t.Value {
	case "since":
		startTime, err = t.Children()[0].(*TimeExpressionNode).Time()
		// It shouldn't be possible there to be an error here (we should catch
		// it earlier when parsing, so panic
		if err != nil {
			panic(err)
		}

		endTime = time.Now()
	}

	timeRange := database.TimeRange{Start: startTime, End: endTime}

	return func(data database.Entries) database.Entries {
		if data == nil {
			return db.Retrieve(database.Query{Range: &timeRange, RangeSemantics: t.Value})
		}

		// TODO: Handle non-nil case! Let's factor out some of the Retrieve functionality for
		//       filtering ranges.
		return nil
	}
}

//-- TimeExpressionNode

func (t TimeExpressionNode) Time() (time.Time, error) {
	// TODO: Support full time-expression syntax here
	child := t.Children()[0].(*TimeWhenceNode)
	return child.Time()
}

//-- TimeWhenceNode

func (t TimeWhenceNode) Time() (time.Time, error) {
	switch {
	case t.Value == "~now":
		return time.Now(), nil
	default:
		return time.Parse(time.RFC3339, t.Value[1:])
	}
}

//-- TimespanNode

func (t TimespanNode) Duration() time.Duration {
	switch t.Value {
	case "@year":
		return time.Hour * 24 * 365
	case "@month":
		return time.Hour * 24 * 30
	case "@day":
		return time.Hour * 24
	case "@hour":
		return time.Hour
	case "@minute":
		return time.Minute
	case "@second":
		return time.Second
	}
	return 0
}
