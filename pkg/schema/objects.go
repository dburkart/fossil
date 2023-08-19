/*
 * Copyright (c) 2022-2023, Dana Burkart <dana.burkart@gmail.com>
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
	IsNumeric() bool
}

type (
	Unknown struct{}

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

func (u Unknown) Validate(_ []byte) bool { return false }
func (u Unknown) ToSchema() string       { return "Unknown" }
func (u Unknown) IsNumeric() bool        { return false }

func (t Type) ToSchema() string {
	return t.Name
}

func (t Type) IsNumeric() bool {
	switch t.Name {
	case "int8", "uint8", "int16", "uint16", "int32", "uint32", "int64", "uint64", "float32", "float64":
		return true
	default:
		return false
	}
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

func (a Array) IsNumeric() bool {
	return false
}

func (c Composite) Validate(val []byte) bool {
	var size int
	hasString := false
	for _, val := range c.Values {
		switch t := val.(type) {
		case *Type:
			if t.Name == "string" {
				hasString = true
			}
			size += t.Size()
		case *Array:
			if t.Type.Name == "string" {
				hasString = true
			}
			size += t.Size()
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

		schema += fmt.Sprintf(`"%s":%s,`, key, val)
	}

	schema += "}"
	return schema
}

func (c Composite) IsNumeric() bool {
	return false
}

func (c Composite) SchemaForKey(key string) Object {
	for i, v := range c.Keys {
		if v == key {
			return c.Values[i]
		}
	}

	return Unknown{}
}
