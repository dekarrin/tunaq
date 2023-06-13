package tunascript

import (
	"strings"

	"github.com/dekarrin/tunaq/tunascript/syntax"
)

type funcImpl func(args []Value) Value

type funcInfo struct {
	def  syntax.Function
	call funcImpl
}

func unaryImpl(fname string, impl func(v Value) Value) funcInfo {
	return funcInfo{
		def: syntax.BuiltInFunctions[fname],
		call: func(args []Value) Value {
			return impl(args[0])
		},
	}
}

func binaryImpl(fname string, impl func(v Value, v2 Value) Value) funcInfo {
	return funcInfo{
		def: syntax.BuiltInFunctions[fname],
		call: func(args []Value) Value {
			return impl(args[0], args[1])
		},
	}
}

// file contains implementation setups for function calls.
func (interp *Interpreter) initFuncs() {
	interp.fn = map[string]funcInfo{}

	interp.fn["ADD"] = binaryImpl("ADD", Value.Add)
	interp.fn["SUB"] = binaryImpl("SUB", Value.Subtract)
	interp.fn["MULT"] = binaryImpl("MULT", Value.Multiply)
	interp.fn["DIV"] = binaryImpl("DIV", Value.Divide)
	interp.fn["NEG"] = unaryImpl("NEG", Value.Negate)
	interp.fn["OR"] = binaryImpl("OR", Value.Or)
	interp.fn["AND"] = binaryImpl("AND", Value.And)
	interp.fn["NOT"] = unaryImpl("NOT", Value.Not)
	interp.fn["FLAG_ENABLED"] = unaryImpl("FLAG_ENABLED", Value.CastToBool)
	interp.fn["FLAG_DISABLED"] = unaryImpl("FLAG_DISABLED", func(v Value) Value {
		return v.CastToBool().Negate()
	})
	interp.fn["FLAG_IS"] = binaryImpl("FLAG_IS", Value.EqualTo)
	interp.fn["FLAG_LESS_THAN"] = binaryImpl("FLAG_LESS_THAN", Value.LessThan)
	interp.fn["FLAG_GREATER_THAN"] = binaryImpl("FLAG_GREATER_THAN", Value.GreaterThan)
	interp.fn["ENABLE"] = unaryImpl("ENABLE", interp.enable)
	interp.fn["DISABLE"] = unaryImpl("DISABLE", interp.disable)
	interp.fn["TOGGLE"] = unaryImpl("TOGGLE", interp.toggle)
	interp.fn["IN_INVEN"] = unaryImpl("IN_INVEN", interp.inInven)
	interp.fn["SET"] = binaryImpl("SET", interp.set)
	interp.fn["MOVE"] = binaryImpl("MOVE", interp.move)
	interp.fn["OUTPUT"] = unaryImpl("OUTPUT", interp.output)

	// and the two variable-arity functions
	interp.fn["INC"] = funcInfo{
		def: syntax.BuiltInFunctions["INC"],
		call: func(args []Value) Value {
			if len(args) > 1 {
				return interp.inc(args[0], args[1])
			}
			return interp.inc(args[0], syntax.ValueOf(1))
		},
	}
	interp.fn["DEC"] = funcInfo{
		def: syntax.BuiltInFunctions["DEC"],
		call: func(args []Value) Value {
			if len(args) > 1 {
				return interp.dec(args[0], args[1])
			}
			return interp.dec(args[0], syntax.ValueOf(1))
		},
	}
}

func (interp *Interpreter) inInven(v Value) Value {
	itemLabelName := strings.ToUpper(v.String())

	return syntax.ValueOf(interp.Target.InInventory(itemLabelName))
}

func (interp *Interpreter) move(target, dest Value) Value {
	targetStr := strings.ToUpper(target.String())
	destStr := strings.ToUpper(dest.String())

	return syntax.ValueOf(interp.Target.Move(targetStr, destStr))
}

func (interp *Interpreter) output(msg Value) Value {
	return syntax.ValueOf(interp.Target.Output(msg.String()))
}

func (interp *Interpreter) set(name Value, value Value) Value {
	flagName := strings.ToUpper(name.String())

	interp.flags[flagName] = value
	return value
}

func (interp *Interpreter) enable(v Value) Value {
	flagName := strings.ToUpper(v.String())
	newVal := syntax.ValueOf(true)

	interp.flags[flagName] = newVal
	return newVal
}

func (interp *Interpreter) disable(v Value) Value {
	flagName := strings.ToUpper(v.String())
	newVal := syntax.ValueOf(false)

	interp.flags[flagName] = newVal
	return newVal
}

func (interp *Interpreter) toggle(v Value) Value {
	flagName := strings.ToUpper(v.String())
	curVal := interp.flags[flagName]
	newVal := curVal.CastToBool().Not()

	interp.flags[flagName] = newVal
	return newVal
}

func (interp *Interpreter) inc(name Value, amount Value) Value {
	flagName := strings.ToUpper(name.String())

	curVal := interp.flags[flagName]
	newVal := curVal.Add(amount)

	interp.flags[flagName] = newVal
	return newVal
}

func (interp *Interpreter) dec(name Value, amount Value) Value {
	flagName := strings.ToUpper(name.String())

	curVal := interp.flags[flagName]
	newVal := curVal.Subtract(amount)

	interp.flags[flagName] = newVal
	return newVal
}
