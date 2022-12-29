/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package schema

import "fmt"

type Object interface {
	Validate([]byte) bool
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
