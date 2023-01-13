/*
 * Copyright (c) 2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package plan

import (
	"github.com/dburkart/fossil/pkg/query/ast"
	"github.com/dburkart/fossil/pkg/query/types"
	"sync"
)

type FilterStage struct {
	next  Stage
	root  *ast.DataFunctionNode
	input chan []WrappedEntry
	once  sync.Once
}

func MakeFilterStage(node *ast.DataFunctionNode) *FilterStage {
	var f FilterStage

	f.input = make(chan []WrappedEntry)
	f.root = node
	return &f
}

func (f *FilterStage) Chain(next Stage) {
	f.next = next
}

func (f *FilterStage) Next() Stage {
	return f.next
}

func (f *FilterStage) Add(entries []WrappedEntry) {
	f.input <- entries
}

func (f *FilterStage) Finish() {
	f.once.Do(func() {
		close(f.input)
	})
}

func (f *FilterStage) Execute() {
	for entries := range f.input {
		symbols := make(SymbolMap)

		for idx, arg := range f.root.Arguments {
			symbols[arg.Value()] = entries[idx].Value()
		}

		fn := MakeFunction(symbols)
		ast.Walk(&fn, f.root)

		allowed := types.BooleanVal(fn.Result[0])

		if allowed {
			f.Next().Add(entries)
		}
	}
	f.Next().Finish()
}
