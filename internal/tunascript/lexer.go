package tunascript

import (
	"strings"
	"unicode"
)

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

	var currentfullLine = readFullLine(sRunes)
	flushCurrentPendingToken := func() {
		if sb.Len() > 0 {
			curToken.value = sb.String()
			curToken.fullLine = currentfullLine
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

		// if it's a newline for any reason, get the next line for the current
		// one
		if ch == '\n' {
			//
			currentfullLine = readFullLine(sRunes[i+1:])
		}

		switch mode {
		case lexIdent:
			if ('A' <= ch && ch <= 'Z') || ('a' <= ch && ch <= 'z') || ('0' <= ch && ch <= '9') || ch == '_' {
				sb.WriteRune(ch)
			} else {
				curToken.value = sb.String()
				sb.Reset()
				curToken.fullLine = currentfullLine
				tokens = append(tokens, curToken)
				curToken = opTokenizedLexeme{}
				mode = lexDefault
				i-- // re-lex in normal mode
			}
		case lexString:
			if !escaping && ch == '@' {
				sb.WriteRune('@')
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
			if !escaping && ch == '@' {
				flushCurrentPendingToken()

				// we are entering a string, set type and current position
				// (value set on a deferred basis once string is complete)
				curToken.pos = curLinePos
				curToken.line = curLine
				curToken.token = opTokenQuotedString
				mode = lexString
				sb.WriteRune('@')
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
					curToken.fullLine = currentfullLine
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
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
						i++
					} else if i+1 < len(sRunes) && sRunes[i+1] == '=' {
						// it is inc-by
						curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenIncSet, value: "-="}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
						i++
					} else {
						// it is a plus
						curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenAdd, value: "+"}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
					}
				}
			} else if ch == '-' {
				if escaping {
					sb.WriteRune('-')
				} else {
					flushCurrentPendingToken()
					if i+1 < len(sRunes) && sRunes[i+1] == '-' {
						// it is double-minus
						curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenDec, value: "--"}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
						i++
					} else if i+1 < len(sRunes) && sRunes[i+1] == '=' {
						// it is dec-by
						curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenDecSet, value: "-="}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
						i++
					} else {
						// it is a minus
						curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenSub, value: "-"}
						curToken.fullLine = currentfullLine
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
					curToken.fullLine = currentfullLine
					tokens = append(tokens, curToken)
					curToken = opTokenizedLexeme{}
				}
			} else if ch == '*' {
				if escaping {
					sb.WriteRune('*')
				} else {
					flushCurrentPendingToken()
					curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenMult, value: "*"}
					curToken.fullLine = currentfullLine
					tokens = append(tokens, curToken)
					curToken = opTokenizedLexeme{}
				}
			} else if ch == '!' {
				if escaping {
					sb.WriteRune('!')
				} else {
					flushCurrentPendingToken()

					if i+1 < len(sRunes) && sRunes[i+1] == '=' {
						// it is not-equal
						curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenIsNot, value: "!="}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
						i++
					} else {
						// it is a negation
						curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenNot, value: "!"}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
					}
				}
			} else if ch == '<' {
				if escaping {
					sb.WriteRune('<')
				} else {
					flushCurrentPendingToken()

					if i+1 < len(sRunes) && sRunes[i+1] == '=' {
						// it is lt/eq
						curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenLessThanIs, value: "<="}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
						i++
					} else {
						// it is less-than
						curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenLessThan, value: "<"}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
					}
				}
			} else if ch == '>' {
				if escaping {
					sb.WriteRune('>')
				} else {
					flushCurrentPendingToken()

					if i+1 < len(sRunes) && sRunes[i+1] == '=' {
						// it is gt/eq
						curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenGreaterThanIs, value: ">="}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
						i++
					} else {
						// it is greater-than
						curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenGreaterThan, value: ">"}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
					}
				}
			} else if ch == '(' {
				if escaping {
					sb.WriteRune('(')
				} else {
					flushCurrentPendingToken()
					curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenLeftParen, value: "("}
					curToken.fullLine = currentfullLine
					tokens = append(tokens, curToken)
					curToken = opTokenizedLexeme{}
				}
			} else if ch == ')' {
				if escaping {
					sb.WriteRune(')')
				} else {
					// if we are not the parent this is an error.
					flushCurrentPendingToken()
					curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenRightParen, value: ")"}
					curToken.fullLine = currentfullLine
					tokens = append(tokens, curToken)
					curToken = opTokenizedLexeme{}
				}
			} else if ch == '&' {
				if escaping {
					sb.WriteRune('&')
				} else if i+1 < len(sRunes) && sRunes[i+1] == '&' {
					flushCurrentPendingToken()
					curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenAnd, value: "&&"}
					curToken.fullLine = currentfullLine
					tokens = append(tokens, curToken)
					curToken = opTokenizedLexeme{}
					i++
				} else {
					sb.WriteRune('&')
				}
			} else if ch == '|' {
				if escaping {
					sb.WriteRune('|')
				} else if i+1 < len(sRunes) && sRunes[i+1] == '|' {
					flushCurrentPendingToken()
					curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenOr, value: "||"}
					curToken.fullLine = currentfullLine
					tokens = append(tokens, curToken)
					curToken = opTokenizedLexeme{}
					i++
				} else {
					sb.WriteRune('|')
				}
			} else if ch == '=' {
				if escaping {
					sb.WriteRune('=')
				} else {
					// unary binding will be handled by parsing, no need to lookahead
					// at this time.

					flushCurrentPendingToken()
					if i+1 < len(sRunes) && sRunes[i+1] == '=' {
						// it is double-equals
						curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenIs, value: "=="}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
						i++
					} else {
						// it is an equals
						curToken = opTokenizedLexeme{pos: curLinePos, line: curLine, token: opTokenSet, value: "="}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = opTokenizedLexeme{}
					}
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
		return tokenStream{}, SyntaxError{
			message: "unterminated '@'-string; missing a second '@'",
		}
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

func readFullLine(sRunes []rune) string {
	var lineBuilder strings.Builder
	for i := 0; i < len(sRunes) && sRunes[i] != '\n'; i++ {
		lineBuilder.WriteRune(sRunes[i])
	}
	return lineBuilder.String()
}
