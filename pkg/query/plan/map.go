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

type MapStage struct {
	next  Stage
	root  *ast.DataFunctionNode
	input chan []WrappedEntry
	once  sync.Once
}

func MakeMapStage(node *ast.DataFunctionNode) *MapStage {
	var m MapStage

	m.input = make(chan []WrappedEntry)
	m.root = node
	return &m
}

func (m *MapStage) Chain(next Stage) {
	m.next = next
}

func (m *MapStage) Next() Stage {
	return m.next
}

func (m *MapStage) Add(entries []WrappedEntry) {
	m.input <- entries
}

func (m *MapStage) Finish() {
	m.once.Do(func() {
		close(m.input)
	})
}

func (m *MapStage) Execute() {
	for entries := range m.input {
		symbols := make(SymbolMap)

		for idx, arg := range m.root.Arguments {
			symbols[arg.Value()] = entries[idx].Value()
		}

		fn := MakeFunction(symbols)
		ast.Walk(&fn, m.root)

		var newEntries []WrappedEntry
		prototype := entries[0]
		for _, r := range fn.Result {
			newEntries = append(newEntries, prototype.Copy(r))
		}

		m.Next().Add(newEntries)
	}
	m.Next().Finish()
}
