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
	}{
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
		},
		{
			name: "deeba kannan's epsilon elimination example (TOC Lec 25)",
			terminals: []tokenClass{
				{id: "a", human: "A"},
				{id: "b", human: "B"},
			},
			rules: []Rule{
				mustParseRule("S -> A C A | A a"),
				mustParseRule("A -> B B | ε"),
				mustParseRule("B -> A | b C"),
				mustParseRule("C -> b"),
			},
			expect: []Rule{
				mustParseRule("S -> A C A | C A | A C | C | A a | a"),
				mustParseRule("A -> B B | B"),
				mustParseRule("B -> A | b C"),
				mustParseRule("C -> b"),
			},
		},
		{
			name: "purple dragon book ex. 4.4.6",
			terminals: []tokenClass{
				{id: "a", human: "a"},
				{id: "b", human: "b"},
			},
			rules: []Rule{
				mustParseRule(`
					S -> a S b S
					   | b S a S
					   | ε
				`),
			},
			expect: []Rule{
				mustParseRule(`
					S -> a S b S
					   | a b S
					   | a S b
					   | a b
					   | b S a S
					   | b a S
					   | b S a
					   | b a
				`),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			// set up
			g := Grammar{}
			for _, term := range tc.terminals {
				g.AddTerm(strings.ToLower(term.id), term)
			}
			for _, r := range tc.rules {
				for _, alts := range r.Productions {
					g.AddRule(r.NonTerminal, alts)
				}
			}

			// execute
			actual := g.RemoveEpsilons()

			// assert

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

func Test_Grammar_RemoveUnitProductions(t *testing.T) {
	// TODO: make all tests have this input form its super convenient
	testCases := []struct {
		name      string
		terminals []string
		rules     []string
		expect    []string
	}{
		{
			name: "empty grammar",
		},
		{
			name:      "single rule grammar, no unit prods",
			terminals: []string{"a", "b"},
			rules: []string{
				"S -> a | b",
			},
			expect: []string{
				"S -> a | b",
			},
		},
		{
			name:      "grammar with one unit prod",
			terminals: []string{"a", "b"},
			rules: []string{
				"S -> A | b",
				"A -> a",
			},
			expect: []string{
				"S -> a | b",
			},
		},
		{
			name:      "parinita hajra's example 1",
			terminals: []string{"n", "q"},
			rules: []string{
				"S -> N | Q N n q Q",
				"N -> n q N | n",
				"Q -> q Q | ε",
			},
			expect: []string{
				"S -> n q N | n | Q N n q Q",
				"N -> n q N | n",
				"Q -> q Q | ε",
			},
		},
		{
			name:      "parinita hajra's example 2",
			terminals: []string{"a", "b", "c"},
			rules: []string{
				"S -> A a | B | c",
				"B -> A | b b",
				"A -> a | b c | B",
			},
			expect: []string{
				"S -> A a | a | b c | b b | c",
				"A -> a | b c | b b",
			},
		},
		{
			name:      "neso academy example",
			terminals: []string{"a", "b"},
			rules: []string{
				"S -> X Y",
				"X -> a",
				"Y -> Z | b",
				"Z -> M",
				"M -> N",
				"N -> a",
			},
			expect: []string{
				"S -> X Y",
				"X -> a",
				"Y -> a | b",
			},
		},
		{
			name:      "shibaji paul's example",
			terminals: []string{"a", "b", "c", "d"},
			rules: []string{
				"S -> a S b | A",
				"A -> c A d | c d",
			},
			expect: []string{
				"S -> a S b | c A d | c d",
				"A -> c A d | c d",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// setup

			assert := assert.New(t)
			var expectRules = make([]Rule, len(tc.expect))
			for i := range tc.expect {
				expectRules[i] = mustParseRule(tc.expect[i])
			}

			// set up the grammar
			g := Grammar{}
			for _, term := range tc.terminals {
				class := tokenClass{id: strings.ToLower(term), human: term}
				g.AddTerm(strings.ToLower(term), class)
			}
			for _, r := range tc.rules {
				parsedRule := mustParseRule(r)
				for _, alts := range parsedRule.Productions {
					g.AddRule(parsedRule.NonTerminal, alts)
				}
			}

			// execute
			actual := g.RemoveUnitProductions()

			// assert

			// terminals must remain unchanged
			assert.Equal(g.terminals, actual.terminals)

			// rules must be as expected, cant do equal bc we need custom
			// comparison logic for each
			assert.Len(actual.rules, len(expectRules), "grammar %s has incorrect number of rules", actual.String())

			minLen := len(actual.rules)
			if minLen > len(expectRules) {
				minLen = len(expectRules)
			}

			// priority DOES matter so we check them in order
			for i := 0; i < minLen; i++ {
				exp := expectRules[i]
				act := actual.rules[i]

				assert.Truef(exp.Equal(act), "expected rules[%d] to be %q but was %q", i, exp.String(), act.String())
			}
		})
	}
}
