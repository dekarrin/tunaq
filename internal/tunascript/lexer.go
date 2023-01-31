package tunascript

import (
	"regexp"
	"strings"
	"unicode"
)

const (
	// if newline is put in any of these it will break the lexer detection of
	// position so don't do that

	// also the tests are all hardcoded and that is by design as it implies that
	// if you change something here, you must also update any tests that rely on
	// it.

	literalStrStringQuote     = "@"
	literalStrSeparator       = ","
	literalStrGroupOpen       = "("
	literalStrGroupClose      = ")"
	literalStrIdentifierStart = "$"
	literalStrOpSet           = "="
	literalStrOpIs            = "=="
	literalStrOpIsNot         = "!="
	literalStrOpLessThan      = "<"
	literalStrOpGreaterThan   = ">"
	literalStrOpLessThanIs    = "<="
	literalStrOpGreaterThanIs = ">="
	literalStrOpAnd           = "&&"
	literalStrOpOr            = "||"
	literalStrOpNot           = "!"
	literalStrOpPlus          = "+"
	literalStrOpMinus         = "-"
	literalStrOpMultiply      = "*"
	literalStrOpDivide        = "/"
	literalStrOpIncset        = "+="
	literalStrOpDecset        = "-="
	literalStrOpInc           = "++"
	literalStrOpDec           = "--"
)

var regularModeMatchRules = []matchRule{
	{literal: literalStringQuote, class: tsQuotedString, transitionModeTo: lexString},
	{literal: literalIdentifierStart, class: tsIdentifier, transitionModeTo: lexIdent},
	{literal: literalSeparator, class: tsSeparator, lexeme: literalStrSeparator},
	{literal: literalOpPlus, class: tsOpPlus, lexeme: literalStrOpPlus},
	{literal: literalOpInc, class: tsOpInc, lexeme: literalStrOpInc},
	{literal: literalOpIncset, class: tsOpIncset, lexeme: literalStrOpIncset},
	{literal: literalOpMinus, class: tsOpMinus, lexeme: literalStrOpMinus},
	{literal: literalOpDec, class: tsOpDec, lexeme: literalStrOpDec},
	{literal: literalOpDecset, class: tsOpDecset, lexeme: literalStrOpDecset},
	{literal: literalOpDivide, class: tsOpDivide, lexeme: literalStrOpDivide},
	{literal: literalOpMultiply, class: tsOpMultiply, lexeme: literalStrOpMultiply},
	{literal: literalOpNot, class: tsOpNot, lexeme: literalStrOpNot},
	{literal: literalOpIsNot, class: tsOpIsNot, lexeme: literalStrOpIsNot},
	{literal: literalOpLessThan, class: tsOpLessThan, lexeme: literalStrOpLessThan},
	{literal: literalOpLessThanIs, class: tsOpLessThanIs, lexeme: literalStrOpLessThan},
	{literal: literalOpGreaterThan, class: tsOpGreaterThan, lexeme: literalStrOpGreaterThan},
	{literal: literalOpGreaterThanIs, class: tsOpGreaterThanIs, lexeme: literalStrOpGreaterThanIs},
	{literal: literalGroupOpen, class: tsGroupOpen, lexeme: literalStrGroupOpen},
	{literal: literalGroupClose, class: tsGroupClose, lexeme: literalStrGroupClose},
	{literal: literalOpAnd, class: tsOpAnd, lexeme: literalStrOpAnd},
	{literal: literalOpOr, class: tsOpOr, lexeme: literalStrOpOr},
	{literal: literalOpIs, class: tsOpIs, lexeme: literalStrOpIs},
	{literal: literalOpSet, class: tsOpSet, lexeme: literalStrOpSet},
}

type lexMode int

const (
	lexDefault lexMode = iota
	lexIdent
	lexString
	lexWhitespace
)

type matchRule struct {
	literal          []rune
	class            tokenClass
	lexeme           string
	transitionModeTo lexMode
}

type token struct {
	lexeme   string
	class    tokenClass
	pos      int
	line     int
	fullLine string
}

func (tok token) Equal(o any) bool {
	other, ok := o.(token)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*token)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if tok.lexeme != other.lexeme {
		return false
	} else if !tok.class.Equal(other.class) {
		return false
	} else if tok.pos != other.pos {
		return false
	} else if tok.line != other.line {
		return false
	} else if tok.fullLine != other.fullLine {
		return false
	}

	return true
}

type tokenStream struct {
	tokens []token
	cur    int
}

// a type of token
type tokenClass struct {
	id    string
	human string
	lbp   int
}

func (tc tokenClass) Equal(o any) bool {
	other, ok := o.(tokenClass)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*tokenClass)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	// IDs are always case insensitive and are considered sufficient to prove
	// a tokenClass is equal to another.
	return strings.EqualFold(tc.id, other.id)
}

// TODO: Do The Unmarshal Function Thing With The Operator Data Objects. Or
// Structs. Or Something Like That.

func (class tokenClass) String() string {
	return class.id
}

// Human returns human readable name of the string.
func (class tokenClass) Human() string {
	return class.human
}

func (ts *tokenStream) Next() token {
	n := ts.tokens[ts.cur]
	ts.cur++
	return n
}

func (ts *tokenStream) Peek() token {
	return ts.tokens[ts.cur]
}

func (ts tokenStream) Len() int {
	return len(ts.tokens)
}

func (ts tokenStream) Remaining() int {
	return len(ts.tokens) - ts.cur
}

var (
	literalStringQuote     = []rune(literalStrStringQuote)
	literalSeparator       = []rune(literalStrSeparator)
	literalGroupOpen       = []rune(literalStrGroupOpen)
	literalGroupClose      = []rune(literalStrGroupClose)
	literalIdentifierStart = []rune(literalStrIdentifierStart)
	literalOpIs            = []rune(literalStrOpIs)
	literalOpIsNot         = []rune(literalStrOpIsNot)
	literalOpLessThan      = []rune(literalStrOpLessThan)
	literalOpGreaterThan   = []rune(literalStrOpGreaterThan)
	literalOpLessThanIs    = []rune(literalStrOpLessThanIs)
	literalOpGreaterThanIs = []rune(literalStrOpGreaterThanIs)
	literalOpSet           = []rune(literalStrOpSet)
	literalOpPlus          = []rune(literalStrOpPlus)
	literalOpMinus         = []rune(literalStrOpMinus)
	literalOpMultiply      = []rune(literalStrOpMultiply)
	literalOpDivide        = []rune(literalStrOpDivide)
	literalOpIncset        = []rune(literalStrOpIncset)
	literalOpDecset        = []rune(literalStrOpDecset)
	literalOpInc           = []rune(literalStrOpInc)
	literalOpDec           = []rune(literalStrOpDec)
	literalOpAnd           = []rune(literalStrOpAnd)
	literalOpOr            = []rune(literalStrOpOr)
	literalOpNot           = []rune(literalStrOpNot)
)

var (
	tsSeparator       = tokenClass{"TS_SEPARATOR", "'" + literalStrSeparator + "'", 1}
	tsGroupOpen       = tokenClass{"TS_GROUP_OPEN", "'" + literalStrGroupOpen + "'", 100}
	tsGroupClose      = tokenClass{"TS_GROUP_CLOSE", "'" + literalStrGroupClose + "'", 1}
	tsIdentifier      = tokenClass{"TS_IDENTIFIER", "identifier", 1}
	tsEndOfText       = tokenClass{"TS_END_OF_TEXT", "end of text", 0}
	tsUndefined       = tokenClass{"TS_UNDEFINED", "undefined", 0}
	tsNumber          = tokenClass{"TS_NUMBER", "number", 1}
	tsBool            = tokenClass{"TS_BOOL", "boolean value", 1}
	tsUnquotedString  = tokenClass{"TS_UNQUOTED_STRING", "text value", 1}
	tsQuotedString    = tokenClass{"TS_QUOTED_STRING", literalStrStringQuote + "-text value", 1}
	tsOpIs            = tokenClass{"TS_OP_IS", "'" + literalStrOpIs + "'", 5}
	tsOpIsNot         = tokenClass{"TS_OP_IS_NOT", "'" + literalStrOpIsNot + "'", 5}
	tsOpLessThan      = tokenClass{"TS_OP_LESS_THAN", "'" + literalStrOpLessThan + "'", 5}
	tsOpGreaterThan   = tokenClass{"TS_OP_GREATER_THAN", "'" + literalStrOpGreaterThan + "'", 5}
	tsOpLessThanIs    = tokenClass{"TS_OP_LESS_THEN_IS", "'" + literalStrOpLessThanIs + "'", 5}
	tsOpGreaterThanIs = tokenClass{"TS_OP_GREATER_THAN_IS", "'" + literalStrOpGreaterThanIs + "'", 5}
	tsOpSet           = tokenClass{"TS_OP_SET", "'" + literalStrOpSet + "'", 0}
	tsOpPlus          = tokenClass{"TS_OP_PLUS", "'" + literalStrOpPlus + "'", 10}
	tsOpMinus         = tokenClass{"TS_OP_MINUS", "'" + literalStrOpMinus + "'", 10}
	tsOpMultiply      = tokenClass{"TS_OP_MULTIPLY", "'" + literalStrOpMultiply + "'", 20}
	tsOpDivide        = tokenClass{"TS_OP_DIVIDE", "'" + literalStrOpDivide + "'", 20}
	tsOpIncset        = tokenClass{"TS_OP_INCSET", "'" + literalStrOpIncset + "'", 90}
	tsOpDecset        = tokenClass{"TS_OP_DECSET", "'" + literalStrOpDecset + "'", 90}
	tsOpInc           = tokenClass{"TS_OP_INC", "'" + literalStrOpInc + "'", 150}
	tsOpDec           = tokenClass{"TS_OP_DEC", "'" + literalStrOpDec + "'", 150}
	tsOpAnd           = tokenClass{"TS_OP_AND", "'" + literalStrOpAnd + "'", 1}
	tsOpOr            = tokenClass{"TS_OP_OR", "'" + literalStrOpOr + "'", 1}
	tsOpNot           = tokenClass{"TS_OP_NOT", "'" + literalStrOpNot + "'", 9}
	tsWhitespace      = tokenClass{"TS_WHITESPACE", "whitespace", 0}
)

var (
	boolConsts = []string{"TRUE", "FALSE", "ON", "OFF", "YES", "NO"}
	patNum     = regexp.MustCompile(`^[0-9]+$`)
)

func Lex(s string) (tokenStream, error) {
	sRunes := []rune(s)
	tokens, _, err := lexRunes(sRunes, false)
	return tokens, err
}

// returns the created tokenstream, num runes consumed, and error.
func lexRunes(sRunes []rune, endAtMatchingParen bool) (tokenStream, int, error) {
	var tokens []token

	curLine := 1
	curLinePos := 1

	var curToken token
	var sb strings.Builder

	var escaping bool

	mode := lexDefault

	// track our paren-depth in case endAtMatching is set
	parenDepth := 0
	runesConsumed := len(sRunes)

	var currentfullLine = readFullLine(sRunes)
	flushCurrentPendingToken := func() {
		if sb.Len() > 0 {
			curToken.lexeme = sb.String()
			curToken.fullLine = currentfullLine
			sb.Reset()
			tokens = append(tokens, curToken)
			curToken = token{}
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
		case lexWhitespace:
			if unicode.IsSpace(ch) {
				sb.WriteRune(ch)
			} else {
				curToken.lexeme = sb.String()
				sb.Reset()
				curToken.fullLine = currentfullLine
				tokens = append(tokens, curToken)
				curToken = token{}
				mode = lexDefault
				i-- // re-lex in normal mode
			}
		case lexIdent:
			if ('A' <= ch && ch <= 'Z') || ('a' <= ch && ch <= 'z') || ('0' <= ch && ch <= '9') || ch == '_' {
				sb.WriteRune(ch)
			} else {
				curToken.lexeme = sb.String()
				sb.Reset()
				curToken.fullLine = currentfullLine
				tokens = append(tokens, curToken)
				curToken = token{}
				mode = lexDefault
				i-- // re-lex in normal mode
			}
		case lexString:
			if !escaping && startMatches(sRunes[i:], literalStringQuote) {
				writeRuneSlice(&sb, literalStringQuote)
				flushCurrentPendingToken()
				mode = lexDefault
				i += len(literalStringQuote) - 1
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
			if unicode.IsSpace(ch) {
				flushCurrentPendingToken()
				curToken.pos = curLinePos
				curToken.line = curLine
				curToken.class = tsWhitespace
				mode = lexWhitespace
				sb.WriteRune(ch)
			} else if !escaping && ch == '\\' {
				escaping = true
			} else {
				// need to not unroll this bc we need to try all and then select
				// by disambiguation

				matches := []matchRule{}
				if !escaping {
					for j := range regularModeMatchRules {
						if startMatches(sRunes[i:], regularModeMatchRules[j].literal) {
							matches = append(matches, regularModeMatchRules[j])
						}
					}
				}

				// TODO This entire section can a8solutely be replaced with a regular
				// expression! And it wouldn't need a *lot* of modific8ions. Well,
				// not any that wouldn't 8e worth it, anyways.
				//
				// right but i think its super hard to write a regex for a quoted
				// string that can have backslashes. unless you gave a regex for a
				// modeswap hmmmm....
				//
				// Yeah, we should look into it in the future if we want to
				// optimize.
				//
				// Wait, I looked into it! Mode swap isn't needed at all; look,
				// escape sequences can always be represented as a finite
				// altern8ive production in a CFG rule, right? Of course it is!
				// And what does that mean? There's a regular expression for
				// gog sake for it. So there's literally *no reason* we can't
				// just describe all of this with a CFG and base the lexer on
				// the regexes of terminals (and nearly terminals)!!!!!!!!
				if len(matches) > 0 {
					if len(matches) > 1 {
						// need to decide which one to do, select the largest one
						longestMatchLen := 0
						for _, m := range matches {
							if len(m.literal) > longestMatchLen {
								longestMatchLen = len(m.literal)
							}
						}

						// drop all smaller than largest
						bettaMatches := []matchRule{}
						for _, m := range matches {
							if len(m.literal) == longestMatchLen {
								bettaMatches = append(bettaMatches, m)
							}
						}

						// if we STILL have an ambiguity, take the one at start
						matches = bettaMatches[:1]
					}
					action := matches[0]

					flushCurrentPendingToken()
					curToken.pos = curLinePos
					curToken.line = curLine
					curToken.class = action.class
					if action.transitionModeTo != lexDefault {
						mode = action.transitionModeTo
						writeRuneSlice(&sb, action.literal)
					} else {
						// for a normal match, just put in the token
						curToken.lexeme = action.lexeme
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = token{}
					}

					if action.class == tsGroupOpen {
						parenDepth++
					} else if action.class == tsGroupClose {
						parenDepth--
						if endAtMatchingParen && parenDepth == 0 {
							runesConsumed = i + len(action.literal) - 1
							break
						}
					}
					i += len(action.literal) - 1
				} else {
					// is this the first non empty char? set the props for an unquoted string,
					// the default.
					if sb.Len() == 0 {
						curToken.line = curLine
						curToken.pos = curLinePos
						curToken.class = tsUnquotedString
					}
					sb.WriteRune(ch)
					escaping = false
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
		return tokenStream{}, 0, SyntaxError{
			message: "unterminated '" + literalStrStringQuote + "'-string; missing a second '" + literalStrStringQuote + "'",
		}
	}

	// do we have leftover unparsed text? add it to the tokens list
	flushCurrentPendingToken()

	// 2nd pass, do in order:
	// * glue together whitespace between unquoted strings, and consecutive
	//   unquoted string sequences
	// * delete whitespace elsewhere
	// * convert unquoted string constants that match num and bool literals to
	//   that token class
	secondPassTokens := make([]token, len(tokens))
	numNew := 0
	for i := 0; i < len(tokens); i++ {
		if tokens[i].class == tsUnquotedString {
			var fullToken = token{
				line:     tokens[i].line,
				pos:      tokens[i].pos,
				fullLine: tokens[i].fullLine,
				class:    tokens[i].class,
			}
			fullUnquoted := strings.Builder{}
			fullUnquoted.WriteString(tokens[i].lexeme)

			added := 0
			for j := 1; i+j < len(tokens); j++ {
				lookahead := tokens[i+j]

				if lookahead.class == tsUnquotedString {
					fullUnquoted.WriteString(lookahead.lexeme)
					added++
				} else if lookahead.class == tsWhitespace && j+i+1 < len(tokens) && tokens[j+i+1].class == tsUnquotedString {
					// need to look ahead AGAIN; only add if the whitespace is
					// between unquoted strings
					fullUnquoted.WriteString(lookahead.lexeme)
					added++
				} else {
					break
				}
			}
			i += added

			fullToken.lexeme = fullUnquoted.String()

			// mark if the whitespace-glued string matches a bool or a num
			// constant
			vUp := strings.ToUpper(fullToken.lexeme)
			for i := range boolConsts {
				if boolConsts[i] == vUp {
					fullToken.class = tsBool
				}
			}
			if patNum.MatchString(vUp) {
				fullToken.class = tsNumber
			}

			secondPassTokens[numNew] = fullToken
			numNew++
		} else if tokens[i].class != tsWhitespace {
			secondPassTokens[numNew] = tokens[i]
			numNew++
		}
	}
	tokens = secondPassTokens[:numNew]

	// add special EOT token
	tokens = append(tokens, token{
		pos:   curLinePos,
		line:  curLine,
		class: tsEndOfText,
	})

	return tokenStream{tokens: tokens}, runesConsumed, nil
}

func readFullLine(sRunes []rune) string {
	var lineBuilder strings.Builder
	for i := 0; i < len(sRunes) && sRunes[i] != '\n'; i++ {
		lineBuilder.WriteRune(sRunes[i])
	}
	return lineBuilder.String()
}

func startMatches(s []rune, check []rune) bool {
	if len(check) > len(s) {
		return false
	}

	for i := range check {
		if s[i] != check[i] {
			return false
		}
	}

	return true
}
