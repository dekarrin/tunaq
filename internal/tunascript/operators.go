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
	token opTokenType
	pos   int
	line  int
}

type tokenStream struct {
	tokens []opTokenizedLexeme
	cur    int
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

type opTokenType int

const (
	opTokenUndefined opTokenType = iota
	opTokenNumber
	opTokenBool
	opTokenUnquotedString
	opTokenQuotedString // NOTE: the opTokenQuotedString token only exists in lexer pass 1.
	opTokenAdd          // Pass 2 eliminates it and combines its body with any consecutive
	opTokenSub          // quote strings and general unparsed TS to produce single opTokenUnparsedText
	opTokenMult
	opTokenDiv
	opTokenInc
	opTokenDec
	opTokenLeftParen
	opTokenRightParen
	opTokenAnd
	opTokenOr // TODO: SWAP OR SIGN AND STRING ESCAPE
	opTokenNot
	opTokenSeparator
	opTokenIdentifier
	opTokenEndOfText
)

func (tok opTokenType) String() string {
	switch tok {
	case opTokenUndefined:
		return "LEX_UNDEFINED"
	case opTokenUnquotedString:
		return "LEX_UNQUOTED_STRING"
	case opTokenQuotedString:
		return "LEX_QUOTED_STRING"
	case opTokenAdd:
		return "LEX_PLUS"
	case opTokenSub:
		return "LEX_MINUS"
	case opTokenMult:
		return "LEX_MULTIPLY"
	case opTokenDiv:
		return "LEX_DIVIDE"
	case opTokenInc:
		return "LEX_DOUBLE_PLUS"
	case opTokenDec:
		return "LEX_DOUBLE_MINUS"
	case opTokenLeftParen:
		return "LEX_LEFT_PAREN"
	case opTokenRightParen:
		return "LEX_RIGHT_PAREN"
	case opTokenAnd:
		return "LEX_DOUBLE_AMPERSAND"
	case opTokenOr:
		return "LEX_DOUBLE_COLON"
	case opTokenNot:
		return "LEX_EXCLAMATION_MARK"
	case opTokenSeparator:
		return "LEX_COMMA"
	case opTokenIdentifier:
		return "LEX_IDENTIFIER"
	case opTokenEndOfText:
		return "LEX_EOT"
	case opTokenNumber:
		return "LEX_NUMBER"
	case opTokenBool:
		return "LEX_BOOL"
	default:
		return fmt.Sprintf("LEX_UNKNOWN(%d)", int(tok))
	}
}

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
		right, err := parseOpExpression(tokens, lex.lbp())
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
		right, err := parseOpExpression(tokens, lex.lbp())
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
		right, err := parseOpExpression(tokens, lex.lbp())
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
		right, err := parseOpExpression(tokens, lex.lbp())
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

// left binding power values for pratt parsing. higher vals = tighter binding
//
// TODO: these have no reason to ever need prior value or token stream so should
// be associated with the opToken, not the opTokenizedLexeme probably
func (lex opTokenizedLexeme) lbp() int {
	switch lex.token {
	case opTokenDec:
		fallthrough
	case opTokenInc:
		return 150
	case opTokenAdd:
		fallthrough
	case opTokenSub:
		return 10
	case opTokenDiv:
		fallthrough
	case opTokenMult:
		return 20
	case opTokenLeftParen:
		return 100
	default:
		return 0
	}
}
