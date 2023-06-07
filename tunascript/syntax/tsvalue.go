package syntax

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

type ValueType int

const (
	Int ValueType = iota
	Float
	String
	Bool
)

// TSValue is a value in Tunascript. It is formally quad-typed, though official
// documentation does not distinguish between ints and floats (they are called
// "numbers"). If math with a float is performed, the result is float.
//
// Only two of the values in a TSValue will be valid: vType and one other,
// depending on the value of vType.
type TSValue struct {
	vType ValueType
	f     float64
	i     int
	s     string
	b     bool
}

// TSValueOf creates a TSValueFrom of the appropriate type based on the
// arguments. The argument must be an int, float64, bool, or string.
func TSValueOf(v any) TSValue {
	switch typedV := v.(type) {
	case int:
		return TSValue{vType: Int, i: typedV}
	case float64:
		return TSValue{vType: Float, f: typedV}
	case bool:
		return TSValue{vType: Bool, b: typedV}
	case string:
		return TSValue{vType: String, s: typedV}
	default:
		panic("should never happen")
	}
}

func (v TSValue) Equal(o any) bool {
	other, ok := o.(TSValue)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*TSValue)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	// if one of the operands is a string, do string comparison
	if v.Type() == String || other.Type() == String {
		s1 := v.String()
		s2 := other.String()
		return s1 == s2
	}

	// if one of the operands is a bool, they will be compared as bools
	if v.Type() == Bool || other.Type() == Bool {
		b1 := v.Bool()
		b2 := other.Bool()

		return b1 == b2
	}

	// if one of the operands is a float, they will be compared as floats
	if v.Type() == Float || other.Type() == Float {
		f1 := v.Float()
		f2 := other.Float()

		return f1 == f2
	}

	// finally, they must both be ints. do int comparison
	return v.Int() == other.Int()
}

func (v TSValue) IsNumber() bool {
	return v.vType == Int || v.vType == Float
}

func (v TSValue) Type() ValueType {
	return v.vType
}

func (v TSValue) String() string {
	switch v.vType {
	case Float:
		str := fmt.Sprintf("%.9f", v.f)
		// remove extra 0's...
		str = strings.TrimRight(str, "0")
		// ...but there should be at least one 0 if nothing else
		if strings.HasSuffix(str, ".") {
			str = str + "0"
		}
		return str
	case Int:
		return fmt.Sprintf("%d", v.i)
	case String:
		return v.s
	case Bool:
		if v.b {
			return "on"
		}
		return "off"
	default:
		panic("unrecognized TSValue type")
	}
}

// Escaped returns the String() value but with all tunascript-sensitive
// characters escaped, including whitespace at the start or end, as it would
// appear in tunascript source.
func (v TSValue) Escaped() string {
	s := v.String()

	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "@", `\@`)
	s = strings.ReplaceAll(s, ",", `\,`)
	s = strings.ReplaceAll(s, "+", `\+`)
	s = strings.ReplaceAll(s, "-", `\-`)
	s = strings.ReplaceAll(s, "<", `\<`)
	s = strings.ReplaceAll(s, ">", `\>`)
	s = strings.ReplaceAll(s, "!", `\!`)
	s = strings.ReplaceAll(s, "=", `\=`)
	s = strings.ReplaceAll(s, "*", `\*`)
	s = strings.ReplaceAll(s, "/", `\/`)
	s = strings.ReplaceAll(s, "&", `\&`)
	s = strings.ReplaceAll(s, "|", `\|`)
	s = strings.ReplaceAll(s, "(", `\(`)
	s = strings.ReplaceAll(s, ")", `\)`)
	s = strings.ReplaceAll(s, "$", `\$`)

	// now take care of trailing space
	sRunes := []rune(s)

	if len(sRunes) > 0 {
		if unicode.IsSpace(sRunes[0]) {
			sRunes = append([]rune{'\\'}, sRunes...)
		}

		if len(sRunes) > 1 {
			lastChar := sRunes[len(sRunes)-1]
			if unicode.IsSpace(lastChar) {
				sRunes = sRunes[:len(sRunes)-1]
				sRunes = append(sRunes, '\\', lastChar)
			}
		}
	}

	s = string(sRunes)

	return s
}

// Quoted returns the String() value in between two "@" chars with any "@" chars
// within it escaped (as well as any existing backslashes)
func (v TSValue) Quoted() string {
	s := v.String()

	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "@", `\@`)

	return "@" + s + "@"
}

// if it is a float, rounding will occur. If it is a string, attempts to parse
// it. if unparsable, returns 0. Bool true is 1, bool false is 0.
func (v TSValue) Int() int {
	switch v.vType {
	case Float:
		val := math.Round(v.f)
		return int(val)
	case Int:
		return v.i
	case String:
		iVal, err := strconv.Atoi(v.s)
		if err != nil {
			return 0
		}
		return iVal
	case Bool:
		if v.b {
			return 1
		}
		return 0
	default:
		panic("unrecognized TSValue type")
	}
}

// If it is a string, attempts to parse it. if unparsable, returns 0.0. Bool
// true is 1.0, bool false is 0.0.
func (v TSValue) Float() float64 {
	switch v.vType {
	case Float:
		return v.f
	case Int:
		return float64(v.i)
	case String:
		fVal, err := strconv.ParseFloat(v.s, 64)
		if err != nil {
			return 0.0
		}
		return fVal
	case Bool:
		if v.b {
			return 1.0
		}
		return 0.0
	default:
		panic("unrecognized TSValue type")
	}
}

// if it is a number, true if non-zero, false if zero. if a string, true if
// string is not empty.
func (v TSValue) Bool() bool {
	switch v.vType {
	case Float:
		return v.f != 0.0
	case Int:
		return v.i != 0
	case String:
		return len(v.s) > 0
	case Bool:
		return v.b
	default:
		panic("unrecognized TSValue type")
	}
}
