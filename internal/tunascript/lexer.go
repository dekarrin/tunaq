package tunascript

import (
	"strings"
	"unicode"
)

const (
	// if newline is put in any of these it will break the lexer detection of
	// position so don't do that

	literalStrStringQuote     = "@"
	literalStrSeparator       = ","
	literalStrGroupOpen       = "("
	literalStrGroupClose      = ")"
	literalStrIdentifierStart = "$"
	literalStrOpIs            = "=="
	literalStrOpIsNot         = "!="
	literalStrOpLessThan      = "<"
	literalStrOpGreaterThan   = ">"
	literalStrOpLessThanIs    = "<="
	literalStrOpGreaterThanIs = ">="
	literalStrOpSet           = "="
	literalStrOpPlus          = "+"
	literalStrOpMinus         = "-"
	literalStrOpMultiply      = "*"
	literalStrOpDivide        = "/"
	literalStrOpIncset        = "+="
	literalStrOpDecset        = "-="
	literalStrOpInc           = "++"
	literalStrOpDec           = "--"
	literalStrOpAnd           = "&&"
	literalStrOpOr            = "||"
	literalStrOpNot           = "!"
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
	tsSeparator       = tokenClass{"TS_SEPARATOR", "'" + literalStrSeparator + "'", 0}
	tsGroupOpen       = tokenClass{"TS_GROUP_OPEN", "'" + literalStrGroupOpen + "'", 100}
	tsGroupClose      = tokenClass{"TS_GROUP_CLOSE", "'" + literalStrGroupClose + "'", 0}
	tsIdentifier      = tokenClass{"TS_IDENTIFIER", "identifier", 0}
	tsEndOfText       = tokenClass{"TS_END_OF_TEXT", "end of text", 0}
	tsUndefined       = tokenClass{"TS_UNDEFINED", "undefined", 0}
	tsNumber          = tokenClass{"TS_NUMBER", "number", 0}
	tsBool            = tokenClass{"TS_BOOL", "boolean value", 0}
	tsUnquotedString  = tokenClass{"TS_UNQUOTED_STRING", "text value", 0}
	tsQuotedString    = tokenClass{"TS_QUOTED_STRING", literalStrStringQuote + "-text value", 0}
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
	tsOpAnd           = tokenClass{"TS_OP_AND", "'" + literalStrOpAnd + "'", 0}
	tsOpOr            = tokenClass{"TS_OP_OR", "'" + literalStrOpOr + "'", 0}
	tsOpNot           = tokenClass{"TS_OP_NOT", "'" + literalStrOpNot + "'", 0}
)

func LexOperationText(s string) (tokenStream, error) {
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

			// is the cur token literally one of the bool values?
			vUp := strings.ToUpper(curToken.lexeme)
			if patBool.MatchString(vUp) {
				curToken.class = tsBool
			}
			if patNum.MatchString(vUp) {
				curToken.class = tsNumber
			}

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
				writeRuneSlice(sb, literalStringQuote)
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
			if !escaping && ch == '\\' {
				escaping = true
				// TODO: continue but dont actually use that!
				// I Believe It Would Cause It To Go Out Of Scope Yes.
			}

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
					writeRuneSlice(sb, action.literal)
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

				// do not include whitespace unless it is escaped
				if escaping || !unicode.IsSpace(ch) {
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
