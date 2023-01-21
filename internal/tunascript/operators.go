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

type opTokenizedLexeme struct {
	value    string
	token    symbol
	pos      int
	line     int
	fullLine string
}

type tokenStream struct {
	tokens []opTokenizedLexeme
	cur    int
}

// a type of token
type symbol struct {
	id    string
	human string
	lbp   int
}

// TODO: Do The Unmarshal Function Thing With The Operator Data Objects. Or
// Structs. Or Something Like That.

func (s symbol) String() string {
	return s.id
}

// Human returns human readable name of the string.
func (s symbol) Human() string {
	return s.human
}

func (ts *tokenStream) Next() opTokenizedLexeme {
	n := ts.tokens[ts.cur]
	ts.cur++
	return n
}

func (ts *tokenStream) Peek() opTokenizedLexeme {
	return ts.tokens[ts.cur]
}

func (ts tokenStream) Len() int {
	return len(ts.tokens)
}

func (ts tokenStream) Remaining() int {
	return len(ts.tokens) - ts.cur
}

var (
	opTokenUndefined      = symbol{"TS_UNDEFINED", "undefined", 0}
	opTokenNumber         = symbol{"TS_NUMBER", "number", 0}
	opTokenBool           = symbol{"TS_BOOL", "boolean value", 0}
	opTokenUnquotedString = symbol{"TS_UNQUOTED_STRING", "text value", 0}
	opTokenQuotedString   = symbol{"TS_QUOTED_STRING", "@-text value", 0}
	opTokenIs             = symbol{"TS_IS", "'=='", 5}
	opTokenIsNot          = symbol{"TS_IS_NOT", "'!='", 5}
	opTokenLessThan       = symbol{"TS_LESS_THAN", "'<'", 5}
	opTokenGreaterThan    = symbol{"TS_GREATER_THAN_IS", "'>'", 5}
	opTokenLessThanIs     = symbol{"TS_LESS_THEN_IS", "'<='", 5}
	opTokenGreaterThanIs  = symbol{"TS_GREATER_THAN_IS", "'>='", 5}
	opTokenSet            = symbol{"TS_SET", "'='", 0}
	opTokenAdd            = symbol{"TS_ADD", "'+'", 10}
	opTokenSub            = symbol{"TS_SUB", "'-'", 10}
	opTokenMult           = symbol{"TS_MULT", "'*'", 20}
	opTokenDiv            = symbol{"TS_DIV", "'/'", 20}
	opTokenIncSet         = symbol{"TS_INCSET", "'+='", 90}
	opTokenDecSet         = symbol{"TS_DECSET", "'-='", 90}
	opTokenInc            = symbol{"TS_INC", "'++'", 150}
	opTokenDec            = symbol{"TS_DEC", "'--'", 150}
	opTokenLeftParen      = symbol{"TS_LEFT_PAREN", "'('", 100}
	opTokenRightParen     = symbol{"TS_RIGHT_PAREN", "')'", 0}
	opTokenAnd            = symbol{"TS_AND", "'&&'", 0}
	opTokenOr             = symbol{"TS_OR", "'||'", 0}
	opTokenNot            = symbol{"TS_NOT", "'!'", 0}
	opTokenSeparator      = symbol{"TS_SEPARATOR", "','", 0}
	opTokenIdentifier     = symbol{"TS_IDENTIFIER", "identifier", 0}
	opTokenEndOfText      = symbol{"TS_END_OF_TEXT", "end of text", 0}
)

// null denotation values for pratt parsing
//
// return nil for "this token cannot appear at start of lang construct", or
// "represents end of text."
func (lex opTokenizedLexeme) nud(tokens *tokenStream) (*astNode, error) {
	switch lex.token {
	case opTokenUnquotedString:
		return &astNode{
			value: &valueNode{
				unquotedStringVal: &lex.value,
			},
			source: lex,
		}, nil
	case opTokenQuotedString:
		return &astNode{
			value: &valueNode{
				quotedStringVal: &lex.value,
			},
			source: lex,
		}, nil
	case opTokenSub:
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
	case opTokenNumber:
		num, err := strconv.Atoi(lex.value)
		if err != nil {
			panic(fmt.Sprintf("got non-integer value %q in LEX_NUMBER token, should never happen", lex.value))
		}
		return &astNode{
			value: &valueNode{
				numVal: &num,
			},
			source: lex,
		}, nil
	case opTokenBool:
		vUp := strings.ToUpper(lex.value)

		var boolVal bool

		if vUp == "TRUE" || vUp == "ON" || vUp == "YES" {
			boolVal = true
		} else if vUp == "FALSE" || vUp == "OFF" || vUp == "NO" {
			boolVal = false
		} else {
			panic(fmt.Sprintf("got non-bool value %q in LEX_BOOL token, should never happen", lex.value))
		}

		return &astNode{
			value: &valueNode{
				boolVal: &boolVal,
			},
			source: lex,
		}, nil
	case opTokenIdentifier:
		flagName := strings.ToUpper(lex.value)
		return &astNode{
			flag: &flagNode{
				name: flagName,
			},
			source: lex,
		}, nil
	case opTokenLeftParen:
		expr, err := parseOpExpression(tokens, 0)
		if err != nil {
			return nil, err
		}
		next := tokens.Next()
		if next.token != opTokenRightParen {
			return nil, syntaxErrorFromLexeme("unmatched left paren; expected a ')' here", next)
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
func (lex opTokenizedLexeme) led(left *astNode, tokens *tokenStream) (*astNode, error) {
	switch lex.token {
	case opTokenLessThanIs:
		right, err := parseOpExpression(tokens, lex.token.lbp)
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
	case opTokenGreaterThanIs:
		right, err := parseOpExpression(tokens, lex.token.lbp)
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
	case opTokenLessThan:
		right, err := parseOpExpression(tokens, lex.token.lbp)
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
	case opTokenGreaterThan:
		right, err := parseOpExpression(tokens, lex.token.lbp)
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
	case opTokenIsNot:
		right, err := parseOpExpression(tokens, lex.token.lbp)
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
	case opTokenIs:
		right, err := parseOpExpression(tokens, lex.token.lbp)
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
	case opTokenSet:
		right, err := parseOpExpression(tokens, lex.token.lbp)
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
	case opTokenIncSet:
		right, err := parseOpExpression(tokens, lex.token.lbp)
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
	case opTokenDecSet:
		right, err := parseOpExpression(tokens, lex.token.lbp)
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
	case opTokenNot:
		return &astNode{
			opGroup: &operatorGroupNode{
				unaryOp: &unaryOperatorGroupNode{
					op:      "!",
					operand: left,
				},
			},
			source: lex,
		}, nil
	case opTokenInc:
		return &astNode{
			opGroup: &operatorGroupNode{
				unaryOp: &unaryOperatorGroupNode{
					op:      "++",
					operand: left,
				},
			},
			source: lex,
		}, nil
	case opTokenDec:
		return &astNode{
			opGroup: &operatorGroupNode{
				unaryOp: &unaryOperatorGroupNode{
					op:      "--",
					operand: left,
				},
			},
			source: lex,
		}, nil
	case opTokenAdd:
		right, err := parseOpExpression(tokens, lex.token.lbp)
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
	case opTokenSub:
		right, err := parseOpExpression(tokens, lex.token.lbp)
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
	case opTokenMult:
		right, err := parseOpExpression(tokens, lex.token.lbp)
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
	case opTokenDiv:
		right, err := parseOpExpression(tokens, lex.token.lbp)
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
	case opTokenLeftParen:
		// binary op '(' binds to expr
		callArgs := []*astNode{}

		if left.flag == nil {
			return nil, syntaxErrorFromLexeme(fmt.Sprintf("unexpected %s\n(expected it to be after a function name or to group expressions)", lex.token.human), lex)
		}

		if tokens.Peek().token != opTokenRightParen {
			for {
				arg, err := parseOpExpression(tokens, 0)
				if err != nil {
					return nil, err
				}
				callArgs = append(callArgs, arg)

				if tokens.Peek().token == opTokenSeparator {
					tokens.Next() // toss off the separator and prep to parse the next one
				} else {
					break
				}
			}
		}

		nextTok := tokens.Next()
		if nextTok.token != opTokenRightParen {
			return nil, syntaxErrorFromLexeme(fmt.Sprintf("unexpected %s\n(expected a ')' to close the previous ')')", nextTok.token.human), lex)
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

func (lex opTokenizedLexeme) MarshalBinary() ([]byte, error) {
	var data []byte

	data = append(data, encBinaryString(lex.value)...)
	data = append(data, encBinary(lex.token)...)
	data = append(data, encBinaryInt(lex.pos)...)
	data = append(data, encBinaryInt(lex.line)...)
	data = append(data, encBinaryString(lex.fullLine)...)

	return data, nil
}

func (lex *opTokenizedLexeme) UnmarshalBinary(data []byte) error {
	var err error
	var bytesRead int

	lex.value, bytesRead, err = decBinaryString(data)
	if err != nil {
		return err
	}
	data = data[bytesRead:]

	bytesRead, err = decBinary(data, &lex.token)
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

func (sym symbol) MarshalBinary() ([]byte, error) {
	var data []byte

	data = append(data, encBinaryString(sym.id)...)
	data = append(data, encBinaryString(sym.human)...)
	data = append(data, encBinaryInt(sym.lbp)...)

	return data, nil
}

func (sym *symbol) UnmarshalBinary(data []byte) error {
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