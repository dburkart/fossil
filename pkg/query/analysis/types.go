/*
 * Copyright (c) 2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package analysis

import (
	"fmt"
	"strings"

	"github.com/dburkart/fossil/pkg/common/parse"
	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/query/ast"
	"github.com/dburkart/fossil/pkg/query/scanner"
	"github.com/dburkart/fossil/pkg/query/types"
	"github.com/dburkart/fossil/pkg/schema"
)

type TypeChecker struct {
	Errors      []parse.SyntaxError
	initialType schema.Object
	symbols     map[string]schema.Object
	typeLookup  map[ast.ASTNode]schema.Object
	locations   map[ast.ASTNode]parse.Location
	nodes       []ast.ASTNode
	db          *database.Database
}

func MakeTypeChecker(db *database.Database) *TypeChecker {
	return &TypeChecker{
		symbols:    make(map[string]schema.Object),
		typeLookup: make(map[ast.ASTNode]schema.Object),
		locations:  make(map[ast.ASTNode]parse.Location),
		db:         db,
	}
}

// FIXME: Factor out stack into it's own thing
func (t *TypeChecker) push(node ast.ASTNode) {
	t.nodes = append(t.nodes, node)
}

func (t *TypeChecker) pop() ast.ASTNode {
	if len(t.nodes) == 0 {
		return nil
	}
	node := t.nodes[len(t.nodes)-1]
	t.nodes = t.nodes[:len(t.nodes)-1]
	return node
}

func (t *TypeChecker) typeForNode(node ast.ASTNode) schema.Object {
	nt, ok := t.typeLookup[node]
	if !ok {
		return schema.Unknown{}
	}
	return nt
}

func (t *TypeChecker) Visit(node ast.ASTNode) ast.Visitor {
	if node == nil {
		node = t.pop()
		if node == nil {
			return nil
		}

		switch n := node.(type) {
		case *ast.NumberNode:
			t.typeLookup[n] = &schema.Type{Name: "int64"}
			t.locations[n] = n.Token.Location
		case *ast.StringNode:
			t.typeLookup[n] = &schema.Type{Name: "string"}
			t.locations[n] = n.Token.Location
		case *ast.IdentifierNode:
			s, ok := t.symbols[n.Value()]
			if !ok {
				t.Errors = append(t.Errors, parse.NewSyntaxError(n.Token, fmt.Sprintf("Unable to infer type of identifier '%s'", n.Value())))
				return nil
			}
			t.typeLookup[n] = s
			t.locations[n] = n.Token.Location
		case *ast.ElementNode:
			var array *schema.Array
			var composite *schema.Composite
			s, ok := t.symbols[n.Identifier.Value()]
			if !ok {
				t.Errors = append(t.Errors, parse.NewSyntaxError(n.Identifier.Token, fmt.Sprintf("Unable to infer type of identifier '%s'", n.Identifier.Value())))
				return nil
			}

			if array, ok = s.(*schema.Array); !ok {
				if composite, ok = s.(*schema.Composite); !ok {
					t.Errors = append(t.Errors, parse.NewSyntaxError(n.Identifier.Token, fmt.Sprintf("Type of '%s' is not a tuple or composite, subscripting not allowed", n.Identifier.Value())))
					return nil
				}
			}

			if array != nil {
				if types.IntVal(n.Subscript.(*ast.NumberNode).Val) > int64(array.Length-1) {
					t.Errors = append(t.Errors, parse.NewSyntaxError(n.Subscript.(*ast.NumberNode).Token, fmt.Sprintf("Tuple index out of bounds, '%s' has a schema of '%s'", n.Identifier.Value(), array.ToSchema())))
				}

				t.typeLookup[n] = &array.Type
			} else {
				// Ensure that the subscript is a string
				if _, ok := n.Subscript.(*ast.StringNode); !ok {
					t.Errors = append(t.Errors, parse.NewSyntaxError(n.Token, "Expected a string index for composite subscript"))
					return nil
				}

				keyName := n.Subscript.(*ast.StringNode).Val
				obj := composite.SchemaForKey(types.StringVal(keyName))

				t.typeLookup[n] = obj
			}
			t.locations[n] = n.Identifier.Token.Location

		case *ast.TimeWhenceNode, *ast.TimespanNode:
			t.typeLookup[n] = &schema.Type{Name: "int64"}
		case *ast.BinaryOpNode:
			if !t.typeForNode(n.Left).IsNumeric() || !t.typeForNode(n.Right).IsNumeric() {
				t.Errors = append(t.Errors, parse.NewSyntaxError(n.Op, "Both operands must be numeric"))
				return nil
			}

			switch n.Op.Type {
			case scanner.TOK_MINUS, scanner.TOK_PLUS, scanner.TOK_STAR:
				if strings.HasPrefix(t.typeForNode(n.Left).ToSchema(), "float") ||
					strings.HasPrefix(t.typeForNode(n.Right).ToSchema(), "float") {
					t.typeLookup[n] = &schema.Type{Name: "float64"}
				} else {
					t.typeLookup[n] = &schema.Type{Name: "int64"}
				}
			case scanner.TOK_SLASH:
				t.typeLookup[n] = &schema.Type{Name: "float64"}
			case scanner.TOK_LESS, scanner.TOK_LESS_EQ, scanner.TOK_EQ_EQ, scanner.TOK_NOT_EQ, scanner.TOK_GREATER, scanner.TOK_GREATER_EQ:
				t.typeLookup[n] = &schema.Type{Name: "boolean"}
			}
			t.locations[n] = parse.Location{Start: t.locations[n.Left].Start, End: t.locations[n.Right].End}
		case *ast.UnaryOpNode:
			if !t.typeForNode(n.Operand).IsNumeric() {
				err := fmt.Sprintf("Operator '%s' expects a numeric operand, got %s instead", n.Operator.Lexeme, t.typeForNode(n.Operand).ToSchema())
				t.Errors = append(t.Errors, parse.NewSyntaxError(parse.Token{Location: t.locations[n.Operand]}, err))
			}
			// FIXME: This is not quite correct, we should be up-casting to int if operand is uint and the sign is -
			t.typeLookup[n] = t.typeForNode(n.Operand)
			t.locations[n] = parse.Location{Start: n.Operator.Location.Start, End: t.locations[n.Operand].End}
		case *ast.TupleNode:
			var innerType schema.Object

			// Each item must have a compatible type
			for _, item := range n.Elements {
				if innerType == nil {
					innerType = t.typeForNode(item)
					continue
				}

				if (t.typeForNode(item).IsNumeric() && !innerType.IsNumeric()) ||
					(!t.typeForNode(item).IsNumeric() && innerType.IsNumeric()) {
					t.Errors = append(t.Errors, parse.NewSyntaxError(parse.Token{Location: t.locations[item]}, "Incompatible type found"))
				}

				if strings.HasPrefix(t.typeForNode(item).ToSchema(), "float") {
					innerType = t.typeForNode(item)
				}

				// FIXME: Up-sample to largest numeric
			}
			t.typeLookup[n] = &schema.Array{Type: *innerType.(*schema.Type), Length: len(n.Elements)}
			t.locations[n] = parse.Location{Start: t.locations[n.Elements[0]].Start, End: t.locations[n.Elements[len(n.Elements)-1]].End}
		case *ast.DataFunctionNode:
			t.typeLookup[n] = t.typeForNode(n.Expression)
			// Reduce must have 2 arguments
			if n.Name.Lexeme == "reduce" && len(n.Arguments) != 2 {
				t.Errors = append(t.Errors, parse.NewSyntaxError(n.Name, fmt.Sprintf("The reduce function expects 2 arguments, %d provided", len(n.Arguments))))
			}

			// Populate symbols for the next stage in our pipeline
			if n.Next != nil {
				// Ensure we have the same number of return values as the next stage's
				// arguments
				nextNumArgs := len(n.Next.Arguments)
				var argType schema.Object

				// Filter operations don't mutate the input, and simply pass it along
				if n.Name.Lexeme == "filter" {
					argType = t.symbols[n.Arguments[0].Value()]
				} else {
					if array, ok := t.typeForNode(n.Expression).(schema.Array); ok {
						if nextNumArgs == 1 {
							argType = array
						} else if nextNumArgs == array.Length {
							argType = array.Type
						} else {
							txt := fmt.Sprintf("Argument mismatch: %s stage expected %d arguments, but got %d", n.Next.Value(), nextNumArgs, array.Length)
							t.Errors = append(t.Errors, parse.NewSyntaxError(parse.Token{Location: t.locations[n.Expression]}, txt))
						}
					} else {
						argType = t.typeForNode(n.Expression)
					}
				}

				for _, arg := range n.Next.Arguments {
					t.symbols[arg.Value()] = argType
				}
			}
		case *ast.BuiltinFunctionNode:
			builtin, ok := types.LookupBuiltinFunction(n.Name.Lexeme)
			if !ok {
				t.Errors = append(t.Errors, parse.NewSyntaxError(n.Name, fmt.Sprintf("Unknown builtin function: '%s'", n.Name.Lexeme)))
				return nil
			}

			argType := t.typeForNode(n.Expression)
			retType, err := builtin.Validate(argType)

			if err != nil {
				t.Errors = append(t.Errors, parse.NewSyntaxError(parse.Token{Location: t.locations[n.Expression]}, err.Error()))
				return nil
			}

			t.typeLookup[n] = retType
		}

		return nil
	}

	switch n := node.(type) {
	case *ast.QueryNode:
		if n.DataPipeline != nil {
			var s schema.Object
			if n.Topic == nil {
				s = &schema.Type{Name: "string"}
			} else {
				topic := n.Topic.(*ast.TopicSelectorNode).Topic
				s = t.db.SchemaForTopic(topic.Lexeme)
				if s == nil {
					t.Errors = append(t.Errors, parse.NewSyntaxError(topic, "Unknown topic specified."))
					return nil
				}
			}

			t.initialType = s
			return t
		}
		return nil

	case *ast.DataPipelineNode:
		first := n.Stages[0].(*ast.DataFunctionNode)

		for _, arg := range first.Arguments {
			t.symbols[arg.Value()] = t.initialType
		}

		return t

	case *ast.NumberNode, *ast.StringNode, *ast.IdentifierNode, *ast.BinaryOpNode, *ast.UnaryOpNode, *ast.TupleNode,
		*ast.DataFunctionNode, *ast.ElementNode, *ast.BuiltinFunctionNode, *ast.TimespanNode, *ast.TimeWhenceNode:
		t.push(n)
		return t
	}

	return nil
}
