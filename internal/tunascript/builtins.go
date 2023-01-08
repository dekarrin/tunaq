package tunascript

import (
	"math"
	"strings"
)

func builtIn_Add(args []Value) Value {
	x := args[0]
	y := args[1]

	if x.Type() == Bool {
		x = NewNum(x.Num())
	}

	var ret Value

	if x.Type() == Num {
		xNum := x.Num()
		yNum := y.Num()

		ret = NewNum(xNum + yNum)
	} else {
		xStr := x.Str()
		yStr := y.Str()

		ret = NewStr(xStr + yStr)
	}

	return ret
}

func builtIn_Sub(args []Value) Value {
	x := args[0]
	y := args[1]

	return NewNum(x.Num() - y.Num())
}

func builtIn_Mult(args []Value) Value {
	x := args[0]
	y := args[1]

	if x.Type() == Bool {
		x = NewStr(x.Str())
	}

	var ret Value

	if x.Type() == Str {
		builtStr := ""

		num := y.Num()
		for i := 0; i < num; i++ {
			builtStr += y.Str()
		}

		ret = NewStr(builtStr)
	} else {
		ret = NewNum(x.Num() * y.Num())
	}

	return ret
}

func builtIn_Div(args []Value) Value {
	x := args[0]
	y := args[1]

	div := float64(x.Num()) / float64(y.Num())
	retVal := int(math.Round(div))

	return NewNum(retVal)
}

func builtIn_Or(args []Value) Value {
	x := args[0]
	y := args[1]

	return NewBool(x.Bool() || y.Bool())
}

func builtIn_And(args []Value) Value {
	x := args[0]
	y := args[1]

	return NewBool(x.Bool() && y.Bool())
}

func builtIn_Not(args []Value) Value {
	x := args[0]

	return NewBool(!x.Bool())
}

func (inter Interpreter) builtIn_FlagEnabled(args []Value) Value {
	flagName := args[0].Str()

	if flag, ok := inter.flags[flagName]; !ok {
		return NewBool(false)
	} else {
		return NewBool(flag.Value.Bool())
	}
}

func (inter Interpreter) builtIn_FlagDisabled(args []Value) Value {
	flagName := args[0].Str()

	if flag, ok := inter.flags[flagName]; !ok {
		return NewBool(true)
	} else {
		return NewBool(!flag.Value.Bool())
	}
}

func (inter Interpreter) builtIn_FlagIs(args []Value) Value {
	flagName := args[0].Str()
	checkVal := args[1]

	var ret Value
	switch checkVal.Type() {
	case Num:
		if flag, ok := inter.flags[flagName]; !ok {
			ret = NewBool(checkVal.Num() == 0)
		} else {
			ret = NewBool(flag.Value.Num() == checkVal.Num())
		}
	case Bool:
		if flag, ok := inter.flags[flagName]; !ok {
			ret = NewBool(!checkVal.Bool())
		} else {
			ret = NewBool(flag.Value.Bool() == checkVal.Bool())
		}
	case Str:
		if flag, ok := inter.flags[flagName]; !ok {
			ret = NewBool(checkVal.Str() == "")
		} else {
			ret = NewBool(flag.Value.Str() == checkVal.Str())
		}
	}

	return ret
}

func (inter Interpreter) builtIn_FlagLessThan(args []Value) Value {
	flagName := args[0].Str()
	checkVal := args[1]

	if flag, ok := inter.flags[flagName]; !ok {
		return NewBool(checkVal.Num() > 0)
	} else {
		return NewBool(flag.Value.Num() < checkVal.Num())
	}
}

func (inter Interpreter) builtIn_FlagGreaterThan(args []Value) Value {
	flagName := args[0].Str()
	checkVal := args[1]

	if flag, ok := inter.flags[flagName]; !ok {
		return NewBool(checkVal.Num() < 0)
	} else {
		return NewBool(flag.Value.Num() > checkVal.Num())
	}
}

func (inter Interpreter) builtIn_InInven(args []Value) Value {
	itemLabelName := args[0].Str()

	return NewBool(inter.world.InInventory(itemLabelName))
}

func (inter Interpreter) builtIn_Enable(args []Value) Value {
	flagName := args[0].Str()

	flag := inter.flags[flagName]
	if flag == nil {
		flag = &Flag{
			Name: strings.ToUpper(flagName),
		}
		inter.flags[flagName] = flag
	}

	flag.Value = NewBool(true)
	return flag.Value
}

func (inter Interpreter) builtIn_Disable(args []Value) Value {
	flagName := args[0].Str()

	flag := inter.flags[flagName]
	if flag == nil {
		flag = &Flag{
			Name: strings.ToUpper(flagName),
		}
		inter.flags[flagName] = flag
	}

	flag.Value = NewBool(false)
	return flag.Value
}

func (inter Interpreter) builtIn_Toggle(args []Value) Value {
	flagName := args[0].Str()

	flag := inter.flags[flagName]
	if flag == nil {
		flag = &Flag{
			Name: strings.ToUpper(flagName),
		}
		inter.flags[flagName] = flag
	}

	flag.Value = NewBool(!flag.Value.Bool())
	return flag.Value
}

func (inter Interpreter) builtIn_Inc(args []Value) Value {
	flagName := args[0].Str()
	amount := 1
	if len(args) > 1 {
		amount = args[1].Num()
	}

	flag := inter.flags[flagName]
	if flag == nil {
		flag = &Flag{
			Name: strings.ToUpper(flagName),
		}
		inter.flags[flagName] = flag
	}

	flag.Value = NewNum(flag.Value.Num() + amount)
	return flag.Value
}

func (inter Interpreter) builtIn_Dec(args []Value) Value {
	flagName := args[0].Str()
	amount := 1
	if len(args) > 1 {
		amount = args[1].Num()
	}

	flag := inter.flags[flagName]
	if flag == nil {
		flag = &Flag{
			Name: strings.ToUpper(flagName),
		}
		inter.flags[flagName] = flag
	}

	flag.Value = NewNum(flag.Value.Num() - amount)
	return flag.Value
}

func (inter Interpreter) builtIn_Set(args []Value) Value {
	flagName := args[0].Str()
	setVal := args[1]

	flag := inter.flags[flagName]
	if flag == nil {
		flag = &Flag{
			Name: strings.ToUpper(flagName),
		}
		inter.flags[flagName] = flag
	}

	if setVal.Type() == Num {
		flag.Value = NewNum(setVal.Num())
	} else {
		flag.Value = NewStr(setVal.Str())
	}

	return flag.Value
}

func (inter Interpreter) builtIn_Move(args []Value) Value {
	target := args[0].Str()
	dest := args[1].Str()

	return NewBool(inter.world.Move(target, dest))
}

func (inter Interpreter) builtIn_Output(args []Value) Value {
	msg := args[0].Str()

	return NewBool(inter.world.Output(msg))
}
