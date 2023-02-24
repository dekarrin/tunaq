package lex

import (
	"fmt"
	"io"
	"regexp"

	"github.com/dekarrin/tunaq/internal/ictiobus/types"
)

// TODO: src is useless as its in pat.
type patAct struct {
	src string
	pat *regexp.Regexp
	act Action
}

type lexerTemplate struct {
	lazy bool

	patterns   map[string][]patAct
	startState string

	// classes by ID by state
	classes map[string]map[string]types.TokenClass

	listener func(types.Token)
}

func NewLexer(lazy bool) *lexerTemplate {
	return &lexerTemplate{
		lazy:       lazy,
		patterns:   map[string][]patAct{},
		startState: "",
		classes:    map[string]map[string]types.TokenClass{},
	}
}

func (lx *lexerTemplate) Lex(input io.Reader) (types.TokenStream, error) {
	if lx.lazy {
		return lx.LazyLex(input)
	} else {
		return lx.ImmediatelyLex(input)
	}
}

func (lx *lexerTemplate) SetStartingState(s string) {
	lx.startState = s
}

func (lx *lexerTemplate) StartingState() string {
	return lx.startState
}

func (lx *lexerTemplate) RegisterTokenListener(fn func(t types.Token)) {
	lx.listener = fn
}

// AddClass adds the given token class to the lexer. This will mark that token
// class as a lexable token class, and make it available for use in the Action
// of an AddPattern.
//
// If the given token class's ID() returns a string matching one already added,
// the provided one will replace the existing one.
func (lx *lexerTemplate) RegisterClass(cl types.TokenClass, forState string) {
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

var eolRegex = regexp.MustCompile(`([^\n]*)(?:\n|$)`)

// scans through the reader to find the remainder of the current line and
// returns it
func readLineWithoutAdvancing(r *regexReader) string {
	r.Mark("line")
	matches, err := r.SearchAndAdvance(eolRegex)
	if err != nil {
		panic(fmt.Sprintf("trying to get rest of line: %s", err))
	}
	if len(matches) < 2 {
		panic("rest of line did not have subexpression")
	}
	line := matches[1]

	r.Restore("line")

	return line
}
