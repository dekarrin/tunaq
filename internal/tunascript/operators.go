package tunascript

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
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
		if tokens.Peek().token != opTokenRightParen {
			for {
				arg, err := parseOpExpression(tokens, 0)
				if err != nil {
					return nil, err
				}
				callArgs = append(callArgs, arg)

				if tokens.Peek().token == opTokenSeparator {
					tokens.Next() // toss of the separator and parse the next
				} else {
					break
				}
			}

			nextTok := tokens.Next()
			if nextTok.token != opTokenRightParen {
				return nil, fmt.Errorf("unexpected token %s; expected \")\"", nextTok.token.String())
			}
		}
	default:
		return nil, nil
	}
	return nil, nil
}

// left binding power values for pratt parsing. higher vals = tighter binding
//
// TODO: these have no reason to ever need prior value or token stream so should
// be associated with the opToken, not the opTokenizedLexeme probably
func (lex opTokenizedLexeme) lbp() int {
	switch lex.token {
	case opTokenAdd:
		return 10
	case opTokenSub:
		return 10
	case opTokenDiv:
		return 20
	case opTokenMult:
		return 20
	default:
		return 0
	}
}

// InterpretOpText returns the interpreted TS op text.
func InterpretOpText(s string) (string, error) {
	lexed, err := LexOperationText(s)
	if err != nil {
		return "", err
	}

	// TODO: need debug
	ast, err := parseOpExpression(&lexed, 0)
	if err != nil {
		return "", err
	}

	fullTree := opAST{
		nodes: []*opASTNode{ast},
	}

	output := executeOpTree(fullTree)

	return output, nil
}

// just apply to the parse tree
func executeOpTree(ast opAST) string {
	var sb strings.Builder

	for i := 0; i < len(ast.nodes); i++ {
		node := ast.nodes[i]

		if node.value != nil {
			if node.value.quotedStringVal != nil {
				// should never happen
				// TODO: since lexer is designed for no distinction between unparsed text and quoted string
				// eliminate this branch from the AST entirely
				sb.WriteString(*node.value.quotedStringVal)
			} else if node.value.unquotedStringVal != nil {
				sb.WriteString(*node.value.unquotedStringVal)
			} else if node.value.boolVal != nil {
				sb.WriteString(fmt.Sprintf("%t", *node.value.boolVal))
			} else if node.value.numVal != nil {
				sb.WriteString(fmt.Sprintf("%d", *node.value.numVal))
			} else {
				panic("empty value node in AST")
			}
		} else if node.fn != nil {
			sb.WriteString(node.fn.name)
			sb.WriteRune('(')

			for i := range node.fn.args {
				toExec := opAST{
					nodes: []*opASTNode{node.fn.args[i]},
				}
				insert := executeOpTree(toExec)
				sb.WriteString(insert)
				if i+1 < len(node.fn.args) {
					sb.WriteRune(',')
					sb.WriteRune(' ')
				}
			}

			sb.WriteRune(')')
		} else if node.flag != nil {
			sb.WriteString(node.flag.name)
		} else if node.group != nil {
			sb.WriteRune('(')
			toExec := opAST{
				nodes: []*opASTNode{node.group.expr},
			}
			insert := executeOpTree(toExec)
			sb.WriteString(insert)
			sb.WriteRune(')')
		} else if node.opGroup != nil {
			if node.opGroup.infixOp != nil {
				op := node.opGroup.infixOp.op
				leftExec := opAST{
					nodes: []*opASTNode{node.opGroup.infixOp.left},
				}
				rightExec := opAST{
					nodes: []*opASTNode{node.opGroup.infixOp.right},
				}

				leftInsert := executeOpTree(leftExec)
				rightInsert := executeOpTree(rightExec)

				var opFunc string
				if op == "+" {
					opFunc = "ADD"
				} else if op == "-" {
					opFunc = "SUB"
				} else if op == "/" {
					opFunc = "DIV"
				} else if op == "*" {
					opFunc = "MULT"
				} else if op == "&&" {
					opFunc = "AND"
				} else if op == "::" {
					opFunc = "OR"
				} else {
					// should never happen
					panic(fmt.Sprintf("unknown binary operator %q", op))
				}

				sb.WriteString(opFunc)
				sb.WriteRune('(')
				sb.WriteString(strings.TrimSpace(leftInsert))
				sb.WriteRune(',')
				sb.WriteRune(' ')
				sb.WriteString(strings.TrimSpace(rightInsert))
				sb.WriteRune(')')
			} else if node.opGroup.unaryOp != nil {
				op := node.opGroup.unaryOp.op
				toExec := opAST{
					nodes: []*opASTNode{node.opGroup.unaryOp.operand},
				}
				toInsert := executeOpTree(toExec)
				var opFunc string
				if op == "!" {
					opFunc = "NOT"
				} else if op == "++" {
					opFunc = "INC"
				} else if op == "--" {
					opFunc = "DEC"
				} else {
					// should never happen
					panic(fmt.Sprintf("unknown unary operator %q", op))
				}

				sb.WriteString(opFunc)
				sb.WriteRune('(')
				sb.WriteString(toInsert)
				sb.WriteRune(')')
			} else {
				// should never happen
				panic("opGroup node in AST does not assign infix or unary")
			}
		} else {
			// should never happen
			panic("empty AST node")
		}
	}

	return sb.String()
}

func parseOpExpression(stream *tokenStream, rbp int) (*opASTNode, error) {
	var err error

	if stream.Remaining() < 1 {
		return nil, fmt.Errorf("no tokens to parse")
	}

	t := stream.Next()
	left, err := t.nud(stream)
	if err != nil {
		return nil, err
	}
	if left == nil {
		return nil, fmt.Errorf("%s cannot appear at start of expression", t.token.String())
	}

	for rbp < stream.Peek().lbp() {
		t = stream.Next()
		left, err = t.led(left, stream)
		if err != nil {
			return nil, err
		}
	}
	return left, nil

}

func LexOperationText(s string) (tokenStream, error) {
	sRunes := []rune(s)

	var tokens []opTokenizedLexeme

	curLine := 1
	curLinePos := 1

	var curToken opTokenizedLexeme
	var sb strings.Builder

	var escaping bool

	type lexMode int

	const (
		lexDefault lexMode = iota
		lexIdent
		lexString
	)

	mode := lexDefault

	flushCurrentPendingToken := func() {
		if sb.Len() > 0 {
			curToken.value = sb.String()
			sb.Reset()

			// is the cur token literally one of the bool values?
			vUp := strings.ToUpper(curToken.value)
			if patBool.MatchString(vUp) {
				curToken.token = opTokenBool
			}
			if patNum.MatchString(vUp) {
				curToken.token = opTokenNumber
			}

			tokens = append(tokens, curToken)
			curToken = opTokenizedLexeme{}
		}
	}

	for i := 0; i < len(sRunes); i++ {
		ch := sRunes[i]

		switch mode {
		case lexIdent:
			if ('A' <= ch && ch <= 'Z') || ('a' <= ch && ch <= 'z') || ('0' <= ch && ch <= '9') || ch == '_' {
				sb.WriteRune(ch)
			} else {
				curToken.value = sb.String()
				sb.Reset()
				tokens = append(tokens, curToken)
				curToken = opTokenizedLexeme{}
				mode = lexDefault
				i-- // re-lex in normal mode
			}
		case lexString:
			if !escaping && ch == '|' {
				sb.WriteRune('|')
				flushCurrentPendingToken()
				mode = lexDefault
				sb.Reset()
			} else if !escaping && ch == '\\' {
				// preserve ALL escape sequences not directly linked to
				// operators as further parse passes may need to interpret
				// them
				escaping = true
				sb.WriteRune('\\')
			} else {
				escaping = false
				sb.WriteRune(ch)
			}
		case lexDefault:
			if !escaping && ch == '|' {
				flushCurrentPendingToken()

				// we are entering a string, set type and current position
				// (value set on a deferred basis once string is complete)
				curToken.pos = curLinePos
				curToken.line = curLine
				curToken.token = opTokenQuotedString
				mode = lexString
				sb.WriteRune('|')
			} else if !escaping && ch == '$' {
				flushCurrentPendingToken()

				// we are entering an identifier, set type and current position
				// (value set on a deferred basis once identifier is complete)
				curToken.pos = curLinePos
				curToken.line = curLine
				curToken.token = opTokenIdentifier
				mode = lexIdent
				sb.WriteRune('$')
			} else if !escaping && ch == '\\' {
				escaping = true
			} else if ch == ',' {
				if escaping {
					sb.WriteRune(',')
				} else {
					flushCurrentPendingToken()
					curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenSeparator, value: ","}
					tokens = append(tokens, curToken)
					curToken = opTokenizedLexeme{}
				}
			} else if ch == '+' {
				if escaping {
					sb.WriteRune('+')
				} else {
					flushCurrentPendingToken()
					if i+1 < len(sRunes) && sRunes[i+1] == '+' {
						// it is double-plus
						curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenInc, value: "++"}
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
						i++
					} else {
						// it is a plus
						curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenAdd, value: "+"}
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
					}
				}
			} else if ch == '-' {
				if escaping {
					sb.WriteRune('-')
				} else {
					// unary binding will be handled by parsing, no need to lookahead
					// at this time.

					flushCurrentPendingToken()
					if i+1 < len(sRunes) && sRunes[i+1] == '-' {
						// it is double-minus
						curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenDec, value: "--"}
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
						i++
					} else {
						// it is a minus
						curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenSub, value: "-"}
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
					}
				}
			} else if ch == '/' {
				if escaping {
					sb.WriteRune('/')
				} else {
					flushCurrentPendingToken()
					curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenDiv, value: "/"}
					tokens = append(tokens, curToken)
					curToken = opTokenizedLexeme{}
				}
			} else if ch == '*' {
				if escaping {
					sb.WriteRune('*')
				} else {
					if sb.Len() > 0 {
						curToken.value = sb.String()
						sb.Reset()
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
					}
					curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenMult, value: "*"}
					tokens = append(tokens, curToken)
					curToken = opTokenizedLexeme{}
				}
			} else if ch == '!' {
				if escaping {
					sb.WriteRune('!')
				} else {
					flushCurrentPendingToken()
					curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenNot, value: "!"}
					tokens = append(tokens, curToken)
					curToken = opTokenizedLexeme{}
				}
			} else if ch == '(' {
				if escaping {
					sb.WriteRune('(')
				} else {
					flushCurrentPendingToken()
					curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenLeftParen, value: "("}
					tokens = append(tokens, curToken)
					curToken = opTokenizedLexeme{}
				}
			} else if ch == ')' {
				if escaping {
					sb.WriteRune(')')
				} else {
					flushCurrentPendingToken()
					curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenRightParen, value: ")"}
					tokens = append(tokens, curToken)
					curToken = opTokenizedLexeme{}
				}
			} else if ch == '&' {
				if escaping {
					sb.WriteRune('&')
				} else if i+1 < len(sRunes) && sRunes[i] == '&' {
					flushCurrentPendingToken()
					curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenAnd, value: "&&"}
					tokens = append(tokens, curToken)
					curToken = opTokenizedLexeme{}
					i++
				} else {
					sb.WriteRune('&')
				}
			} else if ch == ':' {
				if escaping {
					sb.WriteRune(':')
				} else if i+1 < len(sRunes) && sRunes[i] == ':' {
					flushCurrentPendingToken()
					curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenOr, value: "::"}
					tokens = append(tokens, curToken)
					curToken = opTokenizedLexeme{}
					i++
				} else {
					sb.WriteRune(':')
				}
			} else {

				// do not include whitespace unless it is escaped
				if escaping || !unicode.IsSpace(ch) {
					// is this the first non empty char? set the props for an unquoted string,
					// the default.
					if sb.Len() == 0 {
						curToken.line = curLine
						curToken.pos = curLinePos
						curToken.token = opTokenUnquotedString
					}
					sb.WriteRune(ch)
				}
			}
		}

		curLinePos++
		if ch == '\n' {
			curLine++
			curLinePos = 1
		}
	}

	// do we have leftover parsing string? this is a lexing error, immediately
	// end
	if mode == lexString {
		return tokenStream{}, fmt.Errorf("unterminated quoted string")
	}

	// do we have leftover unparsed text? add it to the tokens list
	flushCurrentPendingToken()

	// add special EOT token
	tokens = append(tokens, opTokenizedLexeme{
		pos:   curLinePos,
		line:  curLine,
		token: opTokenEndOfText,
	})

	return tokenStream{tokens: tokens}, nil
}
