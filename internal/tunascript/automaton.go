package tunascript

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dekarrin/tunaq/internal/util"
)

type FATransition struct {
	input string
	next  string
}

func (t FATransition) String() string {
	inp := t.input
	if inp == "" {
		inp = "ε"
	}
	return fmt.Sprintf("=(%s)=> %s", inp, t.next)
}

func mustParseFATransition(s string) FATransition {
	t, err := parseFATransition(s)
	if err != nil {
		panic(err.Error())
	}
	return t
}

func parseFATransition(s string) (FATransition, error) {
	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, " ", 2)

	left, right := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])

	if len(left) < 3 {
		return FATransition{}, fmt.Errorf("not a valid FATransition: left len < 3: %q", left)
	}

	if left[0] != '=' {
		return FATransition{}, fmt.Errorf("not a valid FATransition: left[0] != '=': %q", left)
	}
	if left[1] != '(' {
		return FATransition{}, fmt.Errorf("not a valid FATransition: left[1] != '(': %q", left)
	}
	left = left[2:]
	// also chop off the ending arrow
	if len(left) < 4 {
		return FATransition{}, fmt.Errorf("not a valid left: len(chopped) < 4: %q", left)
	}
	if left[len(left)-1] != '>' {
		return FATransition{}, fmt.Errorf("not a valid left: chopped[-1] != '>': %q", left)
	}
	if left[len(left)-2] != '=' {
		return FATransition{}, fmt.Errorf("not a valid left: chopped[-2] != '=': %q", left)
	}
	if left[len(left)-3] != ')' {
		return FATransition{}, fmt.Errorf("not a valid left: chopped[-3] != ')': %q", left)
	}
	input := left[:len(left)-3]
	if input == "ε" {
		input = ""
	}

	// next is EASY af
	next := right
	if next == "" {
		return FATransition{}, fmt.Errorf("not a valid FATransition: bad next: %q", s)
	}

	return FATransition{
		input: input,
		next:  next,
	}, nil
}

type DFAState[E fmt.Stringer] struct {
	name        string
	value       E
	transitions map[string]FATransition
	accepting   bool
}

func (ns DFAState[E]) String() string {
	var moves strings.Builder

	inputs := util.OrderedKeys(ns.transitions)

	for i, input := range inputs {
		moves.WriteString(ns.transitions[input].String())
		if i+1 < len(inputs) {
			moves.WriteRune(',')
			moves.WriteRune(' ')
		}
	}

	str := fmt.Sprintf("(%s [%s])", ns.name, moves.String())

	if ns.accepting {
		str = "(" + str + ")"
	}

	return str
}

type NFAState[E fmt.Stringer] struct {
	name        string
	value       E
	transitions map[string][]FATransition
	accepting   bool
}

func (ns NFAState[E]) String() string {
	var moves strings.Builder

	inputs := util.OrderedKeys(ns.transitions)

	for i, input := range inputs {
		var tStrings []string

		for _, t := range ns.transitions[input] {
			tStrings = append(tStrings, t.String())
		}

		sort.Strings(tStrings)

		for tIdx, t := range tStrings {
			moves.WriteString(t)
			if tIdx+1 < len(tStrings) || i+1 < len(inputs) {
				moves.WriteRune(',')
				moves.WriteRune(' ')
			}
		}
	}

	str := fmt.Sprintf("(%s [%s])", ns.name, moves.String())

	if ns.accepting {
		str = "(" + str + ")"
	}

	return str
}

type DFA[E fmt.Stringer] struct {
	states map[string]DFAState[E]
	start  string
}

func (dfa DFA[E]) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("<START: %q, STATES:", dfa.start))

	orderedStates := util.OrderedKeys(dfa.states)

	for i := range orderedStates {
		sb.WriteString("\n\t")
		sb.WriteString(dfa.states[orderedStates[i]].String())

		if i+1 < len(dfa.states) {
			sb.WriteRune(',')
		} else {
			sb.WriteRune('\n')
		}
	}

	sb.WriteRune('>')

	return sb.String()
}

type NFA[E fmt.Stringer] struct {
	states map[string]NFAState[E]
	start  string
}

// EpsilonClosure gives the set of states reachable from state using one or more
// ε-moves.
func (nfa NFA[E]) EpsilonClosure(s string) util.Set[string] {
	stateItem, ok := nfa.states[s]
	if !ok {
		panic(fmt.Sprintf("not a state in NFA: %q", s))
	}

	closure := util.Set[string]{}
	checkingStates := util.Stack[NFAState[E]]{}
	checkingStates.Push(stateItem)

	for checkingStates.Len() > 0 {
		checking := checkingStates.Pop()

		if closure.Has(checking.name) {
			// we've already checked it. skip.
			continue
		}

		// add it to the closure and then check it for recursive closures
		closure.Add(checking.name)

		epsilonMoves, hasEpsilons := checking.transitions[""]
		if !hasEpsilons {
			continue
		}

		for _, move := range epsilonMoves {
			stateName := move.next
			state, ok := nfa.states[stateName]
			if !ok {
				// should never happen unless someone manually adds to
				// unexported properties; AddTransition ensures that only valid
				// and followable transitions are allowed to be added.
				panic(fmt.Sprintf("points to invalid state: %q", stateName))
			}

			checkingStates.Push(state)
		}
	}

	return closure
}

func (nfa NFA[E]) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("<START: %q, STATES:", nfa.start))

	orderedStates := util.OrderedKeys(nfa.states)

	for i := range orderedStates {
		sb.WriteString("\n\t")
		sb.WriteString(nfa.states[orderedStates[i]].String())

		if i+1 < len(nfa.states) {
			sb.WriteRune(',')
		} else {
			sb.WriteRune('\n')
		}
	}

	sb.WriteRune('>')

	return sb.String()
}

func (nfa *NFA[E]) AddState(state E, accepting bool) {
	if _, ok := nfa.states[state.String()]; ok {
		// Gr8! We are done.
		return
	}

	newState := NFAState[E]{
		name:        state.String(),
		value:       state,
		transitions: make(map[string][]FATransition),
		accepting:   accepting,
	}

	if nfa.states == nil {
		nfa.states = map[string]NFAState[E]{}
	}

	nfa.states[state.String()] = newState
}

func (nfa *NFA[E]) AddTransition(fromState E, input string, toState E) {
	toName := toState.String()
	fromName := fromState.String()
	curFromState, ok := nfa.states[fromName]

	if !ok {
		// Can't let you do that, Starfox
		panic(fmt.Sprintf("add transition from non-existent state %q", fromState.String()))
	}
	if _, ok := nfa.states[toName]; !ok {
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

func NewViablePrefixNDA(g Grammar) NFA[LR0Item] {
	// we are about to modify the grammar, get a copy
	g = g.Copy()

	// add the dummy production
	oldStart := g.StartSymbol()
	dummySym := g.GenerateUniqueName(oldStart)
	g.AddRule(dummySym, []string{oldStart})
	g.Start = dummySym

	nfa := NFA[LR0Item]{}

	// set the start state
	nfa.start = LR0Item{NonTerminal: dummySym, Right: []string{oldStart}}.String()

	items := g.LRItems()

	// The NFA states are the items of G
	// (including the extra production)

	// add all of them first so we don't accidentally panic on adding
	// transitions
	for i := range items {
		nfa.AddState(items[i], true)
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

	return nfa
}
