package tunascript

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// File operators.go contains transpilation functions for turning operator-based
// TunaScript expressions into function-based ones that can be parsed by the
// rest of the system.

var (
	patBool = regexp.MustCompile(`^(?:[Tt][Rr][Uu][Ee])|(?:[Ff][Aa][Ll][Ss][Ee])|(?:[Oo][Nn])|(?:[Oo][Ff][Ff])|(?:[Yy][Ee][Ss])|(?:[Nn][Oo])$`)
	patNum  = regexp.MustCompile(`^-?[0-9]+$`)
)

const maxTokenBindingPower = 200

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
					op:      "-",
					operand: negatedVal,
				},
			},
			source: lex,
		}, nil
	case tsNumber:
		num, err := strconv.Atoi(lex.lexeme)
		if err != nil {
			panic(fmt.Sprintf("got non-integer value %q in LEX_NUMBER token, should never happen", lex.lexeme))
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
			panic(fmt.Sprintf("got non-bool value %q in LEX_BOOL token, should never happen", lex.lexeme))
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
			return nil, syntaxErrorFromLexeme("unmatched left paren; expected a '"+literalStrGroupClose+"' here", next)
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
					op:    "<=",
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
					op:    ">=",
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
					op:    "<",
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
					op:    ">",
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
					op:    "!=",
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
					op:    "==",
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
					op:    "=",
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
					op:    "+=",
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
					op:    "-=",
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
					op:      "!",
					operand: left,
				},
			},
			source: lex,
		}, nil
	case tsOpInc:
		return &astNode{
			opGroup: &operatorGroupNode{
				unaryOp: &unaryOperatorGroupNode{
					op:      "++",
					operand: left,
				},
			},
			source: lex,
		}, nil
	case tsOpDec:
		return &astNode{
			opGroup: &operatorGroupNode{
				unaryOp: &unaryOperatorGroupNode{
					op:      "--",
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
					op:    "+",
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
					op:    "-",
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
					op:    "*",
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
					op:    "/",
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

func (lex token) MarshalBinary() ([]byte, error) {
	var data []byte

	data = append(data, encBinaryString(lex.lexeme)...)
	data = append(data, encBinary(lex.class)...)
	data = append(data, encBinaryInt(lex.pos)...)
	data = append(data, encBinaryInt(lex.line)...)
	data = append(data, encBinaryString(lex.fullLine)...)

	return data, nil
}

func (lex *token) UnmarshalBinary(data []byte) error {
	var err error
	var bytesRead int

	lex.lexeme, bytesRead, err = decBinaryString(data)
	if err != nil {
		return err
	}
	data = data[bytesRead:]

	bytesRead, err = decBinary(data, &lex.class)
	if err != nil {
		return err
	}
	data = data[bytesRead:]

	lex.pos, bytesRead, err = decBinaryInt(data)
	if err != nil {
		return err
	}
	data = data[bytesRead:]

	lex.line, bytesRead, err = decBinaryInt(data)
	if err != nil {
		return err
	}
	data = data[bytesRead:]

	lex.fullLine, _, err = decBinaryString(data)
	if err != nil {
		return err
	}
	//data = data[bytesRead:]

	return nil
}

func (sym tokenClass) MarshalBinary() ([]byte, error) {
	var data []byte

	data = append(data, encBinaryString(sym.id)...)
	data = append(data, encBinaryString(sym.human)...)
	data = append(data, encBinaryInt(sym.lbp)...)

	return data, nil
}

func (sym *tokenClass) UnmarshalBinary(data []byte) error {
	var err error
	var bytesRead int

	sym.id, bytesRead, err = decBinaryString(data)
	if err != nil {
		return err
	}
	data = data[bytesRead:]

	sym.human, bytesRead, err = decBinaryString(data)
	if err != nil {
		return err
	}
	data = data[bytesRead:]

	sym.lbp, _, err = decBinaryInt(data)
	if err != nil {
		return err
	}
	// data = data[bytesRead:]

	return nil
}
