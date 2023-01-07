/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package schema

import (
	"fmt"
)

type Object interface {
	Validate([]byte) bool
	ToSchema() string
}

type (
	Type struct {
		Name string
	}

	Array struct {
		Length int
		Type   Type
	}

	Composite struct {
		Keys   []string
		Values []Object
	}
)

func TypeFromString(input string) Object {
	switch input {
	case "boolean":
		return &Type{
			Name: "boolean",
		}
	case "int8":
		return &Type{
			Name: "int8",
		}
	case "uint8":
		return &Type{
			Name: "uint8",
		}
	case "int16":
		return &Type{
			Name: "int16",
		}
	case "uint16":
		return &Type{
			Name: "uint16",
		}
	case "int32":
		return &Type{
			Name: "int32",
		}
	case "uint32":
		return &Type{
			Name: "uint32",
		}
	case "int64":
		return &Type{
			Name: "int64",
		}
	case "uint64":
		return &Type{
			Name: "uint64",
		}
	case "float32":
		return &Type{
			Name: "float32",
		}
	case "float64":
		return &Type{
			Name: "float64",
		}
	case "string":
		return &Type{
			Name: "string",
		}
	case "binary":
		return &Type{
			Name: "binary",
		}
	}
	panic("unknown schema type")
}

func (t Type) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, t.ToSchema())), nil
}

func (t Type) Size() int {
	switch {
	case t.Name == "boolean":
		return 1
	case t.Name == "int8" || t.Name == "uint8":
		return 1
	case t.Name == "int16" || t.Name == "uint16":
		return 2
	case t.Name == "int32" || t.Name == "uint32":
		return 4
	case t.Name == "int64" || t.Name == "uint64":
		return 8
	case t.Name == "float32":
		return 4
	case t.Name == "float64":
		return 8
	case t.Name == "string":
		return 4
	case t.Name == "binary":
		return 4
	}
	return 0
}

func (t Type) Validate(val []byte) bool {
	switch {
	case t.Name == "boolean":
		if len(val) != 1 {
			return false
		}
	case t.Name == "int8" || t.Name == "uint8":
		if len(val) != 1 {
			return false
		}
	case t.Name == "int16" || t.Name == "uint16":
		if len(val) != 2 {
			return false
		}
	case t.Name == "int32" || t.Name == "uint32":
		if len(val) != 4 {
			return false
		}
	case t.Name == "int64" || t.Name == "uint64":
		if len(val) != 8 {
			return false
		}
	case t.Name == "float32":
		if len(val) != 4 {
			return false
		}
	case t.Name == "float64":
		if len(val) != 8 {
			return false
		}
	}

	return true
}

func (t Type) ToSchema() string {
	return t.Name
}

func (a Array) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, a.ToSchema())), nil
}

func (a Array) Size() int {
	return a.Length * a.Type.Size()
}

func (a Array) Validate(val []byte) bool {
	typeSize := a.Type.Size()

	// string / binary is not allowed.
	if a.Type.Name == "string" || a.Type.Name == "binary" {
		panic(fmt.Sprintf("invalid type found in array: %s", a.Type.Name))
	}

	if len(val) != a.Length*typeSize {
		return false
	}

	return true
}

func (a Array) ToSchema() string {
	return fmt.Sprintf("[%d]%s", a.Length, a.Type.ToSchema())
}

func (c Composite) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, c.ToSchema())), nil
}

func (c Composite) Validate(val []byte) bool {
	var size int
	hasString := false
	for _, val := range c.Values {
		switch val.(type) {
		case Type:
			if val.(Type).Name == "string" {
				hasString = true
			}
			size += val.(Type).Size()
		case Array:
			if val.(Array).Type.Name == "string" {
				hasString = true
			}
			size += val.(Array).Size()
		}
	}

	if hasString {
		return len(val) >= size
	}
	return len(val) == size
}

func (c Composite) ToSchema() string {
	var schema string

	schema += "{"

	for idx := range c.Keys {
		key := c.Keys[idx]
		val := c.Values[idx].ToSchema()

		schema += fmt.Sprintf(`'%s':%s,`, key, val)
	}

	schema += "}"
	return schema
}
