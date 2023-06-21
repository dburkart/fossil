/*
 * Copyright (c) 2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package ast

import (
	"reflect"
	"strings"
)

type Dumper struct {
	Output string
	indent int
}

func (d *Dumper) Visit(node ASTNode) Visitor {
	if node == nil {
		d.indent -= 1
		return nil
	}

	level := strings.Repeat("    ", d.indent)

	value := node.Value()
	switch t := node.(type) {
	case *TopicSelectorNode:
		value = "in " + t.Topic.Lexeme
	case *DataFunctionNode:
		var args string
		for _, a := range t.Arguments {
			args += a.Value() + ", "
		}
		value = "name(" + node.Value() + ") args(" + args[:len(args)-2] + ")"
	case *ElementNode:
		value = t.Identifier.Value() + "[" + t.Subscript.Value() + "]"
	}

	t := reflect.TypeOf(node)
	output := level + t.Elem().Name() + "[" + value + "]" + "\n"

	d.Output += output
	d.indent += 1

	return d
}
