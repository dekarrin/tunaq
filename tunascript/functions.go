package tunascript

import (
	"strings"

	"github.com/dekarrin/tunaq/tunascript/syntax"
)

type funcImpl func(args []syntax.Value) syntax.Value

type funcInfo struct {
	def  syntax.Function
	call funcImpl
}

func unaryImpl(fname string, impl func(v syntax.Value) syntax.Value) funcInfo {
	return funcInfo{
		def: syntax.BuiltInFunctions[fname],
		call: func(args []syntax.Value) syntax.Value {
			return impl(args[0])
		},
	}
}

func binaryImpl(fname string, impl func(v syntax.Value, v2 syntax.Value) syntax.Value) funcInfo {
	return funcInfo{
		def: syntax.BuiltInFunctions[fname],
		call: func(args []syntax.Value) syntax.Value {
			return impl(args[0], args[1])
		},
	}
}

// file contains implementation setups for function calls.
func (interp *Interpreter) initFuncs() {
	interp.fn = map[string]funcInfo{}

	interp.fn["ADD"] = binaryImpl("ADD", syntax.Value.Add)
	interp.fn["SUB"] = binaryImpl("SUB", syntax.Value.Subtract)
	interp.fn["MULT"] = binaryImpl("MULT", syntax.Value.Multiply)
	interp.fn["DIV"] = binaryImpl("DIV", syntax.Value.Divide)
	interp.fn["NEG"] = unaryImpl("NEG", syntax.Value.Negate)
	interp.fn["OR"] = binaryImpl("OR", syntax.Value.Or)
	interp.fn["AND"] = binaryImpl("AND", syntax.Value.And)
	interp.fn["NOT"] = unaryImpl("NOT", syntax.Value.Not)
	interp.fn["FLAG_ENABLED"] = unaryImpl("FLAG_ENABLED", syntax.Value.CastToBool)
	interp.fn["FLAG_DISABLED"] = unaryImpl("FLAG_DISABLED", func(v syntax.Value) syntax.Value {
		return v.CastToBool().Negate()
	})
	interp.fn["FLAG_IS"] = binaryImpl("FLAG_IS", syntax.Value.EqualTo)
	interp.fn["FLAG_LESS_THAN"] = binaryImpl("FLAG_LESS_THAN", syntax.Value.LessThan)
	interp.fn["FLAG_GREATER_THAN"] = binaryImpl("FLAG_GREATER_THAN", syntax.Value.GreaterThan)
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
		call: func(args []syntax.Value) syntax.Value {
			if len(args) > 1 {
				return interp.inc(args[0], args[1])
			}
			return interp.inc(args[0], syntax.ValueOf(1))
		},
	}
	interp.fn["DEC"] = funcInfo{
		def: syntax.BuiltInFunctions["DEC"],
		call: func(args []syntax.Value) syntax.Value {
			if len(args) > 1 {
				return interp.dec(args[0], args[1])
			}
			return interp.dec(args[0], syntax.ValueOf(1))
		},
	}
}

func (interp *Interpreter) inInven(v syntax.Value) syntax.Value {
	itemLabelName := strings.ToUpper(v.String())

	return syntax.ValueOf(interp.Target.InInventory(itemLabelName))
}

func (interp *Interpreter) move(target, dest syntax.Value) syntax.Value {
	targetStr := strings.ToUpper(target.String())
	destStr := strings.ToUpper(dest.String())

	return syntax.ValueOf(interp.Target.Move(targetStr, destStr))
}

func (interp *Interpreter) output(msg syntax.Value) syntax.Value {
	return syntax.ValueOf(interp.Target.Output(msg.String()))
}

func (interp *Interpreter) set(name syntax.Value, value syntax.Value) syntax.Value {
	flagName := strings.ToUpper(name.String())

	interp.Flags[flagName] = value
	return value
}

func (interp *Interpreter) enable(v syntax.Value) syntax.Value {
	flagName := strings.ToUpper(v.String())
	newVal := syntax.ValueOf(true)

	interp.Flags[flagName] = newVal
	return newVal
}

func (interp *Interpreter) disable(v syntax.Value) syntax.Value {
	flagName := strings.ToUpper(v.String())
	newVal := syntax.ValueOf(false)

	interp.Flags[flagName] = newVal
	return newVal
}

func (interp *Interpreter) toggle(v syntax.Value) syntax.Value {
	flagName := strings.ToUpper(v.String())
	curVal := interp.Flags[flagName]
	newVal := curVal.CastToBool().Not()

	interp.Flags[flagName] = newVal
	return newVal
}

func (interp *Interpreter) inc(name syntax.Value, amount syntax.Value) syntax.Value {
	flagName := strings.ToUpper(name.String())

	curVal := interp.Flags[flagName]
	newVal := curVal.Add(amount)

	interp.Flags[flagName] = newVal
	return newVal
}

func (interp *Interpreter) dec(name syntax.Value, amount syntax.Value) syntax.Value {
	flagName := strings.ToUpper(name.String())

	curVal := interp.Flags[flagName]
	newVal := curVal.Subtract(amount)

	interp.Flags[flagName] = newVal
	return newVal
}
