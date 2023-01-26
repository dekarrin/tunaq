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
		terminals []string
		rules     []string
		expect    []string
	}{
		{
			name: "empty grammar",
		},
		{
			name:      "single rule grammar, no epsilons",
			terminals: []string{"int"},
			rules:     []string{"S -> A"},
			expect:    []string{"S -> A"},
		},
		{
			name:      "deeba kannan's epsilon elimination example (TOC Lec 25)",
			terminals: []string{"a", "b"},
			rules: []string{
				"S -> A C A | A a",
				"A -> B B | ε",
				"B -> A | b C",
				"C -> b",
			},
			expect: []string{
				"S -> A C A | C A | A C | C | A a | a",
				"A -> B B | B",
				"B -> A | b C",
				"C -> b",
			},
		},
		{
			name:      "purple dragon book ex. 4.4.6",
			terminals: []string{"a", "b"},
			rules: []string{
				`S -> a S b S
					   | b S a S
					   | ε
				`,
			},
			expect: []string{
				`S -> a S b S
				   | a b S
				   | a S b
				   | a b
				   | b S a S
				   | b a S
				   | b S a
				   | b a
				`,
			},
		},
		{
			name:      "grammar (4.18) from purple dragon book",
			terminals: []string{"a", "b", "c", "d"},
			rules: []string{
				"S -> A a | b",
				"A -> A c | S d | ε",
			},
			expect: []string{
				"S -> A a | a | b",
				"A -> A c | c | S d",
			},
		},
		{
			name:      "before, after, and recursive use of epsilon-producer",
			terminals: []string{"a", "b", "c", "d"},
			rules: []string{
				"S -> A a | B B",
				"A -> A c | S d | ε",
				"B -> A b S A | d | d d",
			},
			expect: []string{
				"S -> A a | a | B B",
				"A -> A c | c | S d",
				"B -> A b S A | b S A | A b S | b S | d | d d",
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
			actual := g.RemoveEpsilons()

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

func Test_Grammar_RemoveLeftRecursion(t *testing.T) {
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
			name:      "grammar with no left recursion",
			terminals: []string{"a", "b"},
			rules: []string{
				"S -> b A | b",
				"A -> a",
			},
			expect: []string{
				"S -> b A | b",
				"A -> a",
			},
		},
		{
			name:      "rule with immediate recursion only",
			terminals: []string{"a", "b"},
			rules: []string{
				"S -> b A | b",
				"A -> A a",
			},
			expect: []string{
				"S -> b A | b",
				"A -> a A | ε",
			},
		},
		{
			name:      "rule with immediate left recursion and other prods",
			terminals: []string{"a", "b"},
			rules: []string{
				"S -> b A | b",
				"A -> A a | a",
			},
			expect: []string{
				"S   -> b A | b",
				"A   -> a A-P",
				"A-P -> a A-P | ε",
			},
		},
		{
			name:      "indirect left recursion",
			terminals: []string{"a", "b"},
			rules: []string{
				"S -> b A | b",
				"A -> B a | a B a b",
				"B -> A b | b b b",
			},
			expect: []string{
				"S   -> b A | b",
				"A   -> b b b a A-P | a B a b A-P",
				"A-P -> b a A-P | ε",
				"B   -> A b | b b b",
			},
		},
		{
			name:      "indirect left recursion resolution eliminates rules made unreachable",
			terminals: []string{"a", "b"},
			rules: []string{
				"S -> b A | b",
				"A -> B a | a a b",
				"B -> A b | b b b",
			},
			expect: []string{
				"S   -> b A | b",
				"A   -> b b b a A-P | a a b A-P",
				"A-P -> b a A-P | ε",
			},
		},
		{
			// slightly diverges from actual 4.20 answer bc in that case they
			// explain the fact that the epsilon prod will not be an issue, but
			// that is by chance and the algo does not rely on chance and so
			// will ALWAYS remove epsilon prods.
			name:      "purple dragon book example 4.20 (blaze it) requires epsilon removal pre-step",
			terminals: []string{"a", "b", "c", "d"},
			rules: []string{
				"S -> A a | b",
				"A -> A c | S d | ε",
			},
			expect: []string{
				"S   -> c A-P a S-P | a S-P | b S-P",
				"S-P -> d A-P a S-P | ε",
				"A-P -> c A-P | ε",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// setup
			assert := assert.New(t)
			expect := setupGrammar(tc.terminals, tc.expect)
			g := setupGrammar(tc.terminals, tc.rules)

			// execute
			actual := g.RemoveLeftRecursion()

			// assert

			// terminals must remain unchanged
			assert.Equal(g.terminals, actual.terminals)
			assertIdenticalProductionSets(assert, expect, actual)
		})
	}
}

func Test_Grammar_LeftFactor(t *testing.T) {
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
			name:      "grammar glubglub",
			terminals: []string{"i", "t", "e", "a", "b"},
			rules: []string{
				"S -> i E t S | i E t S e S | a",
				"E -> b",
			},
			expect: []string{
				"S -> i E t S S-P | a",
				"E -> b",
				"S-P -> e S | ε",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// setup
			assert := assert.New(t)
			expect := setupGrammar(tc.terminals, tc.expect)
			g := setupGrammar(tc.terminals, tc.rules)

			// execute
			actual := g.LeftFactor()

			// assert

			// terminals must remain unchanged
			assert.Equal(g.terminals, actual.terminals)
			assertIdenticalProductionSets(assert, expect, actual)
		})
	}
}

func setupGrammar(terminals []string, rules []string) Grammar {
	g := Grammar{}

	for _, term := range terminals {
		class := tokenClass{id: strings.ToLower(term), human: term}
		g.AddTerm(class.id, class)
	}
	for _, r := range rules {
		parsedRule := mustParseRule(r)
		for _, alts := range parsedRule.Productions {
			g.AddRule(parsedRule.NonTerminal, alts)
		}
	}

	return g
}

// assertIdenticalProductionSets asserts whether the two grammars have the same
// nonterminals and that all nonterminals with the same name have the same sets
// of productions, not necessarily in the same order.
func assertIdenticalProductionSets(assert *assert.Assertions, expect, actual Grammar) {
	// priority does not matter so we check them in arbitrary order
	expectNonTerminals := expect.NonTerminals()
	actualNonTerminals := actual.NonTerminals()
	minLen := len(actualNonTerminals)
	if minLen > len(expectNonTerminals) {
		minLen = len(expectNonTerminals)
	}

	if !assert.ElementsMatch(expectNonTerminals, actualNonTerminals, "grammars do not have the same non-terminals") {
		return
	}

	for i := 0; i < minLen; i++ {
		ruleName := expectNonTerminals[i]
		exp := expect.Rule(ruleName)
		act := actual.Rule(ruleName)

		assert.ElementsMatchf(exp.Productions, act.Productions, "expected rule to have same prod set as %q but was %q", exp.String(), act.String())
	}
}
