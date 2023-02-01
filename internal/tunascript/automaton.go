package tunascript

import (
	"fmt"
	"strings"
)

type FATransition struct {
	input string
	next  string
}

type NFAState[E fmt.Stringer] struct {
	name        string
	value       E
	transitions map[string][]FATransition
}

type NFA[E fmt.Stringer] struct {
	states map[string]NFAState[E]
}

func (nfa *NFA[E]) AddState(state E) {
	if _, ok := nfa.states[state.String()]; ok {
		// Gr8! We are done.
		return
	}

	newState := NFAState[E]{
		name:        state.String(),
		value:       state,
		transitions: make(map[string][]FATransition),
	}

	nfa.states[state.String()] = newState
}

func (nfa *NFA[E]) AddTransition(fromState E, input string, toState E) {
	curFromState, ok := nfa.states[fromState.String()]

	if !ok {
		// Can't let you do that, Starfox
		panic(fmt.Sprintf("add transition from non-existent state %q", fromState.String()))
	}
	if _, ok := nfa.states[toState.String()]; ok {
		// I'm afraid I can't do that, Dave
		panic(fmt.Sprintf("add transition to non-existent state %q", toState.String()))
	}

	curInputTransitions, ok := curFromState.transitions[input]
	if !ok {
		curInputTransitions = make([]FATransition, 0)
	}

	newTransition := FATransition{
		input: input,
		next:  toState.String(),
	}

	curInputTransitions = append(curInputTransitions, newTransition)

	curFromState.transitions[input] = curInputTransitions
	nfa.states[fromState.String()] = curFromState
}

func ViablePrefixNDA(g Grammar) {
	// we are about to modify the grammar, get a copy
	g = g.Copy()

	// add the dummy production
	dummySym := g.GenerateUniqueName(g.StartSymbol())
	g.AddRule(dummySym, []string{g.StartSymbol()})
	g.Start = dummySym

	nfa := NFA[LR0Item]{}

	items := g.LRItems()

	// The NFA states are the items of G
	// (including the extra production)

	// add all of them first so we don't accidentally panic on adding
	// transitions
	for i := range items {
		nfa.AddState(items[i])
	}

	for i := range items {
		item := items[i]

		if len(item.Right) < 1 {
			// don't deal w E -> αXβ. (dot at right) because it's not useful.
			continue
		}

		alpha := item.Left
		X := item.Right[0]
		beta := item.Right[1:]

		// For item E -> α.Xβ, where X is any grammar symbol, add transition:
		//
		// E -> α.Xβ  =X=>  E -> αX.β
		toItem := LR0Item{
			NonTerminal: item.NonTerminal,
			Left:        append(alpha, X),
			Right:       beta,
		}
		nfa.AddTransition(item, X, toItem)

		// For item E -> α.Xβ and production X -> γ (X is a non-terminal), add
		// transition:
		//
		// E -> α.Xβ  =ε=>  X -> .γ
		if strings.ToUpper(X) == X {
			// need to do this for every production of X
			gammas := g.Rule(X).Productions
			for _, gamma := range gammas {
				prodState := LR0Item{
					NonTerminal: X,
					Right:       gamma,
				}

				nfa.AddTransition(item, "", prodState)
			}
		}
	}
}
