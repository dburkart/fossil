/*
 * Copyright (c) 2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package plan

import (
	"fmt"
	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/query/ast"
	"github.com/dburkart/fossil/pkg/query/types"
	"sync"
)

type DataPipeline interface {
	Execute(entries database.Entries) database.Entries
}

type Pipeline struct {
	stages []Stage
}

func MakePipelineFromNode(node *ast.DataPipelineNode) Pipeline {
	var p Pipeline

	for _, stage := range node.Stages {
		stage, ok := stage.(*ast.DataFunctionNode)
		if !ok {
			panic("Unexpected node found in data pipeline")
		}

		switch stage.Name.Lexeme {
		case "filter":
			p.Add(MakeFilterStage(stage))
		case "map":
			p.Add(MakeMapStage(stage))
		case "reduce":
			p.Add(MakeReduceStage(stage))
		default:
			panic(fmt.Sprintf("Unsupported stage type: %s", stage.Name.Lexeme))
		}
	}
	p.Finalize()

	return p
}

func (p *Pipeline) Add(s Stage) {
	if len(p.stages) > 0 {
		p.stages[len(p.stages)-1].Chain(s)
	}
	p.stages = append(p.stages, s)
}

func (p *Pipeline) Finalize() {
	collect := MakeCollectStage()
	if len(p.stages) > 0 {
		p.stages[len(p.stages)-1].Chain(collect)
	}
	p.stages = append(p.stages, collect)
}

func (p *Pipeline) Execute(entries database.Entries) database.Entries {
	var results database.Entries
	var wg sync.WaitGroup

	// Start our pipeline stages
	for _, stage := range p.stages {
		go stage.Execute()
	}

	first := p.stages[0]
	last := p.stages[len(p.stages)-1].(*CollectStage)

	// Start our reader for output
	wg.Add(1)
	go func() {
		for result := range last.Output {
			results = append(results, result.Entry())
		}
		wg.Done()
	}()

	// Pass in everything to the first stage
	for _, entry := range entries {
		first.Add([]WrappedEntry{Wrap(entry)})
	}
	first.Finish()

	wg.Wait()
	return results
}

type WrappedEntry struct {
	entry *database.Entry
	val   types.Value
}

func Wrap(entry database.Entry) WrappedEntry {
	return WrappedEntry{entry: &entry}
}

func (w *WrappedEntry) Value() types.Value {
	if w.val != nil {
		return w.val
	}

	w.val = types.MakeFromEntry(*w.entry)
	return w.val
}

func (w *WrappedEntry) Copy(v types.Value) WrappedEntry {
	return WrappedEntry{entry: w.entry, val: v}
}

func (w *WrappedEntry) SetTopic(t string) {
	w.entry.Topic = t
}

func (w *WrappedEntry) Entry() database.Entry {
	if w.val == nil {
		return *w.entry
	}
	e, err := types.EntryFromValue(w.Value())
	if err != nil {
		e.Schema = "string"
		e.Data = []byte(err.Error())
	}
	e.Time = w.entry.Time
	e.Topic = w.entry.Topic
	return e
}

type Stage interface {
	Chain(Stage)
	Next() Stage
	Add(entries []WrappedEntry)
	Finish()
	Execute()
}

type CollectStage struct {
	Output chan WrappedEntry
	once   sync.Once
}

func MakeCollectStage() *CollectStage {
	return &CollectStage{
		Output: make(chan WrappedEntry),
	}
}

func (c *CollectStage) Chain(s Stage) { return }
func (c *CollectStage) Next() Stage   { return nil }
func (c *CollectStage) Execute()      {}

func (c *CollectStage) Finish() {
	c.once.Do(func() {
		close(c.Output)
	})
}

func (c *CollectStage) Add(entries []WrappedEntry) {
	for _, entry := range entries {
		c.Output <- entry
	}
}
