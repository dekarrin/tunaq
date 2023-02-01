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
			expect := buildNFA(tc.expect, tc.expectStart)

			// execute
			actual := NewViablePrefixNDA(g)

			// assert
			assert.Equal(expect.String(), actual.String())
		})
	}
}

func buildNFA(from map[string][]string, start string) *NFA[LR0Item] {
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
