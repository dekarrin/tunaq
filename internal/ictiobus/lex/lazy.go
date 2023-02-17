package lex

import (
	"fmt"
	"io"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/dekarrin/tunaq/internal/ictiobus/types"
)

type lazyLex struct {
	// buffered reader that can run regex and retrieve results
	r *regexReader

	// cur state
	state string

	// track these for placement in tokens, for later error reporting
	curLine     int
	curPos      int
	curFullLine string

	// set to true when the lazyLex has reached end of input, causing all
	// subsequent calls to Next() to return a Token with class
	// types.TokenEndOfText and all subsequent calls to HasNext() to return
	// false.
	done bool

	// panic mode is entered when no lexeme is found; the next call to Next()
	// will begin discarding characters until a valid one is found
	panicMode bool

	// classes mapping
	classes map[string]map[string]types.TokenClass

	// split actions from regexes to match indexes to capturing groups
	actions map[string][]Action

	// one regex per state. each regex will be constructed by taking all regex
	// for a state and placing them in capturing groups separated by alternation
	// operators.
	patterns map[string]*regexp.Regexp
}

func (lx *lexerTemplate) LazyLex(input io.Reader) (types.TokenStream, error) {
	// okay, we're going to run some operations on our reader that will require
	// knowing exactly what was read by regex, so toss our reader into a
	// TeeReader

	active := &lazyLex{
		r:        NewRegexReader(input),
		patterns: make(map[string]*regexp.Regexp),
		classes:  make(map[string]map[string]types.TokenClass),
		actions:  make(map[string][]Action),
		state:    lx.StartingState(),
	}

	// move all patterns into "super pattern"; one per state. and separate the
	// actions into their own data structure
	for k := range lx.patterns {
		statePats := lx.patterns[k]
		var superRegex strings.Builder
		superRegex.WriteString("^(?:")
		lazyActs := make([]Action, len(statePats))

		for i := range statePats {
			act := statePats[i].act
			src := statePats[i].src
			superRegex.WriteString("(" + src + ")")
			if i+1 < len(statePats) {
				superRegex.WriteRune('|')
			}
			lazyActs[i] = act
		}

		superRegex.WriteRune(')')

		compiled, err := regexp.Compile(superRegex.String())
		if err != nil {
			// should never happen
			return nil, fmt.Errorf("composing token regexes: %w", err)
		}

		active.patterns[k] = compiled
		active.actions[k] = lazyActs
	}

	// move over classes too (although they might not be needed)
	for k := range lx.classes {
		stateClasses := lx.classes[k]
		stateClassesCopy := make(map[string]types.TokenClass)

		for j := range stateClasses {
			stateClassesCopy[j] = stateClasses[j]
		}

		active.classes[k] = stateClassesCopy
	}

	// set current line and pos
	active.curLine = 1
	active.curPos = 1

	return active, nil
}

// Next returns the next token in the stream and advances the stream by one
// token. If at the end of the stream, this will return a token whose Class()
// is types.TokenEndOfText. If an error in lexing occurs, it will return a token
// whose Class() is types.TokenError and whose lexeme is a message explaining
// the error.
func (lx *lazyLex) Next() types.Token {
	if lx.done {
		return lx.makeEOTToken()
	}

	pat := lx.patterns[lx.state]
	stateActions := lx.actions[lx.state]
	stateClasses := lx.classes[lx.state]

	var matches []string
	var readError error
	for {
		// retrieve the current matches, discarding runes until we find a match
		// if in panic mode.

		if lx.panicMode {
			for lx.panicMode {
				// track the rune we are dropping to add to source text context
				// tracking
				var ch rune
				ch, _, readError = lx.r.ReadRune()

				if readError != nil {
					return lx.tokenForIOError(readError)
				}

				if ch == '\n' {
					lx.curLine++
					lx.curPos = 0
					lx.curFullLine = ""
				}
				lx.curPos++
				lx.curFullLine += string(ch)

				matches, readError = lx.r.SearchAndAdvance(pat)
				if readError != nil {
					return lx.tokenForIOError(readError)
				}

				if len(matches) > 0 {
					// we found something. exit panic mode and continue
					lx.panicMode = false
				}
			}
		} else {
			matches, readError = lx.r.SearchAndAdvance(pat)
			if readError != nil {
				return lx.tokenForIOError(readError)
			}

			if len(matches) < 1 {
				// no match at start of reader. return an error token and enter
				// panic mode
				lx.panicMode = true
				return lx.makeErrorTokenf("unknown input")
			}
		}

		actionIdx, lexeme := lx.selectMatch(matches)

		// update source text context tracking
		for _, ch := range lexeme {
			if ch == '\n' {
				lx.curLine++
				lx.curPos = 0
				lx.curFullLine = ""
			}
			lx.curPos++

			// TODO: this cannot be efficient, there is shorely a betta way
			lx.curFullLine += string(ch)
		}

		action := stateActions[actionIdx]

		switch action.Type {
		case ActionNone:
			// discard the lexeme (do nothing), then keep lexing
		case ActionScan:
			// return the token
			class := stateClasses[action.ClassID]
			tok := lx.makeToken(class, lexeme)

			return tok
		case ActionState:
			// modify state, then keep lexing
			newState := action.State
			lx.state = newState
		case ActionScanAndState:
			// modify state, then return the token

			// doing token creation first in case a state shift alters what is
			// in the token
			class := stateClasses[action.ClassID]
			tok := lx.makeToken(class, lexeme)

			newState := action.State
			lx.state = newState

			return tok
		}
	}
}

// Peek returns the next token in the stream without advancing the stream.
func (lx *lazyLex) Peek() types.Token {
	// preserve all parts of the lexer that might change during a call to Next()
	// so we can restore it afterward
	lx.r.Mark("peek")
	oldState := lx.state
	oldFullLine := lx.curFullLine
	oldLine := lx.curLine
	oldPos := lx.curPos
	oldDone := lx.done
	oldPanic := lx.panicMode

	// run lexing as normal:
	tok := lx.Next()

	// restore original data
	lx.r.Restore("peek")
	lx.state = oldState
	lx.curFullLine = oldFullLine
	lx.curLine = oldLine
	lx.curPos = oldPos
	lx.done = oldDone
	lx.panicMode = oldPanic

	// and finally, return the token
	return tok
}

// HasNext returns whether the stream has any additional tokens.
func (lx *lazyLex) HasNext() bool {
	return !lx.done
}

func (lx *lazyLex) makeToken(class types.TokenClass, lexeme string) types.Token {
	return lexerToken{
		class:   class,
		line:    lx.curFullLine,
		linePos: lx.curPos,
		lineNum: lx.curLine,
		lexed:   lexeme,
	}
}

func (lx *lazyLex) makeEOTToken() types.Token {
	return lx.makeToken(types.TokenEndOfText, "")
}

func (lx *lazyLex) makeErrorTokenf(formatMsg string, args ...any) types.Token {
	msg := fmt.Sprintf(formatMsg, args...)
	return lx.makeToken(types.TokenError, msg)
}

// token for read error takes the given error returned from an I/O operation,
// sets state on lx based on whether the error is io.EOF or some other error,
// then returns a token appropriate for the error, either one of class
// types.TokenEndOfText for io.EOF or types.TokenError for all other errors.
func (lx *lazyLex) tokenForIOError(err error) types.Token {
	lx.done = true

	if err == io.EOF {
		lx.panicMode = false
		return lx.makeEOTToken()
	}
	return lx.makeErrorTokenf("I/O error: %s", err.Error())
}

// select match from slice of all regex matches. If there is exactly 1 match,
// return that. assumes that the first element of candidates is a 'full match'
// and therefore useless, and that blank entries in subsequent indexes indicates
// non-match.
//
// Returns the index of the action associated with the match, and the match
// itself.
func (lx *lazyLex) selectMatch(candidates []string) (int, string) {
	// we now have our list of matches. which sub-expression(s) matched?
	// (and consider a blank match to be 'no match' at this time)
	// TODO: distinguish between blank match and no match in regexReader.

	// toss them all into a 'sparse array' at their index-1 so they have
	// direct correspondance to the index of the action they imply.
	subExprMatches := map[int]string{}
	for i := 1; i < len(candidates); i++ {
		if candidates[i] != "" {
			subExprMatches[i-1] = candidates[i]
		}
	}

	// do we have a conflict between two lexemes? if so, do gnu lex style
	// resolution: prefer the longer one, and if all are equal, prefer the
	// one first defined.
	if len(subExprMatches) > 1 {
		// find the longest length
		var longest int
		for i := range subExprMatches {
			m := subExprMatches[i]
			runeCount := utf8.RuneCountInString(m)
			if runeCount > longest {
				longest = runeCount
			}
		}

		// eliminate all but the longest length one(s)
		keep := map[int]string{}
		for i := range subExprMatches {
			m := subExprMatches[i]
			runeCount := utf8.RuneCountInString(m)
			if runeCount == longest {
				keep[i] = m
			}
		}
		subExprMatches = keep

		// do we still have multiple matches? if so, take the first one
		// defined (with the lowest index)
		if len(subExprMatches) > 1 {

			// need to scan for lowest index because iteration order is not
			// guaranteed
			lowestIndex := math.MaxInt
			for i := range subExprMatches {
				if i < lowestIndex {
					lowestIndex = i
				}
			}

			// just grab that one and put it into a new map
			keep := map[int]string{
				lowestIndex: subExprMatches[lowestIndex],
			}
			subExprMatches = keep
		}
	}

	// we now have exactly one candidate match in our map, so iteration will
	// give us this value

	var matchIndex int
	var matchText string
	for i := range subExprMatches {
		matchIndex = i
		matchText = subExprMatches[i]
		break
	}

	return matchIndex, matchText
}
