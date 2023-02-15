package lex

import (
	"fmt"
	"regexp"

	"github.com/dekarrin/tunaq/internal/ictiobus/types"
)

type patAct struct {
	src string
	pat *regexp.Regexp
	act Action
}

type Lexer struct {
	patterns map[string][]patAct
	state    string

	// classes by ID by state
	classes map[string]map[string]types.TokenClass
}

func NewLexer() types.TokenStream {
	return &Lexer{
		patterns:         make(map[string][]string, 0),
		compiledPatterns: make(map[string]*regexp.Regexp, 0),
		state:            "",
	}
}

func (lx *Lexer) AddPattern(pat string, action Action, forState string) error {
	// make shore it can be compiled
	_, err := regexp.Compile(pat)
	if err != nil {
		return fmt.Errorf("cannot compile regex: %w", err)
	}

	statePatterns, ok := lx.patterns[forState]
	if !ok {
		statePatterns = make([]string, 0)
	}

	someRx := regexp.MustCompile("e")

	lx.patterns[forState] = statePatterns
	return nil
}

// Next returns the next token in the stream and advances the stream by one
// token.
func (lx *Lexer) Next() types.Token {
	return lexerToken{}
}

// Peek returns the next token in the stream without advancing the stream.
func (lx *Lexer) Peek() types.Token {
	return lexerToken{}
}

// HasNext returns whether the stream has any additional tokens.
func (lx *Lexer) HasNext() bool {
	return false
}
