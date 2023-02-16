package lex

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"

	"github.com/dekarrin/tunaq/internal/ictiobus/types"
)

type patAct struct {
	src string
	pat *regexp.Regexp
	act Action
}

type Lexer interface {
	// Lex returns a token stream. The tokens may be lexed in a lazy fashion or
	// an immediate fashion; if it is immediate, errors will be returned at that
	// point. If it is lazy, then error token productions will be returned to
	// the callers of the returned TokenStream at the point where the error
	// occured.
	Lex(input io.Reader) (types.TokenStream, error)
	AddClass(cl types.TokenClass, forState string)
	AddPattern(pat string, action Action, forState string) error
}

type lexerTemplate struct {
	patterns   map[string][]patAct
	StartState string

	// classes by ID by state
	classes map[string]map[string]types.TokenClass
}

func (lx *lexerTemplate) Lex(input io.Reader) (types.TokenStream, error) {
	// okay, we're going to run some operations on our reader that will require
	// knowing exactly what was read by regex, so toss our reader into a
	// TeeReader

	active := &lazyLex{
		rbuf:     &bytes.Buffer{},
		patterns: make(map[string][]patAct),
		classes:  make(map[string]map[string]types.TokenClass),
		state:    lx.StartState,
	}

	// copy anyfin read from the reader to our buffer for later checking after
	// regex match
	teeReader := io.TeeReader(input, active.rbuf)
	active.r = *bufio.NewReader(teeReader)

	// now copy templated values from the parent Lexer.
	for k := range lx.patterns {
		statePats := lx.patterns[k]
		statePatsCopy := make([]patAct, len(statePats))

		for i := range statePats {
			patActCopy := patAct{
				pat: statePats[i].pat,
				act: statePats[i].act,
				src: statePats[i].src,
			}
			statePatsCopy[i] = patActCopy
		}
		active.patterns[k] = statePatsCopy
	}

	for k := range lx.classes {
		stateClasses := lx.classes[k]
		stateClassesCopy := make(map[string]types.TokenClass)

		for j := range stateClasses {
			stateClassesCopy[j] = stateClasses[j]
		}

		active.classes[k] = stateClassesCopy
	}

	return active, nil
}

func NewLexer() Lexer {
	return &lexerTemplate{
		patterns:   map[string][]patAct{},
		StartState: "",
		classes:    map[string]map[string]types.TokenClass{},
	}
}

// AddClass adds the given token class to the lexer. This will mark that token
// class as a lexable token class, and make it available for use in the Action
// of an AddPattern.
//
// If the given token class's ID() returns a string matching one already added,
// the provided one will replace the existing one.
func (lx *lexerTemplate) AddClass(cl types.TokenClass, forState string) {
	stateClasses, ok := lx.classes[forState]
	if !ok {
		stateClasses = map[string]types.TokenClass{}
	}

	stateClasses[cl.ID()] = cl
	lx.classes[forState] = stateClasses
}

func (lx *lexerTemplate) AddPattern(pat string, action Action, forState string) error {
	statePatterns, ok := lx.patterns[forState]
	if !ok {
		statePatterns = make([]patAct, 0)
	}
	stateClasses, ok := lx.classes[forState]
	if !ok {
		stateClasses = map[string]types.TokenClass{}
	}

	compiled, err := regexp.Compile(pat)
	if err != nil {
		return fmt.Errorf("cannot compile regex: %w", err)
	}

	if action.Type == ActionScan || action.Type == ActionScanAndState {
		// check class exists
		id := action.ClassID
		_, ok := stateClasses[id]
		if !ok {
			return fmt.Errorf("%q is not a defined token class on this lexer; add it with AddClass first", id)
		}
	}
	if action.Type == ActionState || action.Type == ActionScanAndState {
		if action.State == "" {
			return fmt.Errorf("action includes state shift but does not define state to shift to (cannot shift to empty state)")
		}
	}

	record := patAct{
		src: pat,
		pat: compiled,
		act: action,
	}
	statePatterns = append(statePatterns, record)

	lx.patterns[forState] = statePatterns
	// not modifying lx.classes so no need to set it again
	return nil
}

type lazyLex struct {
	r    bufio.Reader
	rbuf *bytes.Buffer

	curLine     int
	curPos      int
	curFullLine string
	done        bool
	patterns    map[string][]patAct
	state       string
	classes     map[string]map[string]types.TokenClass
}

// Next returns the next token in the stream and advances the stream by one
// token. If at the end of the stream, this will return a token whose Class()
// is types.TokenEndOfText. If an error in lexing occurs, it will return a token
// whose Class() is types.TokenError and whose lexeme is a message explaining
// the error.
func (lx *lazyLex) Next() types.Token {
	if lx.done {
		return lexerToken{
			class:   types.TokenEndOfText,
			line:    lx.curFullLine,
			linePos: lx.curPos,
			lineNum: lx.curLine,
		}
	}

	statePatterns := lx.patterns[lx.state]
	stateClasses := lx.classes[lx.state]
	matchingRegexes := make([]patAct, len(statePatterns))
	copy(matchingRegexes, statePatterns)

	// this could probably be optimized as somefin other than a rune-by-rune
	// scan, altho the fact that its buffered means at least we don't have to
	// worry about ensuring we load all bytes of a variable-length UTF-8 code
	// unit.
	var unconsumedBuffer []rune
	for {
		nextRune, _, err := lx.r.ReadRune()
		if err == io.EOF {
			if len(unconsumedBuffer) > 0 {
				return lexerToken{
					class:   types.TokenError,
					line:    lx.curFullLine,
					linePos: lx.curPos,
					lineNum: lx.curLine,
					lexed:   "unexpected end of input",
				}
			}
			// otherwise, this is not a problem at all; nothing remains
			// unconsumed, we are simply at end of input. If there is a problem
			// with input, it will be syntactic, and up to the parser to
			// determine.
			lx.done = true
			return lexerToken{
				class:   types.TokenEndOfText,
				line:    lx.curFullLine,
				linePos: lx.curPos,
				lineNum: lx.curLine,
			}
		} else if err != nil {
			// if it fails for any reason we need to stop lexing because our
			// underlying input source is no longer working.
			return lexerToken{
				class:   types.TokenError,
				line:    lx.curFullLine,
				linePos: lx.curPos,
				lineNum: lx.curLine,
				lexed:   err.Error(),
			}
		}

		// error handling is complete; add the scanned rune to our buf
		unconsumedBuffer = append(unconsumedBuffer)

		// and now, run all of our regex on it based on state.
	}

	return lexerToken{}
}

// Peek returns the next token in the stream without advancing the stream.
func (lx *lazyLex) Peek() types.Token {
	return lexerToken{}
}

// HasNext returns whether the stream has any additional tokens.
func (lx *lazyLex) HasNext() bool {
	return false
}
