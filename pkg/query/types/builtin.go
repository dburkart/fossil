/*
 * Copyright (c) 2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package types

import (
	"errors"
	"github.com/dburkart/fossil/pkg/schema"
)

func LookupBuiltinFunction(name string) (b Builtin, ok bool) {
	builtinMap := map[string]Builtin{
		"max": BuiltinMax{},
	}
	b, ok = builtinMap[name]
	return
}

type Builtin interface {
	Name() string
	Validate(input schema.Object) (schema.Object, error)
	Execute(input Value) Value
}

type BuiltinMax struct{}

func (b BuiltinMax) Name() string { return "max" }

func (b BuiltinMax) Validate(input schema.Object) (schema.Object, error) {
	switch t := input.(type) {
	case *schema.Array:
		if t.Type.IsNumeric() {
			return t.Type, nil
		}
		return nil, errors.New("max expects arguments to be numeric")
	default:
		return nil, errors.New("expected multiple values as input to max")
	}
}

func (b BuiltinMax) Execute(input Value) Value {
	maxValue := MakeInt(0)

	for _, v := range TupleVal(input) {
		maxValue, v = upcast(maxValue, v)

		switch b := v.(type) {
		case intVal:
			a := maxValue.(intVal)
			if b > a {
				maxValue = b
			}
		case floatVal:
			a := maxValue.(floatVal)
			if b > a {
				maxValue = b
			}
		}
	}

	return maxValue
}
