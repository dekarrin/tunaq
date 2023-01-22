package tunascript

import (
	"fmt"
	"strconv"
	"strings"
)

// File operators.go contains transpilation functions for turning operator-based
// TunaScript expressions into function-based ones that can be parsed by the
// rest of the system.

const maxTokenBindingPower = 200

// TODO: Dude, Go has the template package! Why aren't we using it????????
var binaryOpFuncTranslations = map[string]string{
	literalStrOpPlus:          "%[1]sADD%[2]s%[4]s%[6]s %[5]s%[3]s",
	literalStrOpMinus:         "%[1]sSUB%[2]s%[4]s%[6]s %[5]s%[3]s",
	literalStrOpDivide:        "%[1]sDIV%[2]s%[4]s%[6]s %[5]s%[3]s",
	literalStrOpMultiply:      "%[1]sMULT%[2]s%[4]s%[6]s %[5]s%[3]s",
	literalStrOpAnd:           "%[1]sAND%[2]s%[4]s%[6]s %[5]s%[3]s",
	literalStrOpOr:            "%[1]sOR%[2]s%[4]s%[6]s %[5]s%[3]s",
	literalStrOpIncset:        "%[1]sINC%[2]s%[4]s%[6]s %[5]s%[3]s",
	literalStrOpDecset:        "%[1]sDEC%[2]s%[4]s%[6]s %[5]s%[3]s",
	literalStrOpIsNot:         "%[1]sNOT%[2]s%[1]sFLAG_IS%[2]s%[4]s%[6]s %[5]s%[3]s%[3]s",
	literalStrOpIs:            "%[1]sFLAG_IS%[2]s%[4]s%[6]s %[5]s%[3]s",
	literalStrOpLessThan:      "%[1]sFLAG_LESS_THAN%[2]s%[4]s%[6]s %[5]s%[3]s",
	literalStrOpGreaterThan:   "%[1]sFLAG_GREATER_THAN%[2]s%[4]s%[6]s %[5]s%[3]s",
	literalStrOpGreaterThanIs: "%[1]sOR%[2]s%[1]sFLAG_GREATER_THAN%[2]s%[4]s%[6]s %[5]s%[3]s%[6]s %[1]sFLAG_IS%[2]s%[4]s%[6]s %[5]s%[3]s%[3]s",
	literalStrOpLessThanIs:    "%[1]sOR%[2]s%[1]sFLAG_LESS_THAN%[2]s%[4]s%[6]s %[5]s%[3]s%[6]s %[1]sFLAG_IS%[2]s%[4]s%[6]s %[5]s%[3]s%[3]s",
	literalStrOpSet:           "%[1]sSET%[2]s%[4]s%[6]s %[5]s%[3]s",
}

var unaryOpFuncTranslations = map[string]string{
	literalStrOpNot:   "%[1]sNOT%[2]s%[4]s%[3]s",
	literalStrOpInc:   "%[1]sINC%[2]s%[4]s%[3]s",
	literalStrOpDec:   "%[1]sDEC%[2]s%[4]s%[3]s",
	literalStrOpMinus: "%[1]sNEG%[2]s%[4]s%[3]s",
}

// null denotation values for pratt parsing
//
// return nil for "this token cannot appear at start of lang construct", or
// "represents end of text."
func (lex token) nud(tokens *tokenStream) (*astNode, error) {
	switch lex.class {
	case tsUnquotedString:
		return &astNode{
			value: &valueNode{
				unquotedStringVal: &lex.lexeme,
			},
			source: lex,
		}, nil
	case tsQuotedString:
		return &astNode{
			value: &valueNode{
				quotedStringVal: &lex.lexeme,
			},
			source: lex,
		}, nil
	case tsOpMinus:
		negatedVal, err := parseOpExpression(tokens, maxTokenBindingPower)
		if err != nil {
			return nil, err
		}

		return &astNode{
			opGroup: &operatorGroupNode{
				unaryOp: &unaryOperatorGroupNode{
					op:      literalStrOpMinus,
					operand: negatedVal,
				},
			},
			source: lex,
		}, nil
	case tsNumber:
		num, err := strconv.Atoi(lex.lexeme)
		if err != nil {
			panic(fmt.Sprintf("got non-integer value %q in %s token, should never happen", lex.lexeme, lex.class.id))
		}
		return &astNode{
			value: &valueNode{
				numVal: &num,
			},
			source: lex,
		}, nil
	case tsBool:
		vUp := strings.ToUpper(lex.lexeme)

		var boolVal bool

		if vUp == "TRUE" || vUp == "ON" || vUp == "YES" {
			boolVal = true
		} else if vUp == "FALSE" || vUp == "OFF" || vUp == "NO" {
			boolVal = false
		} else {
			panic(fmt.Sprintf("got non-bool value %q in %s token, should never happen", lex.lexeme, lex.class.id))
		}

		return &astNode{
			value: &valueNode{
				boolVal: &boolVal,
			},
			source: lex,
		}, nil
	case tsIdentifier:
		flagName := strings.ToUpper(lex.lexeme)
		return &astNode{
			flag: &flagNode{
				name: flagName,
			},
			source: lex,
		}, nil
	case tsGroupOpen:
		expr, err := parseOpExpression(tokens, 0)
		if err != nil {
			return nil, err
		}
		next := tokens.Next()
		if next.class != tsGroupClose {
			return nil, syntaxErrorFromLexeme("unmatched '"+literalStrGroupOpen+"'; expected a '"+literalStrGroupClose+"' here", next)
		}

		return &astNode{
			group: &groupNode{
				expr: expr,
			},
			source: lex,
		}, nil
	default:
		return nil, nil
	}
}

// left denotation values for pratt parsing
//
// return nil for "this token MUST appear at start of language construct", or
// "represents end of text."
//
// err is only non-nil on failure to parse
func (lex token) led(left *astNode, tokens *tokenStream) (*astNode, error) {
	switch lex.class {
	case tsOpLessThanIs:
		right, err := parseOpExpression(tokens, lex.class.lbp)
		if err != nil {
			return nil, err
		}

		return &astNode{
			opGroup: &operatorGroupNode{
				infixOp: &binaryOperatorGroupNode{
					op:    literalStrOpLessThanIs,
					left:  left,
					right: right,
				},
			},
			source: lex,
		}, nil
	case tsOpGreaterThanIs:
		right, err := parseOpExpression(tokens, lex.class.lbp)
		if err != nil {
			return nil, err
		}

		return &astNode{
			opGroup: &operatorGroupNode{
				infixOp: &binaryOperatorGroupNode{
					op:    literalStrOpGreaterThanIs,
					left:  left,
					right: right,
				},
			},
			source: lex,
		}, nil
	case tsOpLessThan:
		right, err := parseOpExpression(tokens, lex.class.lbp)
		if err != nil {
			return nil, err
		}

		return &astNode{
			opGroup: &operatorGroupNode{
				infixOp: &binaryOperatorGroupNode{
					op:    literalStrOpLessThan,
					left:  left,
					right: right,
				},
			},
			source: lex,
		}, nil
	case tsOpGreaterThan:
		right, err := parseOpExpression(tokens, lex.class.lbp)
		if err != nil {
			return nil, err
		}

		return &astNode{
			opGroup: &operatorGroupNode{
				infixOp: &binaryOperatorGroupNode{
					op:    literalStrOpGreaterThan,
					left:  left,
					right: right,
				},
			},
			source: lex,
		}, nil
	case tsOpIsNot:
		right, err := parseOpExpression(tokens, lex.class.lbp)
		if err != nil {
			return nil, err
		}

		return &astNode{
			opGroup: &operatorGroupNode{
				infixOp: &binaryOperatorGroupNode{
					op:    literalStrOpIsNot,
					left:  left,
					right: right,
				},
			},
			source: lex,
		}, nil
	case tsOpIs:
		right, err := parseOpExpression(tokens, lex.class.lbp)
		if err != nil {
			return nil, err
		}

		return &astNode{
			opGroup: &operatorGroupNode{
				infixOp: &binaryOperatorGroupNode{
					op:    literalStrOpIs,
					left:  left,
					right: right,
				},
			},
			source: lex,
		}, nil
	case tsOpSet:
		right, err := parseOpExpression(tokens, lex.class.lbp)
		if err != nil {
			return nil, err
		}

		return &astNode{
			opGroup: &operatorGroupNode{
				infixOp: &binaryOperatorGroupNode{
					op:    literalStrOpSet,
					left:  left,
					right: right,
				},
			},
			source: lex,
		}, nil
	case tsOpIncset:
		right, err := parseOpExpression(tokens, lex.class.lbp)
		if err != nil {
			return nil, err
		}

		return &astNode{
			opGroup: &operatorGroupNode{
				infixOp: &binaryOperatorGroupNode{
					op:    literalStrOpIncset,
					left:  left,
					right: right,
				},
			},
			source: lex,
		}, nil
	case tsOpDecset:
		right, err := parseOpExpression(tokens, lex.class.lbp)
		if err != nil {
			return nil, err
		}

		return &astNode{
			opGroup: &operatorGroupNode{
				infixOp: &binaryOperatorGroupNode{
					op:    literalStrOpDecset,
					left:  left,
					right: right,
				},
			},
			source: lex,
		}, nil
	case tsOpNot:
		return &astNode{
			opGroup: &operatorGroupNode{
				unaryOp: &unaryOperatorGroupNode{
					op:      literalStrOpNot,
					operand: left,
				},
			},
			source: lex,
		}, nil
	case tsOpInc:
		return &astNode{
			opGroup: &operatorGroupNode{
				unaryOp: &unaryOperatorGroupNode{
					op:      literalStrOpInc,
					operand: left,
				},
			},
			source: lex,
		}, nil
	case tsOpDec:
		return &astNode{
			opGroup: &operatorGroupNode{
				unaryOp: &unaryOperatorGroupNode{
					op:      literalStrOpDec,
					operand: left,
				},
			},
			source: lex,
		}, nil
	case tsOpPlus:
		right, err := parseOpExpression(tokens, lex.class.lbp)
		if err != nil {
			return nil, err
		}
		return &astNode{
			opGroup: &operatorGroupNode{
				infixOp: &binaryOperatorGroupNode{
					op:    literalStrOpPlus,
					left:  left,
					right: right,
				},
			},
			source: lex,
		}, nil
	case tsOpMinus:
		right, err := parseOpExpression(tokens, lex.class.lbp)
		if err != nil {
			return nil, err
		}
		return &astNode{
			opGroup: &operatorGroupNode{
				infixOp: &binaryOperatorGroupNode{
					op:    literalStrOpMinus,
					left:  left,
					right: right,
				},
			},
			source: lex,
		}, nil
	case tsOpMultiply:
		right, err := parseOpExpression(tokens, lex.class.lbp)
		if err != nil {
			return nil, err
		}
		return &astNode{
			opGroup: &operatorGroupNode{
				infixOp: &binaryOperatorGroupNode{
					op:    literalStrOpMultiply,
					left:  left,
					right: right,
				},
			},
			source: lex,
		}, nil
	case tsOpDivide:
		right, err := parseOpExpression(tokens, lex.class.lbp)
		if err != nil {
			return nil, err
		}
		return &astNode{
			opGroup: &operatorGroupNode{
				infixOp: &binaryOperatorGroupNode{
					op:    literalStrOpDivide,
					left:  left,
					right: right,
				},
			},
			source: lex,
		}, nil
	case tsGroupOpen:
		// binary op '(' binds to expr
		callArgs := []*astNode{}

		if left.flag == nil {
			return nil, syntaxErrorFromLexeme(fmt.Sprintf("unexpected %s\n(expected it to be after a function name or to group expressions)", lex.class.human), lex)
		}

		if tokens.Peek().class != tsGroupClose {
			for {
				arg, err := parseOpExpression(tokens, 0)
				if err != nil {
					return nil, err
				}
				callArgs = append(callArgs, arg)

				if tokens.Peek().class == tsSeparator {
					tokens.Next() // toss off the separator and prep to parse the next one
				} else {
					break
				}
			}
		}

		nextTok := tokens.Next()
		if nextTok.class != tsGroupClose {
			return nil, syntaxErrorFromLexeme(fmt.Sprintf("unexpected %s\n(expected a '"+literalStrGroupClose+"' to close the previous '"+literalStrGroupOpen+"')", nextTok.class.human), lex)
		}

		return &astNode{
			fn: &fnNode{
				name: left.flag.name,
				args: callArgs,
			},
			source: lex,
		}, nil
	default:
		return nil, nil
	}
}
