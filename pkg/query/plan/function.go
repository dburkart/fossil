/*
 * Copyright (c) 2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package plan

import (
	"fmt"
	"github.com/dburkart/fossil/pkg/query/ast"
	"github.com/dburkart/fossil/pkg/query/types"
)

type SymbolMap map[string]types.Value

type Function struct {
	Result  []types.Value
	symbols SymbolMap
	results map[ast.ASTNode]types.Value
	stack   []ast.ASTNode
}

func MakeFunction(symbols SymbolMap) Function {
	return Function{symbols: symbols, results: make(map[ast.ASTNode]types.Value)}
}

// FIXME: Factor out stack into it's own thing
func (f *Function) push(node ast.ASTNode) {
	f.stack = append(f.stack, node)
}

func (f *Function) pop() ast.ASTNode {
	if len(f.stack) == 0 {
		return nil
	}
	node := f.stack[len(f.stack)-1]
	f.stack = f.stack[:len(f.stack)-1]
	return node
}

func (f *Function) Visit(node ast.ASTNode) ast.Visitor {
	if node == nil {
		switch n := f.pop().(type) {
		case *ast.IdentifierNode:
			result, ok := f.symbols[n.Value()]
			if !ok {
				panic(fmt.Sprintf("Symbol %s did not resolve!", n.Value()))
			}

			f.results[n] = result
		case *ast.NumberNode:
			f.results[n] = n.Val
		case *ast.StringNode:
			f.results[n] = n.Val
		case *ast.UnaryOpNode:
			f.results[n] = types.UnaryOp(n.Operator, f.results[n.Operand])
		case *ast.BinaryOpNode:
			f.results[n] = types.BinaryOp(f.results[n.Left], n.Op, f.results[n.Right])
		case *ast.DataFunctionNode:
			// FIXME: Handle tuples
			f.Result = append(f.Result, f.results[n.Expression])
		}

		return nil
	}

	switch n := node.(type) {
	case *ast.DataFunctionNode, *ast.IdentifierNode, *ast.NumberNode, *ast.UnaryOpNode, *ast.BinaryOpNode:
		f.push(n)
		return f
	}

	return nil
}