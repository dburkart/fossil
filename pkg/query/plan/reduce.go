/*
 * Copyright (c) 2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package plan

import (
	"github.com/dburkart/fossil/pkg/query/ast"
	"sync"
)

type ReduceStage struct {
	next  Stage
	root  *ast.DataFunctionNode
	input chan []WrappedEntry
	once  sync.Once
}

func MakeReduceStage(node *ast.DataFunctionNode) *ReduceStage {
	var r ReduceStage

	r.input = make(chan []WrappedEntry)
	r.root = node
	return &r
}

func (r *ReduceStage) Chain(next Stage) {
	r.next = next
}

func (r *ReduceStage) Next() Stage {
	return r.next
}

func (r *ReduceStage) Add(entries []WrappedEntry) {
	r.input <- entries
}

func (r *ReduceStage) Finish() {
	r.once.Do(func() {
		close(r.input)
	})
}

func (r *ReduceStage) Execute() {
	var b []WrappedEntry
	for {
		a := <-r.input

		if b == nil {
			b = <-r.input
		}

		if a == nil {
			r.Next().Add(b)
			break
		}

		if b == nil {
			r.Next().Add(a)
			break
		}

		symbols := make(SymbolMap)
		symbols[r.root.Arguments[0].Value()] = a[0].Value()
		symbols[r.root.Arguments[1].Value()] = b[0].Value()

		fn := MakeFunction(symbols)
		ast.Walk(&fn, r.root)

		entry := a[0].Copy(fn.Result[0])
		entry.SetTopic("N/A")
		b = []WrappedEntry{a[0].Copy(fn.Result[0])}

	}
	r.Next().Finish()
}
