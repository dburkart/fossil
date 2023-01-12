/*
 * Copyright (c) 2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package plan

import (
	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/query/ast"
	"strings"
	"time"
)

type MetaDataFilterBuilder struct {
	Filters database.Filters
	DB      *database.Database
}

func (m *MetaDataFilterBuilder) Visit(node ast.ASTNode) ast.Visitor {

	switch n := node.(type) {
	case *ast.QueryNode:
		return m
	case *ast.QuantifierNode:
		m.Filters = append(m.Filters, m.makeQuantifierFilter(n))
	case *ast.TopicSelectorNode:
		m.Filters = append(m.Filters, m.makeTopicSelectionFilter(n))
	case *ast.TimePredicateNode:
		m.Filters = append(m.Filters, m.makeTimePredicateFilter(n))
	}

	return nil
}

func (m *MetaDataFilterBuilder) makeQuantifierFilter(q *ast.QuantifierNode) database.Filter {
	return func(data database.Entries) database.Entries {
		if data == nil {
			data = m.DB.Retrieve(database.Query{Quantifier: q.Value(), Range: nil})
		}

		switch q.Value() {
		case "all":
			return data
		case "sample":
			quantity, ok := q.TimeQuantity.(ast.Numeric)
			if !ok {
				panic("Expected child to be of type *TimespanNode")
			}

			sampleDuration := quantity.DerivedValue()
			nextTime := data[0].Time
			filtered := database.Entries{}

			for _, val := range data {
				if val.Time.After(nextTime) || val.Time.Equal(nextTime) {
					filtered = append(filtered, val)
					nextTime = val.Time.Add(time.Duration(sampleDuration))
				}
			}

			return filtered

		}
		// TODO: What's the right thing to return here? Maybe we should panic?
		return database.Entries{}
	}
}

func (m *MetaDataFilterBuilder) makeTopicSelectionFilter(q *ast.TopicSelectorNode) database.Filter {
	topic := q.Topic.Lexeme

	// Capture the desired topics in our closure
	var topicFilter = make(map[string]bool)

	// Since topics are hierarchical, we want any topic which has the desired prefix
	for _, t := range m.DB.TopicLookup {
		if strings.HasPrefix(t, topic) {
			topicFilter[t] = true
		}
	}

	return func(data database.Entries) database.Entries {
		if data == nil {
			data = m.DB.Retrieve(database.Query{Range: nil})
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

func (m *MetaDataFilterBuilder) makeTimePredicateFilter(t *ast.TimePredicateNode) database.Filter {
	var startTime, endTime time.Time

	switch t.Value() {
	case "before":
		endTime = t.Begin.(*ast.TimeExpressionNode).Time()
		startTime = m.DB.Segments[0].HeadTime
	case "since":
		startTime = t.Begin.(*ast.TimeExpressionNode).Time()
		endTime = time.Now()
	case "between":
		startTime = t.Begin.(*ast.TimeExpressionNode).Time()
		endTime = t.End.(*ast.TimeExpressionNode).Time()
	}

	timeRange := database.TimeRange{Start: startTime, End: endTime}

	return func(data database.Entries) database.Entries {
		if data == nil {
			return m.DB.Retrieve(database.Query{Range: &timeRange, RangeSemantics: t.Value()})
		}

		// TODO: Handle non-nil case! Let's factor out some of the Retrieve functionality for
		//       filtering ranges.
		return nil
	}
}
