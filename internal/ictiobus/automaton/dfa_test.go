package automaton

import (
	"testing"

	"github.com/dekarrin/tunaq/internal/ictiobus/grammar"
	"github.com/dekarrin/tunaq/internal/util"
	"github.com/stretchr/testify/assert"
)

func Test_NewLALR1ViablePrefixDFA(t *testing.T) {
	testCases := []struct {
		name        string
		grammar     string
		expect      string
		expectStart string
	}{
		{
			name: "2-rule ex from https://www.cs.york.ac.uk/fp/lsa/lectures/lalr.pdf",
			grammar: `
				S -> C C ;
				C -> c C | d ;
			`,
			expect: `<START: "{C -> . c C, c, C -> . c C, d, C -> . d, c, C -> . d, d, S -> . C C, $, S-P -> . S, $}", STATES:
	(({C -> . c C, $, C -> . c C, c, C -> . c C, d, C -> . d, $, C -> . d, c, C -> . d, d, C -> c . C, $, C -> c . C, c, C -> c . C, d} [=(C)=> {C -> c C ., $, C -> c C ., c, C -> c C ., d}, =(c)=> {C -> . c C, $, C -> . c C, c, C -> . c C, d, C -> . d, $, C -> . d, c, C -> . d, d, C -> c . C, $, C -> c . C, c, C -> c . C, d}, =(d)=> {C -> d ., $, C -> d ., c, C -> d ., d}])),
	(({C -> . c C, $, C -> . d, $, S -> C . C, $} [=(C)=> {S -> C C ., $}, =(c)=> {C -> . c C, $, C -> . c C, c, C -> . c C, d, C -> . d, $, C -> . d, c, C -> . d, d, C -> c . C, $, C -> c . C, c, C -> c . C, d}, =(d)=> {C -> d ., $, C -> d ., c, C -> d ., d}])),
	(({C -> . c C, c, C -> . c C, d, C -> . d, c, C -> . d, d, S -> . C C, $, S-P -> . S, $} [=(C)=> {C -> . c C, $, C -> . d, $, S -> C . C, $}, =(S)=> {S-P -> S ., $}, =(c)=> {C -> . c C, $, C -> . c C, c, C -> . c C, d, C -> . d, $, C -> . d, c, C -> . d, d, C -> c . C, $, C -> c . C, c, C -> c . C, d}, =(d)=> {C -> d ., $, C -> d ., c, C -> d ., d}])),
	(({C -> c C ., $, C -> c C ., c, C -> c C ., d} [])),
	(({C -> d ., $, C -> d ., c, C -> d ., d} [])),
	(({S -> C C ., $} [])),
	(({S-P -> S ., $} []))
>`,
		}, /*
			{
				name: "purple dragon 'efficient' LALR construction grammar",
				grammar: `
					S -> L = R | R ;
					L -> * R | id ;
					R -> L ;
				`,
			},*/
	}

	// TODO: FILL WITH PROPER INFO, IT DOES WORK (IN THEORY)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// setup
			assert := assert.New(t)
			g := grammar.MustParse(tc.grammar)

			// execute
			actual, err := NewLALR1ViablePrefixDFA(g)
			if !assert.NoError(err) {
				return
			}

			// assert
			assert.Equal(tc.expect, actual.String())
		})
	}

}

func buildDFA(from map[string][]string, start string, acceptingStates []string) *DFA[string] {
	dfa := &DFA[string]{}

	acceptSet := util.StringSetOf(acceptingStates)

	for k := range from {
		dfa.AddState(k, acceptSet.Has(k))
		dfa.SetValue(k, k)
	}

	// add transitions AFTER all states are already in or it will cause a panic
	for k := range from {
		for i := range from[k] {
			transition := mustParseFATransition(from[k][i])
			dfa.AddTransition(k, transition.input, transition.next)
		}
	}

	dfa.Start = start

	return dfa
}
