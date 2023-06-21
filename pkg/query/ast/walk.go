/*
 * Copyright (c) 2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package ast

func Walk(v Visitor, node ASTNode) {
	if v.Visit(node); v == nil {
		return
	}

	switch n := node.(type) {
	case *QueryNode:
		Walk(v, n.Quantifier)

		if n.Identifier != nil {
			Walk(v, n.Identifier)
		}

		if n.Topic != nil {
			Walk(v, n.Topic)
		}

		if n.TimePredicate != nil {
			Walk(v, n.TimePredicate)
		}

		if n.DataPipeline != nil {
			Walk(v, n.DataPipeline)
		}

	case *QuantifierNode:
		if n.TimeQuantity != nil {
			Walk(v, n.TimeQuantity)
		}

	case *TopicSelectorNode:
		// Skip, leaf node

	case *TimePredicateNode:
		Walk(v, n.Begin)

		if n.End != nil {
			Walk(v, n.End)
		}

	case *TimeExpressionNode:
		Walk(v, n.Whence)

		if n.Quantity != nil {
			Walk(v, n.Quantity)
		}

	case *TimeWhenceNode:
		// Skip, leaf node

	case *BinaryOpNode:
		Walk(v, n.Left)
		Walk(v, n.Right)

	case *UnaryOpNode:
		Walk(v, n.Operand)

	case *TimespanNode, *IdentifierNode, *NumberNode, *StringNode, *ElementNode:
		// Skip, leaf nodes

	case *TupleNode:
		for _, e := range n.Elements {
			Walk(v, e)
		}

	case *DataPipelineNode:
		for _, s := range n.Stages {
			Walk(v, s)
		}

	case *DataFunctionNode:
		Walk(v, n.Expression)

	case *BuiltinFunctionNode:
		Walk(v, n.Expression)

	case *CompositeNode:
		for idx, _ := range n.Keys {
			Walk(v, &n.Keys[idx])
			Walk(v, n.Values[idx])
		}

	default:
		panic("Unexpected ASTNode passed to Walk")
	}

	v.Visit(nil)
}
