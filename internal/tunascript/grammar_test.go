package tunascript

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Grammar_Validate(t *testing.T) {
	testCases := []struct {
		name      string
		rules     []Rule
		terminals []tokenClass
		expectErr bool
	}{
		{
			name:      "empty grammar",
			expectErr: true,
		},
		{
			name: "no rules in grammar",
			terminals: []tokenClass{
				tsNumber,
			},
			expectErr: true,
		},
		{
			name: "no terms in grammar",
			rules: []Rule{{
				NonTerminal: "S",
				Productions: []Production{
					{"S"},
				},
			}},
			expectErr: true,
		},
		{
			name: "single rule grammar",
			rules: []Rule{
				{
					NonTerminal: "S",
					Productions: []Production{
						{strings.ToLower(tsNumber.id)},
					},
				},
			},
			terminals: []tokenClass{
				tsNumber,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			// set up the grammar
			g := Grammar{}
			for _, term := range tc.terminals {
				g.AddTerm(strings.ToLower(term.id), term)
			}
			for _, r := range tc.rules {
				for _, alts := range r.Productions {
					g.AddRule(r.NonTerminal, alts)
				}
			}

			// checkActual
			actual := g.Validate()

			if tc.expectErr {
				assert.Error(actual)
			} else {
				assert.NoError(actual)
			}
		})
	}
}

func Test_Grammar_RemoveEpsilons(t *testing.T) {
	testCases := []struct {
		name      string
		rules     []Rule
		terminals []tokenClass
		expect    []Rule
	}{ /*
			{
				name: "empty grammar",
			},
			{
				name: "single rule grammar, no epsilons",
				terminals: []tokenClass{
					tsNumber,
				},
				rules: []Rule{
					{
						NonTerminal: "S",
						Productions: []Production{
							{"A"},
						},
					},
				},
				expect: []Rule{
					{
						NonTerminal: "S",
						Productions: []Production{
							{"A"},
						},
					},
				},
			},*/
		{
			name: "deeba kannan's epsilon elimination example (TOC Lec 25)",
			terminals: []tokenClass{
				{id: "a", human: "A"},
				{id: "b", human: "B"},
			},
			rules: []Rule{
				{NonTerminal: "S", Productions: []Production{{"A", "C", "A"}, {"A", "a"}}},
				{NonTerminal: "A", Productions: []Production{{"B", "B"}, {""}}},
				{NonTerminal: "B", Productions: []Production{{"A"}, {"b", "C"}}},
				{NonTerminal: "C", Productions: []Production{{"b"}}},
			},
			expect: []Rule{
				{NonTerminal: "S", Productions: []Production{{"A", "C", "A"}, {"C", "A"}, {"A", "C"}, {"C"}, {"A", "a"}, {"a"}}},
				{NonTerminal: "A", Productions: []Production{{"B", "B"}, {"B"}}},
				{NonTerminal: "B", Productions: []Production{{"A"}, {"b", "C"}}},
				{NonTerminal: "C", Productions: []Production{{"b"}}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			// set up the grammar
			g := Grammar{}
			for _, term := range tc.terminals {
				g.AddTerm(strings.ToLower(term.id), term)
			}
			for _, r := range tc.rules {
				for _, alts := range r.Productions {
					g.AddRule(r.NonTerminal, alts)
				}
			}

			actual := g.RemoveEpsilons()

			// terminals must remain unchanged
			assert.Equal(g.terminals, actual.terminals)

			// rules must be as expected, cant do equal bc we need custom
			// comparison logic for each
			assert.Len(actual.rules, len(tc.expect))

			// priority DOES matter so we check them in order
			for i := range tc.expect {
				exp := tc.expect[i]
				act := actual.rules[i]

				assert.Truef(exp.Equal(act), "expected rules[%d] to be %q but was %q", i, exp.String(), act.String())
			}
		})
	}
}
