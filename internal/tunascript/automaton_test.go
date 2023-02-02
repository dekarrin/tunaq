package tunascript

import (
	"testing"

	"github.com/dekarrin/tunaq/internal/util"
	"github.com/stretchr/testify/assert"
)

func Test_NewViablePrefixNFA(t *testing.T) {
	testCases := []struct {
		name        string
		grammar     string
		expect      map[string][]string
		expectStart string
	}{
		{
			name: "aiken example",
			grammar: `
				E -> T + E | T ;
				T -> int * T | int | ( E ) ;
			`,
			expect: map[string][]string{
				// first row from vid
				"T -> . ( E )": {
					"=(()=> T -> ( . E )",
				},
				"T -> ( . E )": {
					"=(ε)=> E -> . T",
					"=(ε)=> E -> . T + E",
					"=(E)=> T -> ( E . )",
				},
				"T -> ( E . )": {
					"=())=> T -> ( E ) .",
				},
				"T -> ( E ) .": {},

				// 2nd row from vid
				"E-P -> E .": {},
				"E -> . T + E": {
					"=(ε)=> T -> . ( E )",
					"=(T)=> E -> T . + E",
					"=(ε)=> T -> . int",
					"=(ε)=> T -> . int * T",
				},
				"E -> T . + E": {
					"=(+)=> E -> T + . E",
				},
				"E -> T + . E": {
					"=(ε)=> E -> . T + E",
					"=(E)=> E -> T + E .",
					"=(ε)=> E -> . T",
				},

				// 3rd row from vid
				"E-P -> . E": {
					"=(E)=> E-P -> E .",
					"=(ε)=> E -> . T + E",
					"=(ε)=> E -> . T",
				},
				"T -> . int": {
					"=(int)=> T -> int .",
				},
				"T -> int .":   {},
				"E -> T + E .": {},

				// 4th row from vid
				"E -> . T": {
					"=(ε)=> T -> . int",
					"=(ε)=> T -> . int * T",
					"=(T)=> E -> T .",
					"=(ε)=> T -> . ( E )",
				},
				"T -> int . * T": {
					"=(*)=> T -> int * . T",
				},

				// 5th row from vid
				"E -> T .": {},
				"T -> . int * T": {
					"=(int)=> T -> int . * T",
				},
				"T -> int * . T": {
					"=(ε)=> T -> . int",
					"=(T)=> T -> int * T .",
					"=(ε)=> T -> . ( E )",
					"=(ε)=> T -> . int * T",
				},
				"T -> int * T .": {},
			},
			expectStart: "E-P -> . E",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// setup
			assert := assert.New(t)
			g := mustParseGrammar(tc.grammar)
			expect := buildLR0NFA(tc.expect, tc.expectStart)

			// execute
			actual := NewViablePrefixNDA(g)

			// assert
			assert.Equal(expect.String(), actual.String())
		})
	}
}

func Test_NFA_EpsilonClosure(t *testing.T) {
	testCases := []struct {
		name      string
		nfa       map[string][]string
		nfaStart  string
		nfaAccept []string
		forState  string
		expect    []string
	}{
		{
			name: "aiken example - B",
			nfa: map[string][]string{
				"A": {
					"=(ε)=> H",
					"=(ε)=> B",
				},
				"B": {
					"=(ε)=> C",
					"=(ε)=> D",
				},
				"C": {
					"=(1)=> E",
				},
				"D": {
					"=(0)=> F",
				},
				"E": {
					"=(ε)=> G",
				},
				"F": {
					"=(ε)=> G",
				},
				"G": {
					"=(ε)=> A",
					"=(ε)=> H",
				},
				"H": {
					"=(ε)=> I",
				},
				"I": {
					"=(1)=> J",
				},
				"J": {},
			},
			nfaAccept: []string{"J"},
			nfaStart:  "A",
			forState:  "B",
			expect:    []string{"B", "C", "D"},
		},
		{
			name: "aiken example - G",
			nfa: map[string][]string{
				"A": {
					"=(ε)=> H",
					"=(ε)=> B",
				},
				"B": {
					"=(ε)=> C",
					"=(ε)=> D",
				},
				"C": {
					"=(1)=> E",
				},
				"D": {
					"=(0)=> F",
				},
				"E": {
					"=(ε)=> G",
				},
				"F": {
					"=(ε)=> G",
				},
				"G": {
					"=(ε)=> A",
					"=(ε)=> H",
				},
				"H": {
					"=(ε)=> I",
				},
				"I": {
					"=(1)=> J",
				},
				"J": {},
			},
			nfaAccept: []string{"J"},
			nfaStart:  "A",
			forState:  "G",
			expect:    []string{"A", "B", "C", "D", "G", "H", "I"},
		},
		{
			name: "aiken example, recursive variant - G",
			nfa: map[string][]string{
				"A": {
					"=(ε)=> H",
					"=(ε)=> B",
				},
				"B": {
					"=(ε)=> C",
					"=(ε)=> D",
				},
				"C": {
					"=(ε)=> E",
				},
				"D": {
					"=(0)=> F",
				},
				"E": {
					"=(ε)=> G",
				},
				"F": {
					"=(ε)=> G",
				},
				"G": {
					"=(ε)=> A",
					"=(ε)=> H",
				},
				"H": {
					"=(ε)=> I",
				},
				"I": {
					"=(1)=> J",
				},
				"J": {},
			},
			nfaAccept: []string{"J"},
			nfaStart:  "A",
			forState:  "G",
			expect:    []string{"A", "B", "C", "D", "G", "H", "I", "E"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// setup
			assert := assert.New(t)
			nfa := buildNFA(tc.nfa, tc.nfaStart, tc.nfaAccept)
			expectSet := util.SetFromSlice(tc.expect)

			// execute
			actual := nfa.EpsilonClosure(tc.forState)

			// assert
			assert.True(actual.Equal(expectSet))
		})
	}
}

type strWrap string

func (s strWrap) String() string {
	return string(s)
}

func buildNFA(from map[string][]string, start string, acceptingStates []string) *NFA[strWrap] {
	nfa := &NFA[strWrap]{}

	acceptSet := util.SetFromSlice(acceptingStates)

	for k := range from {
		stateItem := strWrap(k)
		nfa.AddState(stateItem, acceptSet.Has(k))
	}

	// add transitions AFTER all states are already in or it will cause a panic
	for k := range from {
		fromItem := strWrap(k)
		for i := range from[k] {
			transition := mustParseFATransition(from[k][i])
			input := transition.input
			toItem := strWrap(transition.next)
			nfa.AddTransition(fromItem, input, toItem)
		}
	}

	nfa.start = start

	return nfa
}

func buildLR0NFA(from map[string][]string, start string) *NFA[LR0Item] {
	nfa := &NFA[LR0Item]{}

	for k := range from {
		stateItem := mustParseLR0Item(k)
		nfa.AddState(stateItem, true)
	}

	fromKeys := util.OrderedKeys(from)

	for _, k := range fromKeys {
		fromItem := mustParseLR0Item(k)
		for i := range from[k] {
			transition := mustParseFATransition(from[k][i])
			toItem := mustParseLR0Item(transition.next)
			input := transition.input
			nfa.AddTransition(fromItem, input, toItem)
		}
	}

	nfa.start = start

	return nfa
}
