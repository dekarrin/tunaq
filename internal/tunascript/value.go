package tunascript

import (
	"fmt"
	"strconv"
	"strings"
)

// ValueType is the type of a value in tuna-script.
type ValueType int

const (
	Str ValueType = iota
	Num
	Bool
)

// Value is a value parsed from tunascript.
type Value struct {
	v string
	t ValueType
}

// Type returns the type of the Value.
func (v Value) Type() ValueType {
	return v.t
}

// Returns the current value of v as a bool. If it is not already a bool type,
// it is cast to one.
func (v Value) Bool() bool {
	switch v.Type() {
	case Bool:
		b, err := strconv.ParseBool(v.v)
		if err != nil {
			// should never happen
			panic(fmt.Sprintf("wrong type: %q is not bool-able", v.v))
		}
		return b
	case Num:
		numV := v.Num()
		return numV != 0
	case Str:
		return !(strings.ToUpper(v.v) == "FALSE" || v.v == "")
	default:
		// should never happen
		panic(fmt.Sprintf("unknown tunascript type: %v", v.Type()))
	}
}

// Returns the current value of v as a num. If it is not already a num type,
// it is cast to one.
func (v Value) Num() int {
	var n int
	var err error

	switch v.Type() {
	case Bool:
		boolV := v.Bool()
		if boolV {
			return 1
		}
		return 0
	case Num:
		n, err = strconv.Atoi(v.v)
		if err != nil {
			// should never happen
			panic(fmt.Sprintf("wrong type: %q is not int-able", v.v))
		}
		return n
	case Str:
		n, err = strconv.Atoi(v.v)
		if err != nil {
			// it's not an int, so just default to filled = 1, empty = 0
			if v.v == "" {
				return 0
			} else {
				return 1
			}
		}
		return n
	default:
		// should never happen
		panic(fmt.Sprintf("unknown tunascript type: %v", v.Type()))
	}
}

// Str returns the current value of v as a string. If it is not already a str
// type, its result is cast to one first.
func (v Value) Str() string {
	var s string

	switch v.Type() {
	case Bool:
		return fmt.Sprintf("%t", v.Bool())
	case Num:
		return fmt.Sprintf("%d", v.Num())
	case Str:
		return s
	default:
		// should never happen
		panic(fmt.Sprintf("unknown tunascript type: %v", v.Type()))
	}
}

// NewStr returns a Value of Str type whose value is the given string.
func NewStr(s string) Value {
	return Value{v: s, t: Str}
}

// NewBool returns a Value of Bool type whose value is the given bool.
func NewBool(b bool) Value {
	return Value{
		v: fmt.Sprintf("%t", b),
		t: Bool,
	}
}

// NewNum returns a Value of Num type whose value is the given int.
func NewNum(n int) Value {
	return Value{
		v: fmt.Sprintf("%d", n),
		t: Num,
	}
}
