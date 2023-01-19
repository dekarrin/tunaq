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

type opAST struct {
	nodes []*opASTNode
}

type opASTNode struct {
	value   *opASTValueNode
	fn      *opASTFuncNode
	flag    *opASTFlagNode
	group   *opASTGroupNode
	opGroup *opASTOperationGroupNode
	source  opTokenizedLexeme
}

type opASTFlagNode struct {
	name string
}

type opASTFuncNode struct {
	name string
	args []*opASTNode
}

type opASTValueNode struct {
	quotedStringVal   *string
	unquotedStringVal *string
	numVal            *int
	boolVal           *bool
}

type opASTGroupNode struct {
	expr *opASTNode
}

type opASTOperationGroupNode struct {
	unaryOp *opASTUnaryOperatorGroupNode
	infixOp *opASTBinaryOperatorGroupNode
}

type opASTUnaryOperatorGroupNode struct {
	op      string
	operand *opASTNode
}

type opASTBinaryOperatorGroupNode struct {
	op    string
	left  *opASTNode
	right *opASTNode
}

type opTokenizedLexeme struct {
	value string
	token symbol
	pos   int
	line  int
}

type tokenStream struct {
	tokens []opTokenizedLexeme
	cur    int
}

// a type of token
type symbol struct {
	id   string
	repr string
	lbp  int
}

func (s symbol) String() string {
	return s.id
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
	opTokenUndefined      = symbol{"TS_UNDEFINED", "", 0}
	opTokenNumber         = symbol{"TS_NUMBER", "", 0}
	opTokenBool           = symbol{"TS_BOOL", "", 0}
	opTokenUnquotedString = symbol{"TS_UNQUOTED_STRING", "", 0}
	opTokenQuotedString   = symbol{"TS_QUOTED_STRING", "", 0}
	opTokenAdd            = symbol{"TS_ADD", "+", 10}
	opTokenSub            = symbol{"TS_SUB", "-", 10}
	opTokenMult           = symbol{"TS_MULT", "*", 20}
	opTokenDiv            = symbol{"TS_DIV", "/", 20}
	opTokenInc            = symbol{"TS_INC", "++", 150}
	opTokenDec            = symbol{"TS_DEC", "--", 150}
	opTokenLeftParen      = symbol{"TS_LEFT_PAREN", "(", 100}
	opTokenRightParen     = symbol{"TS_RIGHT_PAREN", ")", 0}
	opTokenAnd            = symbol{"TS_AND", "&&", 0}
	opTokenOr             = symbol{"TS_OR", "||", 0}
	opTokenNot            = symbol{"TS_NOT", "!", 0}
	opTokenSeparator      = symbol{"TS_SEPARATOR", ",", 0}
	opTokenIdentifier     = symbol{"TS_IDENTIFIER", "", 0}
	opTokenEndOfText      = symbol{"TS_END_OF_TEXT", "", 0}
)

// null denotation values for pratt parsing
//
// return nil for "this token cannot appear at start of lang construct", or
// "represents end of text."
func (lex opTokenizedLexeme) nud(tokens *tokenStream) (*opASTNode, error) {
	switch lex.token {
	case opTokenUnquotedString:
		return &opASTNode{
			value: &opASTValueNode{
				unquotedStringVal: &lex.value,
			},
		}, nil
	case opTokenQuotedString:
		return &opASTNode{
			value: &opASTValueNode{
				quotedStringVal: &lex.value,
			},
		}, nil
	case opTokenNumber:
		num, err := strconv.Atoi(lex.value)
		if err != nil {
			panic(fmt.Sprintf("got non-integer value %q in LEX_NUMBER token, should never happen", lex.value))
		}
		return &opASTNode{
			value: &opASTValueNode{
				numVal: &num,
			},
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

		return &opASTNode{
			value: &opASTValueNode{
				boolVal: &boolVal,
			},
		}, nil
	case opTokenIdentifier:
		flagName := strings.ToUpper(lex.value)
		return &opASTNode{
			flag: &opASTFlagNode{
				name: flagName,
			},
		}, nil
	case opTokenLeftParen:
		expr, err := parseOpExpression(tokens, 0)
		if err != nil {
			return nil, err
		}
		if tokens.Next().token != opTokenRightParen {
			return nil, fmt.Errorf("unmatched left paren")
		}

		return &opASTNode{
			group: &opASTGroupNode{
				expr: expr,
			},
		}, nil
	case opTokenRightParen:
		return nil, nil // TODO: come back to this
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
func (lex opTokenizedLexeme) led(left *opASTNode, tokens *tokenStream) (*opASTNode, error) {
	switch lex.token {
	case opTokenInc:
		return &opASTNode{
			opGroup: &opASTOperationGroupNode{
				unaryOp: &opASTUnaryOperatorGroupNode{
					op:      "++",
					operand: left,
				},
			},
		}, nil
	case opTokenDec:
		return &opASTNode{
			opGroup: &opASTOperationGroupNode{
				unaryOp: &opASTUnaryOperatorGroupNode{
					op:      "--",
					operand: left,
				},
			},
		}, nil
	case opTokenAdd:
		right, err := parseOpExpression(tokens, lex.token.lbp)
		if err != nil {
			return nil, err
		}
		return &opASTNode{
			opGroup: &opASTOperationGroupNode{
				infixOp: &opASTBinaryOperatorGroupNode{
					op:    "+",
					left:  left,
					right: right,
				},
			},
		}, nil
	case opTokenSub:
		right, err := parseOpExpression(tokens, lex.token.lbp)
		if err != nil {
			return nil, err
		}
		return &opASTNode{
			opGroup: &opASTOperationGroupNode{
				infixOp: &opASTBinaryOperatorGroupNode{
					op:    "-",
					left:  left,
					right: right,
				},
			},
		}, nil
	case opTokenMult:
		right, err := parseOpExpression(tokens, lex.token.lbp)
		if err != nil {
			return nil, err
		}
		return &opASTNode{
			opGroup: &opASTOperationGroupNode{
				infixOp: &opASTBinaryOperatorGroupNode{
					op:    "*",
					left:  left,
					right: right,
				},
			},
		}, nil
	case opTokenDiv:
		right, err := parseOpExpression(tokens, lex.token.lbp)
		if err != nil {
			return nil, err
		}
		return &opASTNode{
			opGroup: &opASTOperationGroupNode{
				infixOp: &opASTBinaryOperatorGroupNode{
					op:    "/",
					left:  left,
					right: right,
				},
			},
		}, nil
	case opTokenLeftParen:
		// binary op '(' binds to expr
		callArgs := []*opASTNode{}

		if left.flag == nil {
			return nil, fmt.Errorf("unexpected \"(\" char")
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
			return nil, fmt.Errorf("unexpected token %s; expected \")\"", nextTok.token.String())
		}

		return &opASTNode{
			fn: &opASTFuncNode{
				name: left.flag.name,
				args: callArgs,
			},
		}, nil
	default:
		return nil, nil
	}
}
