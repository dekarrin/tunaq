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

// Value is a value in Tunascript. It is formally quad-typed, though official
// documentation does not distinguish between ints and floats (they are called
// "numbers"). If math with a float is performed, the result is float.
//
// Only two of the values in a Value will be valid: vType and one other,
// depending on the value of vType.
//
// Some operations may cause coercion to be performed so that all operands are
// of compatible types. In this case, it is always the type of the Value that
// the method is being called on that will determine the final result.
type Value struct {
	vType ValueType
	f     float64
	i     int
	s     string
	b     bool
}

// ValueOf creates a TSValueFrom of the appropriate type based on the
// arguments. The argument must be an int, float64, bool, or string.
func ValueOf(v any) Value {
	switch typedV := v.(type) {
	case int:
		return Value{vType: Int, i: typedV}
	case float64:
		return Value{vType: Float, f: typedV}
	case bool:
		return Value{vType: Bool, b: typedV}
	case string:
		return Value{vType: String, s: typedV}
	default:
		panic("should never happen")
	}
}

func (v Value) Equal(o any) bool {
	other, ok := o.(Value)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*Value)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	// if left operand is a string, do string comparison
	if v.Type() == String {
		return v.String() == other.String()
	}

	// if left operand is a bool, they will be compared as bools
	if v.Type() == Bool {
		return v.Bool() == other.Bool()
	}

	// if left operand is a float, they will be compared as floats
	if v.Type() == Float {
		return v.Float() == other.Float()
	}

	// finally, they must both be ints. do int comparison
	return v.Int() == other.Int()
}

// GreaterThan returns whether v is greater than v2. Both are interpreted as
// numeric. The result will always be type Bool.
func (v Value) GreaterThan(v2 Value) Value {
}

// Add returns the result of adding v2 to v.
//
// This function performs type coercion on the arguments, in the following
// order: If v is a String, both are converted to String and concatenated
// to produce the result. Otherwise, numeric addition is performed, and if one
// of the arguments is a Bool, the typical Int() value from a Bool is used. If
// either argument is a Float type, the result will also be of type float.
func (v Value) Add(v2 Value) Value {
	if v.Type() == String {
		return ValueOf(v.String() + v2.String())
	}

	// not strictly 'coercion' here, we need to know the type of the result so
	// we check both.
	if v.Type() == Float || v2.Type() == Float {
		return ValueOf(v.Float() + v2.Float())
	}

	return ValueOf(v.Int() + v2.Int())
}

// Subtract returns the result of subtracting v2 from v. The result will always
// be numeric. If either argument is of type Float, the result will be of type
// Float, otherwise it will be Int.
func (v Value) Subtract(v2 Value) Value {
	if v.Type() == Float || v2.Type() == Float {
		return ValueOf(v.Float() - v2.Float())
	}

	return ValueOf(v.Int() - v2.Int())
}

// Multiply returns the result of multiplying v by v2. If v is a String, then it
// is repeated v2 times and the resulting String is returned (in this case, the
// greater of v2 or 0 is used as the repetitions). If either argument is a
// Float, the result will be of type Float, otherwise it will be Int.
func (v Value) Multiply(v2 Value) Value {
	if v.Type() == String {
		str := v.String()
		result := ""
		times := v2.Int()
		for i := 0; i < times; i++ {
			result += str
		}
		return ValueOf(result)
	}

	if v.Type() == Float || v2.Type() == Float {
		return ValueOf(v.Float() * v2.Float())
	}

	return ValueOf(v.Int() * v2.Int())
}

// Divide returns the result of dividing v by v2. The result will always be
// numeric. If either argument is of type Float, the result will be of type
// Float. If the operation is performed with Ints but the result is fractional,
// the result will be Float. Otherwise the result will be Int.
func (v Value) Divide(v2 Value) Value {
	if v.Type() == Float || v2.Type() == Float {
		return ValueOf(v.Float() / v2.Float())
	}

	i1 := v.Int()
	i2 := v.Int()

	if i1%i2 != 0 {
		// division results in remainder, do float math instead
		return ValueOf(v.Float() / v2.Float())
	}

	return ValueOf(v.Int() / v2.Int())
}

func (v Value) IsNumber() bool {
	return v.vType == Int || v.vType == Float
}

func (v Value) Type() ValueType {
	return v.vType
}

func (v Value) String() string {
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
			return "ON"
		}
		return "OFF"
	default:
		panic("unrecognized TSValue type")
	}
}

// Escaped returns the String() value but with all tunascript-sensitive
// characters escaped, including whitespace at the start or end, as it would
// appear in tunascript source.
func (v Value) Escaped() string {
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
func (v Value) Quoted() string {
	s := v.String()

	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "@", `\@`)

	return "@" + s + "@"
}

// if it is a float, rounding will occur. If it is a string, attempts to parse
// it. if unparsable, returns 0. Bool true is 1, bool false is 0.
func (v Value) Int() int {
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
func (v Value) Float() float64 {
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
func (v Value) Bool() bool {
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
