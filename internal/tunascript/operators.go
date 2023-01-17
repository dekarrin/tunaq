package tunascript

import (
	"fmt"
	"strings"
	"unicode"
)

// File operators.go contains transpilation functions for turning operator-based
// TunaScript expressions into function-based ones that can be parsed by the
// rest of the system.

type opAST struct {
	nodes []opASTNode
}

type opASTNode struct {
	value   *opASTValueNode
	group   *opASTGroupNode
	opGroup *opASTOperationGroupNode
	source  opTokenizedLexeme
}

type opASTValueNode struct {
	quotedString       *string
	unparsedTunascript *string
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
	token opToken
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

type opToken int

const (
	opTokenUndefined opToken = iota
	opTokenUnparsedText
	opTokenQuotedString
	opTokenAdd
	opTokenSub
	opTokenMult
	opTokenDiv
	opTokenInc
	opTokenDec
	opTokenLeftParen
	opTokenRightParen
	opTokenAnd
	opTokenOr // TODO: SWAP OR SIGN AND STRING ESCAPE
	opTokenNot
	opTokenEndOfText
)

func (tok opToken) String() string {
	switch tok {
	case opTokenUndefined:
		return "LEX_UNDEFINED"
	case opTokenUnparsedText:
		return "LEX_UNPARSED_TEXT"
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
func (lex opTokenizedLexeme) nud() *opASTNode {
	switch lex.token {
	case opTokenUnparsedText:
		return &opASTNode{
			value: &opASTValueNode{
				unparsedTunascript: &lex.value,
			},
		}
	case opTokenQuotedString:
		return &opASTNode{
			value: &opASTValueNode{
				quotedString: &lex.value,
			},
		}
	case opTokenLeftParen:
		return nil // TODO: come back to this
	case opTokenRightParen:
		return nil // TODO: come back to this
	default:
		return nil
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
		right, err := parseOpExpression(tokens, 10)
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
	case opTokenAdd:
		return 10
	default:
		return 0
	}
}

// just apply to the parse tree
func executeOpTree(ast opAST) (string, error) {
	var sb strings.Builder

	for i := 0; i < len(ast.nodes); i++ {
		node := ast.nodes[i]

		if node.value != nil {
			if node.value.quotedString != nil {
				sb.WriteString(*node.value.quotedString)
			} else if node.value.unparsedTunascript != nil {
				sb.WriteString(*node.value.unparsedTunascript)
			}
		}
	}
}

func parseOpExpression(stream *tokenStream, rbp int) (*opASTNode, error) {
	var err error

	if stream.Remaining() < 1 {
		return nil, fmt.Errorf("no tokens to parse")
	}

	t := stream.Next()
	left := t.nud()
	if left == nil {
		return nil, fmt.Errorf("%s cannot appear at start of expression", t.token.String())
	}

	for rbp < t.lbp() {
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
	var inQuotedString bool

	for i := 0; i < len(sRunes); i++ {
		ch := sRunes[i]

		if inQuotedString {
			if !escaping && ch == '|' {
				sb.WriteRune('|')
				curToken.value = sb.String()
				tokens = append(tokens, curToken)
				curToken = opTokenizedLexeme{}
				inQuotedString = false
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
		} else {
			if !escaping && ch == '|' {
				// we are entering a string, set type and current position
				// (value set on a deferred basis once string is complete)
				curToken.pos = curLinePos
				curToken.line = curLine
				curToken.token = opTokenQuotedString
				inQuotedString = true
			} else if !escaping && ch == '\\' {
				escaping = true
			} else if ch == '+' {
				if escaping {
					sb.WriteRune('+')
				} else {
					if sb.Len() > 0 {
						// save the current unparsed text
						curToken.value = sb.String()
						sb.Reset()
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
					}
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
					// do some lookahead to see if this should instead be interpreted as a
					// negative number and thus be a part of 'unparsed text'

					var partOfNumber bool
					for j := i; j < len(sRunes); j++ {
						numCh := sRunes[j]
						if unicode.IsSpace(numCh) {
							continue // no info
						} else if ('0' <= numCh && numCh <= '9') || numCh == '.' {
							partOfNumber = true
						} else {
							break
						}
					}
					if partOfNumber {
						sb.WriteRune('-')
					} else {
						if sb.Len() > 0 {
							curToken.value = sb.String()
							sb.Reset()
							tokens = append(tokens, curToken)
							curToken = opTokenizedLexeme{}
						}
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
				}
			} else if ch == '/' {
				if escaping {
					sb.WriteRune('/')
				} else {
					if sb.Len() > 0 {
						curToken.value = sb.String()
						sb.Reset()
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
					}
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
					if sb.Len() > 0 {
						curToken.value = sb.String()
						sb.Reset()
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
					}
					curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenNot, value: "!"}
					tokens = append(tokens, curToken)
					curToken = opTokenizedLexeme{}
				}
			} else if ch == '(' {
				if escaping {
					sb.WriteRune('(')
				} else {
					if sb.Len() > 0 {
						curToken.value = sb.String()
						sb.Reset()
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
					}
					curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenLeftParen, value: "("}
					tokens = append(tokens, curToken)
					curToken = opTokenizedLexeme{}
				}
			} else if ch == ')' {
				if escaping {
					sb.WriteRune(')')
				} else {
					if sb.Len() > 0 {
						curToken.value = sb.String()
						sb.Reset()
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
					}
					curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenRightParen, value: ")"}
					tokens = append(tokens, curToken)
					curToken = opTokenizedLexeme{}
				}
			} else if ch == '&' {
				if escaping {
					sb.WriteRune('&')
				} else if i+1 < len(sRunes) && sRunes[i] == '&' {
					if sb.Len() > 0 {
						curToken.value = sb.String()
						sb.Reset()
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
					}
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
					if sb.Len() > 0 {
						curToken.value = sb.String()
						sb.Reset()
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
					}
					curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenOr, value: "::"}
					tokens = append(tokens, curToken)
					curToken = opTokenizedLexeme{}
					i++
				} else {
					sb.WriteRune(':')
				}
			} else {
				// is this the first non empty char? set the props for unparsed
				// tunascript
				if sb.Len() == 0 {
					curToken.line = curLine
					curToken.pos = curLinePos
					curToken.token = opTokenUnparsedText
				}
				// if we are escaping but it wasnt a tunascript operator lexeme,
				// we should preserve the escape char so further passes can
				// interpret it
				if escaping {
					sb.WriteRune('\\')
				}
				sb.WriteRune(ch)
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
	if inQuotedString {
		return tokenStream{}, fmt.Errorf("unterminated quoted string")
	}

	// do we have leftover unparsed text? add it to the tokens list
	if sb.Len() > 0 {
		curToken.value = sb.String()
		tokens = append(tokens, curToken)
	}

	// 2nd pass, combine consecutive quoted string and unparsed ts nodes
	var combinedTokens []opTokenizedLexeme
	for i := 0; i < len(tokens); i++ {
		lexeme := tokens[i]

		if lexeme.token == opTokenUnparsedText || lexeme.token == opTokenQuotedString {
			// build up a full lexeme by combining
		}
	}

	// add special EOT token
	tokens = append(tokens, opTokenizedLexeme{
		pos:   curLinePos,
		line:  curLine,
		token: opTokenEndOfText,
	})

	return tokenStream{tokens: tokens}, nil
}
