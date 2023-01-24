package tunascript

import (
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
						{tsNumber.id},
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
				g.AddTerm(term.id, term)
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
	}{
		{
			name: "empty grammar",
		},
		{
			name: "single rule grammar, no epsilons",
			terminals: []tokenClass{
				tsNumber,
			},
			rules: []Rule{},
		},
		{
			name: "deeba kannan's epsilon elimination example (TOC Lec 25)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			// set up the grammar
			g := Grammar{}
			for _, term := range tc.terminals {
				g.AddTerm(term.id, term)
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
			assert.Len(actual.rules, len(tc.rules))

			// priority DOES matter so we check them in order
			for i := range tc.rules {
				assert.Truef(tc.rules[i].Equal(actual.rules[i]), "expected rules[%d] to be %q but was %q", i, tc.rules[i].String(), actual.rules[i].String())
			}
		})
	}
}
