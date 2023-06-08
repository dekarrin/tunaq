package syntax

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/dekarrin/ictiobus/trans"
)

var (
	HooksTable = trans.HookMap{
		"ast":           hookAST,
		"bin_or":        makeHookBinaryOp(OpBinaryLogicalOr),
		"bin_and":       makeHookBinaryOp(OpBinaryLogicalAnd),
		"bin_eq":        makeHookBinaryOp(OpBinaryEqual),
		"bin_ne":        makeHookBinaryOp(OpBinaryNotEqual),
		"bin_lt":        makeHookBinaryOp(OpBinaryLessThan),
		"bin_le":        makeHookBinaryOp(OpBinaryLessThanEqual),
		"bin_gt":        makeHookBinaryOp(OpBinaryGreaterThan),
		"bin_ge":        makeHookBinaryOp(OpBinaryGreaterThanEqual),
		"bin_sub":       makeHookBinaryOp(OpBinarySubtract),
		"bin_add":       makeHookBinaryOp(OpBinaryAdd),
		"bin_mult":      makeHookBinaryOp(OpBinaryMultiply),
		"bin_div":       makeHookBinaryOp(OpBinaryDivide),
		"unary_not":     makeHookUnaryOp(OpUnaryLogicalNot),
		"unary_neg":     makeHookUnaryOp(OpUnaryNegate),
		"group":         hookGroup,
		"func":          hookFunc,
		"args_list":     hookArgsList,
		"assign_set":    makeHookAssignBinary(OpAssignSet),
		"assign_incset": makeHookAssignBinary(OpAssignIncrementBy),
		"assign_decset": makeHookAssignBinary(OpAssignDecrementBy),
		"assign_dec":    makeHookAssignUnary(OpAssignDecrement),
		"assign_inc":    makeHookAssignUnary(OpAssignIncrement),
		"flag":          hookFlag,
		"lit_binary":    hookLitBinary,
		"lit_text":      hookLitText,
		"lit_num":       hookLitNum,
		"identity":      func(info trans.SetterInfo, args []interface{}) (interface{}, error) { return args[0], nil },
	}
)

func makeHookBinaryOp(op BinaryOperation) trans.Hook {
	return func(info trans.SetterInfo, args []interface{}) (interface{}, error) {
		left := args[0].(ASTNode)
		right := args[1].(ASTNode)

		node := BinaryOpNode{
			Left:  left,
			Right: right,
			Op:    op,
			src:   info.FirstToken,
		}

		return node, nil
	}
}

func makeHookAssignBinary(op AssignmentOperation) trans.Hook {
	return func(info trans.SetterInfo, args []interface{}) (interface{}, error) {
		lexedIdent := args[0].(string)
		value := args[1].(ASTNode)

		node := AssignmentNode{
			Flag:  lexedIdent,
			Value: value,
			Op:    op,
			src:   info.FirstToken,
		}

		return node, nil
	}
}

func makeHookAssignUnary(op AssignmentOperation) trans.Hook {
	return func(info trans.SetterInfo, args []interface{}) (interface{}, error) {
		lexedIdent := args[0].(string)

		node := AssignmentNode{
			Flag: lexedIdent,
			Op:   op,
			src:  info.FirstToken,
		}

		return node, nil
	}
}

func makeHookUnaryOp(op UnaryOperation) trans.Hook {
	return func(info trans.SetterInfo, args []interface{}) (interface{}, error) {
		operand := args[0].(ASTNode)

		node := UnaryOpNode{
			Operand: operand,
			Op:      op,
			src:     info.FirstToken,
		}

		return node, nil
	}
}

func hookAST(info trans.SetterInfo, args []interface{}) (interface{}, error) {
	stmt := args[0].(ASTNode)

	ast := AST{
		Nodes: []ASTNode{stmt},
	}

	return ast, nil
}

func hookGroup(info trans.SetterInfo, args []interface{}) (interface{}, error) {
	expr := args[0].(ASTNode)

	node := GroupNode{
		Expr: expr,
		src:  info.FirstToken,
	}

	return node, nil
}

func hookFunc(info trans.SetterInfo, args []interface{}) (interface{}, error) {
	lexedName := args[0].(string)
	fargs := args[1].([]ASTNode)

	fname := strings.ToUpper(lexedName)

	// check that the function is defined and check its arity
	def, ok := BuiltInFunctions[fname]
	if !ok {
		return nil, fmt.Errorf("$%s() is not a function that exists in TunaScript", lexedName)
	}
	min := def.RequiredArgs
	max := def.RequiredArgs + def.OptionalArgs

	if len(fargs) < min || len(fargs) > max {
		if def.OptionalArgs == 0 {
			var argPlural string
			if def.RequiredArgs != 1 {
				argPlural = "s"
			}
			return nil, fmt.Errorf("$%s() requires exactly %d argument%s, but was given %d", lexedName, def.RequiredArgs, argPlural, len(fargs))
		} else {
			var maxPlural string
			if max != 1 {
				maxPlural = "s"
			}
			return nil, fmt.Errorf("$%s() requires between %d and %d argument%s, but was given %d", lexedName, min, max, maxPlural, len(fargs))
		}
	}

	node := FuncNode{
		Func: fname,
		Args: fargs,
		src:  info.FirstToken,
	}

	return node, nil
}

func hookArgsList(info trans.SetterInfo, args []interface{}) (interface{}, error) {
	var list []ASTNode

	var appendNode ASTNode

	if len(args) >= 1 {
		// add item to list
		appendNode = args[0].(ASTNode)
		if len(args) >= 2 {
			list = args[1].([]ASTNode)
		}
	}

	if appendNode != nil {
		list = append(list, appendNode)
	}

	return list, nil
}

func hookFlag(info trans.SetterInfo, args []interface{}) (interface{}, error) {
	lexedIdent := args[0].(string)

	fname := strings.ToUpper(lexedIdent)

	node := FlagNode{
		Flag: fname,
		src:  info.FirstToken,
	}

	return node, nil
}

func hookLitBinary(info trans.SetterInfo, args []interface{}) (interface{}, error) {
	lexedText := args[0].(string)
	node := LiteralNode{src: info.FirstToken}
	boolText := strings.ToUpper(lexedText)

	if boolText == "ON" || boolText == "YES" || boolText == "TRUE" {
		node.Value = ValueOf(true)
		return node, nil
	} else if boolText == "OFF" || boolText == "NO" || boolText == "FALSE" {
		node.Value = ValueOf(false)
		return node, nil
	}
	return nil, fmt.Errorf("not a valid binary value: %v", lexedText)
}

func hookLitText(info trans.SetterInfo, args []interface{}) (interface{}, error) {
	lexedText := args[0].(string)
	node := LiteralNode{src: info.FirstToken}
	str := lexedText

	// is this a quoted string?
	if str[0] == '@' && str[len(str)-1] == '@' {
		// remove quotes
		str = str[1 : len(str)-1]
		node.Quoted = true
	}

	// unescape things
	str = InterpretEscapes(str)

	node.Value = ValueOf(str)
	return node, nil
}

func hookLitNum(info trans.SetterInfo, args []interface{}) (interface{}, error) {
	lexedText := args[0].(string)
	node := LiteralNode{src: info.FirstToken}

	// if it has a period, it is a float
	if strings.Contains(lexedText, ".") {
		fVal, err := strconv.ParseFloat(lexedText, 64)
		if err != nil {
			return nil, fmt.Errorf("not a valid number: %v", lexedText)
		}
		node.Value = ValueOf(fVal)
	} else {
		// it's an int, make sure we chop off any exponent part.
		sVal := strings.ToLower(lexedText)
		expSplit := strings.SplitN(sVal, "e", 2)
		exponent := 0
		if len(expSplit) > 1 {
			var err error
			exponent, err = strconv.Atoi(expSplit[1])
			if err != nil {
				return nil, fmt.Errorf("not a valid exponent value: %v", lexedText)
			}
		}

		iVal, err := strconv.Atoi(expSplit[0])
		if err != nil {
			return nil, fmt.Errorf("not a valid number: %v", lexedText)
		}

		if exponent != 0 {
			absExp := int(math.Abs(float64(exponent)))
			factor := 10
			for i := 0; i < absExp; i++ {
				factor *= 10
			}

			if exponent < 0 {
				factor *= -1
			}

			iVal *= factor
		}

		node.Value = ValueOf(iVal)
	}
	return node, nil
}

func InterpretEscapes(s string) string {
	var sb strings.Builder

	var inEscape bool
	for _, ch := range s {
		if inEscape {
			inEscape = false
			sb.WriteRune(ch)
		} else {
			if ch == '\\' {
				inEscape = true
			} else {
				sb.WriteRune(ch)
			}
		}
	}

	return sb.String()
}
