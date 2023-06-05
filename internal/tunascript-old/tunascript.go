package tunascript

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// tunascript execution engine

// FuncCall is the implementation of a function in tunaquest. It receives some
// number of values defined externally and returns a Value of the appropriate
// type.
type FuncCall func(val []Value) Value

// Function defines a function that can be executed in tunascript.
type Function struct {
	// Name is the name of the function. It would be called with $Name(). Name
	// is case-insensitive and must follow identifier naming rules ([A-Z0-9_]+)
	Name string

	// RequiredArgs is the number of required arguments. This many arguments is
	// gauratneed to be passed to Call.
	RequiredArgs int

	// OptionalArgs is the number of optional arguments. Up to the
	OptionalArgs int

	// Call is a function point to the golang implementation of the function. It
	// is guaranteed to receive RequiredArgs, and may receive up to OptionalArgs
	// additional args.
	Call FuncCall

	// SideEffects tells whether the function has side-effects. Certain contexts
	// such as within an $IF() may restrict the execution of side-effect
	// functions.
	SideEffects bool
}

// Flag is a variable in the engine.
type Flag struct {
	Name string
	Value
}

// Interpreter should not be used directly, use NewInterpreter.
type Interpreter struct {
	fn    map[string]Function
	opFn  map[string]Function
	flags map[string]*Flag
	world WorldInterface
}

type WorldInterface interface {

	// InInventory returns whether the given label Item is in the player
	// inventory.
	InInventory(label string) bool

	// Move moves the label to the dest. The label can be an NPC or an Item. If
	// label is "@PLAYER", the player will be moved. Returns whether the thing
	// moved.
	Move(label string, dest string) bool

	// Output prints the given string. Returns whether it did successfully.
	Output(s string) bool
}

func NewInterpreter(w WorldInterface) Interpreter {
	inter := Interpreter{
		fn:    make(map[string]Function),
		opFn:  make(map[string]Function),
		flags: make(map[string]*Flag),
		world: w,
	}

	inter.fn["ADD"] = Function{Name: "ADD", RequiredArgs: 2, Call: builtIn_Add}
	inter.fn["SUB"] = Function{Name: "SUB", RequiredArgs: 2, Call: builtIn_Sub}
	inter.fn["MULT"] = Function{Name: "MULT", RequiredArgs: 2, Call: builtIn_Mult}
	inter.fn["DIV"] = Function{Name: "DIV", RequiredArgs: 2, Call: builtIn_Div}
	inter.fn["NEG"] = Function{Name: "NEG", RequiredArgs: 1, Call: builtIn_Neg}
	inter.fn["OR"] = Function{Name: "OR", RequiredArgs: 2, Call: builtIn_Or}
	inter.fn["AND"] = Function{Name: "AND", RequiredArgs: 2, Call: builtIn_And}
	inter.fn["NOT"] = Function{Name: "NOT", RequiredArgs: 1, Call: builtIn_Not}

	inter.fn["FLAG_ENABLED"] = Function{Name: "FLAG_ENABLED", RequiredArgs: 1, Call: inter.builtIn_FlagEnabled}
	inter.fn["FLAG_DISABLED"] = Function{Name: "FLAG_DISABLED", RequiredArgs: 1, Call: inter.builtIn_FlagDisabled}
	inter.fn["FLAG_IS"] = Function{Name: "FLAG_IS", RequiredArgs: 2, Call: inter.builtIn_FlagIs}
	inter.fn["FLAG_LESS_THAN"] = Function{Name: "FLAG_LESS_THAN", RequiredArgs: 2, Call: inter.builtIn_FlagLessThan}
	inter.fn["FLAG_GREATER_THAN"] = Function{Name: "FLAG_GREATER_THAN", RequiredArgs: 2, Call: inter.builtIn_FlagGreaterThan}
	inter.fn["IN_INVEN"] = Function{Name: "IN_INVEN", RequiredArgs: 1, Call: inter.builtIn_InInven}
	inter.fn["ENABLE"] = Function{Name: "ENABLE", RequiredArgs: 1, Call: inter.builtIn_Enable, SideEffects: true}
	inter.fn["DISABLE"] = Function{Name: "DISABLE", RequiredArgs: 1, Call: inter.builtIn_Disable, SideEffects: true}
	inter.fn["TOGGLE"] = Function{Name: "DISABLE", RequiredArgs: 1, Call: inter.builtIn_Toggle, SideEffects: true}
	inter.fn["INC"] = Function{Name: "INC", RequiredArgs: 1, OptionalArgs: 1, Call: inter.builtIn_Inc, SideEffects: true}
	inter.fn["DEC"] = Function{Name: "DEC", RequiredArgs: 1, OptionalArgs: 1, Call: inter.builtIn_Dec, SideEffects: true}
	inter.fn["SET"] = Function{Name: "SET", RequiredArgs: 2, Call: inter.builtIn_Set, SideEffects: true}
	inter.fn["MOVE"] = Function{Name: "MOVE", RequiredArgs: 2, Call: inter.builtIn_Move, SideEffects: true}
	inter.fn["OUTPUT"] = Function{Name: "OUTPUT", RequiredArgs: 1, Call: inter.builtIn_Output, SideEffects: true}

	inter.opFn[literalStrOpMinus] = Function{Name: "OP_MINUS", RequiredArgs: 1, OptionalArgs: 1, Call: func(val []Value) Value {
		if len(val) == 2 {
			return builtIn_Sub(val)
		} else if len(val) == 1 {
			return builtIn_Neg(val)
		} else {
			panic("incorrect number of args to predefined operator function")
		}
	}}
	inter.opFn[literalStrOpPlus] = Function{Name: "OP_PLUS", RequiredArgs: 2, Call: builtIn_Add}
	inter.opFn[literalStrOpMultiply] = Function{Name: "OP_TIMES", RequiredArgs: 2, Call: builtIn_Mult}
	inter.opFn[literalStrOpDivide] = Function{Name: "OP_DIVIDED_BY", RequiredArgs: 2, Call: builtIn_Div}
	inter.opFn[literalStrOpIncset] = Function{Name: "OP_PLUS_ASSIGN", RequiredArgs: 2, Call: inter.builtIn_Inc, SideEffects: true}
	inter.opFn[literalStrOpDecset] = Function{Name: "OP_MINUS_ASSIGN", RequiredArgs: 2, Call: inter.builtIn_Dec, SideEffects: true}
	inter.opFn[literalStrOpNot] = Function{Name: "OP_NOT", RequiredArgs: 1, Call: builtIn_Not}
	inter.opFn[literalStrOpSet] = Function{Name: "OP_ASSIGN", RequiredArgs: 2, Call: inter.builtIn_Set, SideEffects: true}
	inter.opFn[literalStrOpIs] = Function{Name: "OP_EQ", RequiredArgs: 2, Call: builtIn_EQ}
	inter.opFn[literalStrOpIsNot] = Function{Name: "OP_NE", RequiredArgs: 2, Call: builtIn_NE}
	inter.opFn[literalStrOpGreaterThan] = Function{Name: "OP_GT", RequiredArgs: 2, Call: builtIn_GT}
	inter.opFn[literalStrOpGreaterThanIs] = Function{Name: "OP_GE", RequiredArgs: 2, Call: builtIn_GE}
	inter.opFn[literalStrOpLessThan] = Function{Name: "OP_LT", RequiredArgs: 2, Call: builtIn_LT}
	inter.opFn[literalStrOpLessThanIs] = Function{Name: "OP_LE", RequiredArgs: 2, Call: builtIn_LE}
	inter.opFn[literalStrOpOr] = Function{Name: "OP_OR", RequiredArgs: 2, Call: builtIn_Or}
	inter.opFn[literalStrOpAnd] = Function{Name: "OP_AND", RequiredArgs: 2, Call: builtIn_And}
	return inter
}

// AddFlag adds a flag to the interpreter's flag store, with an initial value.
func (inter Interpreter) AddFlag(label string, val string) error {
	label = strings.ToUpper(label)

	if len(label) < 1 {
		return fmt.Errorf("label %q does not match pattern /[A-Z0-9_]+/", label)
	}

	for _, ch := range label {
		if !('A' <= ch && ch <= 'Z') && !('0' <= ch && ch <= '9') && ch != '_' {
			return fmt.Errorf("label %q does not match pattern /[A-Z0-9_]+/", label)
		}
	}

	inter.flags[label] = &Flag{
		Name:  label,
		Value: parseUntypedValString(val),
	}

	return nil
}

// ListFlags returns a list of all flags, sorted.
func (inter Interpreter) ListFlags() []string {
	flags := make([]string, len(inter.flags))
	curFlagIdx := 0
	for k := range inter.flags {
		flags[curFlagIdx] = k
	}

	sort.Strings(flags)
	return flags
}

// GetFlag gets the give flag's value. If it is unset, it will be "".
func (inter Interpreter) GetFlag(label string) string {
	label = strings.ToUpper(label)

	flag, ok := inter.flags[label]
	if !ok {
		return ""
	}
	return flag.String()
}

// Evaluate interprets the tunaquest expression in the given string. If there is an
// error in the code, it is returned as a non-nil error, otherwise the output of
// evaluating the expression is returned as a string.
func (inter Interpreter) Evaluate(s string) (string, error) {
	// lexical analysis
	tokens, err := Lex(s)
	if err != nil {
		return "", err
	}

	// syntactic analysis
	ast, err := Parse(tokens)
	if err != nil {
		return "", err
	}

	vals, err := inter.invoke(ast, false)
	if err != nil {
		return "", err
	}

	strVals := make([]string, len(vals))
	for i := range vals {
		strVals[i] = vals[i].Str()
	}

	return strings.Join(strVals, " "), nil
}

func parseUntypedValString(valStr string) Value {
	srcUpper := strings.ToUpper(valStr)
	if srcUpper == "TRUE" || srcUpper == "YES" || srcUpper == "ON" {
		return NewBool(true)
	} else if srcUpper == "FALSE" || srcUpper == "NO" || srcUpper == "OFF" {
		return NewBool(false)
	}

	intVal, err := strconv.Atoi(valStr)
	if err == nil {
		return NewNum(intVal)
	}

	return NewStr(valStr)
}
