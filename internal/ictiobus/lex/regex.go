package lex

import (
	"github.com/dekarrin/tunaq/internal/ictiobus/automaton"
)

// TODO: fill this all in when we want to return to DFA-based impl. for now,
// lex package just uses the pre-built regex processors bc they are easier glub.

// RegexToDFA takes the given regular expression and converts it into a DFA.
//
// This is an implementation of algorithm 3.23 "The McNaughton-Yamada-Thompson
// algorithm to convert a regular expression to an NFA."
func RegexToNFA(r string) automaton.NFA[string] {
	// it would be neat if this just sort of used our own parsing algos for
	// this. sadly, no part of ictiobus is self-hosted, and that includes the
	// lexer.
	return automaton.NFA[string]{}
}

// for any subexpression r in sigma, or epsilon.
func createSingleSymbolFA(symbol string) automaton.NFA[string] {
	var nfa automaton.NFA[string]

	nfa.AddState("A", false)
	nfa.AddState("B", true)
	nfa.AddTransition("A", symbol, "B")
	nfa.Start = "A"

	return nfa
}

// for any expression st.
func createJuxtapositionFA(left, right automaton.NFA[string]) automaton.NFA[string] {
	accept := getSingleAcceptState(left)

	nfa, err := left.Join(&right, [][3]string{{accept, "", right.Start}}, nil, nil, []string{"1:" + accept})
	if err != nil {
		panic(err.Error())
	}

	return *nfa
}

// regex simplification rewrites:
//
// . -> directly interpret. all but newline.
// ^ -> \n|''

func createKleeneStarFA(expr automaton.NFA[string]) automaton.NFA[string] {
	exprAccept := getSingleAcceptState(expr)

	// add an epsilon transition from start to end of the expr
	exprCopy := expr.Copy()
	expr = *exprCopy
	expr.AddTransition(exprAccept, "", expr.Start)

	var nfa *automaton.NFA[string]
	var err error

	nfa.AddState("A", false)
	nfa.AddState("B", true)
	nfa.AddTransition("A", "", "B")
	nfa.Start = "A"
	nfaAccept := "B"

	nfa, err = nfa.Join(&expr, [][3]string{{nfa.Start, "", expr.Start}}, [][3]string{{exprAccept, "", nfaAccept}}, nil, []string{"1:" + exprAccept})
	if err != nil {
		panic(err.Error())
	}

	return *nfa
}

// for any expression s|t, but s and t need to already have been turned to NFAs.
func createAlternationFA(left, right automaton.NFA[string]) automaton.NFA[string] {
	// we know that the only accepting state in the input automatons is their
	// final state, so can just grab them and verify now
	leftAccept := getSingleAcceptState(left)
	rightAccept := getSingleAcceptState(right)

	var nfa *automaton.NFA[string]
	var err error

	nfa.AddState("A", false)
	nfa.AddState("B", true)
	nfa.Start = "A"
	nfaAccept := "B"

	// join with left side
	nfa, err = nfa.Join(&left, [][3]string{{nfa.Start, "", left.Start}}, [][3]string{{leftAccept, "", nfaAccept}}, nil, []string{"1:" + leftAccept})
	if err != nil {
		panic(err.Error())
	}

	// join with right side
	nfaAccept = getSingleAcceptState(*nfa)
	nfa, err = nfa.Join(&right, [][3]string{{nfa.Start, "", right.Start}}, [][3]string{{rightAccept, "", nfaAccept}}, nil, []string{"1:" + rightAccept})
	if err != nil {
		panic(err.Error())
	}

	return *nfa
}

// panics if there is not exactly one accepting state in provided nfa
func getSingleAcceptState(nfa automaton.NFA[string]) string {
	allAcceptStates := nfa.AcceptingStates()
	if allAcceptStates.Len() != 1 {
		panic("NFA has multiple acceptance states")
	}

	// we just verified there's exactly one element in set and can now do this:
	var accept string
	for k := range allAcceptStates {
		accept = k
	}

	return accept
}
