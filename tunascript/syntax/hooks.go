package syntax

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/dekarrin/ictiobus/trans"
)

var (
	escapeSequenceRegex = regexp.MustCompile(`\\(.)`)
)
var (
	HooksTable = trans.HookMap{
		"test_const": func(info trans.SetterInfo, args []interface{}) (interface{}, error) {
			return 1, nil
		},
		"args_list":  hookArgsList,
		"assign_dec": makeHookAssignUnary(OpAssignDecrement),
		"assign_inc": makeHookAssignUnary(OpAssignIncrement),
		"flag":       hookFlag,
		"lit_binary": hookLitBinary,
		"lit_text":   hookLitText,
		"lit_num":    hookLitNum,
		"identity":   func(info trans.SetterInfo, args []interface{}) (interface{}, error) { return args[0], nil },
	}
)

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

	node := FlagNode{
		Flag: lexedIdent,
		src:  info.FirstToken,
	}

	return node, nil
}

func hookLitBinary(info trans.SetterInfo, args []interface{}) (interface{}, error) {
	lexedText := args[0].(string)
	node := LiteralNode{src: info.FirstToken}
	boolText := strings.ToUpper(lexedText)

	if boolText == "ON" || boolText == "YES" || boolText == "TRUE" {
		node.Value = TSValueOf(true)
		return node, nil
	} else if boolText == "OFF" || boolText == "NO" || boolText == "FALSE" {
		node.Value = TSValueOf(false)
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

	node.Value = TSValueOf(str)
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
		node.Value = TSValueOf(fVal)
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

		iVal, err := strconv.Atoi(expSplit[1])
		if err != nil {
			return fmt.Errorf("not a valid number: %v", lexedText)
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

		node.Value = TSValueOf(iVal)
	}
}
