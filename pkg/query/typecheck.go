/*
 * Copyright (c) 2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package query

import (
	"errors"
	"fmt"
	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/schema"
	"strings"
)

type TypeAnnotator struct {
	Symbols       map[string]schema.Object
	origin        schema.Object
	db            *database.Database
	initialSymbol string
	hasGlobalID   bool
}

func NewTypeAnnotator(db *database.Database) *TypeAnnotator {
	return &TypeAnnotator{Symbols: make(map[string]schema.Object), db: db}
}

func (t *TypeAnnotator) Visit(n ASTNode) error {
	switch nt := n.(type) {
	case *TopicNode:
		topic := nt.Val()
		s := t.db.SchemaForTopic(topic)
		if s == nil {
			return errors.New(fmt.Sprintf("Topic '%s' does not exist in the database.", topic))
		}
		t.origin = s
		if t.initialSymbol != "" {
			t.Symbols[t.initialSymbol] = t.origin
		}
	case *NumberNode:
		nt.TypeI = schema.Type{Name: "int64"}
	case *IdentifierNode:
		if t.origin == nil {
			t.initialSymbol = nt.Value
		} else {
			// Check to make sure we exist in the symbol map
			typeInfo, ok := t.Symbols[nt.Val()]
			if !ok {
				return errors.New(fmt.Sprintf("Unknown identifier '%s'.", nt.Val()))
			}
			nt.TypeI = typeInfo
		}
	case *BinaryOpNode:
		// Both operands must be numeric
		if !nt.Children()[0].Type().IsNumeric() || !nt.Children()[1].Type().IsNumeric() {
			return errors.New(fmt.Sprintf("Operands of '%s' must be numeric", nt.Val()))
		}

		switch nt.Val() {
		case "-", "+", "*":
			nt.TypeI = nt.Children()[0].Type()
			// If one operand is a float, prefer that return type
			if strings.HasPrefix(nt.Children()[1].Type().ToSchema(), "float") {
				nt.TypeI = nt.Children()[1].Type()
			}
		case "/":
			if strings.HasPrefix(nt.Children()[0].Type().ToSchema(), "float") {
				nt.TypeI = nt.Children()[0].Type()
			} else if strings.HasPrefix(nt.Children()[1].Type().ToSchema(), "float") {
				nt.TypeI = nt.Children()[1].Type()
			} else {
				nt.TypeI = schema.Type{Name: "float64"}
			}
		}
	case *TupleNode:
		var innerType schema.Object
		// Each item must have a compatible type
		for _, item := range nt.Children() {
			if innerType == nil {
				innerType = item.Type()
				continue
			}

			if (item.Type().IsNumeric() && !innerType.IsNumeric()) ||
				(!item.Type().IsNumeric() && innerType.IsNumeric()) {
				return errors.New(fmt.Sprintf("Incompatible tuple types '%s' and '%s'", item.Type().ToSchema(), innerType.ToSchema()))
			}

			if strings.HasPrefix(item.Type().ToSchema(), "float") {
				innerType = item.Type()
			}

			// FIXME: Up-sample to largest numeric
		}
		nt.TypeI = schema.Array{Type: *innerType.(*schema.Type), Length: len(nt.Children())}
	case *UnaryOpNode:
		// Child should be numeric
		switch c := nt.Children()[0].(type) {
		case *NumberNode:
			nt.TypeI = c.TypeI
		case *IdentifierNode:
			if c.Type() == nil {
				return errors.New(fmt.Sprintf("Unknown identifier '%s'.", c.Val()))
			}
			if !c.Type().IsNumeric() {
				return errors.New(fmt.Sprintf("Identifier '%s' has invalid type (%s) for operand '%s'", c.Val(), c.Type().ToSchema(), nt.Val()))
			}
			nt.TypeI = c.TypeI
		default:
			return errors.New(fmt.Sprintf("Invalid type (%s) for operand '%s', expected a numeric type.", c.Type().ToSchema(), nt.Val()))
		}
	case *DataFunctionNode:
		nt.TypeI = nt.Children()[0].Type()
		// Reduce must have 2 arguments
		if nt.Val() == "reduce" && len(nt.Arguments) != 2 {
			return errors.New(fmt.Sprintf("The reduce function expects 2 arguments, %d provided", len(nt.Arguments)))
		}

		// Populate symbols for the next stage in our pipeline
		if nt.Next != nil {
			// Ensure we have the same number of return values as the next stage's
			// arguments
			nextNumArgs := len(nt.Next.Arguments)
			var argType schema.Object
			if array, ok := nt.TypeI.(schema.Array); ok {
				if nextNumArgs == 1 {
					argType = nt.TypeI
				} else if nextNumArgs == array.Length {
					argType = array.Type
				} else {
					return errors.New(fmt.Sprintf("Argument mismatch: %s stage expected %d arguments, but got %d", nt.Next.Val(), nextNumArgs, array.Length))
				}
			} else {
				argType = nt.TypeI
			}

			for _, arg := range nt.Next.Arguments {
				t.Symbols[arg.Val()] = argType
			}
		}
	case *DataPipelineNode:
		nt.TypeI = nt.Children()[len(nt.Children())-1].Type()
	}

	return nil
}
