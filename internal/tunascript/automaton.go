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

type DFAState struct {
	name        string
	transitions map[string]FATransition
	accepting   bool
}

func (ns DFAState) String() string {
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

type NFAState struct {
	name        string
	transitions map[string][]FATransition
	accepting   bool
}

func (ns NFAState) String() string {
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

type DFA struct {
	states map[string]DFAState
	start  string
}

func (dfa *DFA) AddState(state string, accepting bool) {
	if _, ok := dfa.states[state]; ok {
		// Gr8! We are done.
		return
	}

	newState := DFAState{
		name:        state,
		transitions: make(map[string]FATransition),
		accepting:   accepting,
	}

	if dfa.states == nil {
		dfa.states = map[string]DFAState{}
	}

	dfa.states[state] = newState
}

func (dfa *DFA) AddTransition(fromState string, input string, toState string) {
	curFromState, ok := dfa.states[fromState]

	if !ok {
		// Can't let you do that, Starfox
		panic(fmt.Sprintf("add transition from non-existent state %q", fromState))
	}
	if _, ok := dfa.states[toState]; !ok {
		// I'm afraid I can't do that, Dave
		panic(fmt.Sprintf("add transition to non-existent state %q", toState))
	}

	trans := FATransition{
		input: input,
		next:  toState,
	}

	curFromState.transitions[input] = trans
	dfa.states[fromState] = curFromState
}

func (dfa DFA) String() string {
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

type NFA struct {
	states map[string]NFAState
	start  string
}

// ToDFA converts the NFA into a deterministic finite automaton accepting the
// same strings.
//
// This is an implementation of algorithm 3.20 from the purple dragon book.
func (nfa NFA) ToDFA() DFA {
	inputSymbols := nfa.InputSymbols()

	Dstart := nfa.EpsilonClosure(nfa.start)

	markedStates := util.Set[string]{}
	Dstates := map[string]util.Set[string]{}
	Dstates[Dstart.StringOrdered()] = Dstart

	// these are Dstates but represented in actual format for placement into
	// our implement8ion of DFAs, which is also where transition function info
	// and acceptance info is stored.
	dfa := DFA{
		states: map[string]DFAState{},
	}

	// initially, ε-closure(s₀) is the only state in Dstates, and it is unmarked
	for {
		// get unmarked states in Dstates
		DstateNames := util.SetFromSlice(util.OrderedKeys(Dstates))
		unmarkedStates := DstateNames.Difference(markedStates)

		if unmarkedStates.Len() < 1 {
			break
		}
		// while ( there is an unmarked state T in Dstates )
		for Tname := range unmarkedStates {
			T := Dstates[Tname]

			// mark T
			markedStates.Add(Tname)

			newDFAState := DFAState{name: Tname, transitions: map[string]FATransition{}}

			if T.Any(func(v string) bool {
				return nfa.states[v].accepting
			}) {
				newDFAState.accepting = true
			}

			// for ( each input symbol a )
			for a := range inputSymbols {
				// (but like, glub, not the epsilon symbol itself)
				if a == Epsilon[0] {
					continue
				}

				U := nfa.EpsilonClosureOfSet(nfa.MOVE(T, a))

				// if U is not in Dstates
				if !DstateNames.Has(U.StringOrdered()) {
					// add U as an unmarked state to Dstates
					DstateNames.Add(U.StringOrdered())
					Dstates[U.StringOrdered()] = U
				}

				// Dtran[T, a] = U
				newDFAState.transitions[a] = FATransition{input: a, next: U.StringOrdered()}
			}

			// add it to our working DFA states as well
			dfa.states[Tname] = newDFAState

			if dfa.start == "" {
				// then T is our starting state.
				dfa.start = Tname
			}
		}

	}
	return dfa
}

// InputSymbols returns the set of all input symbols processed by some
// transition in the NFA.
func (nfa NFA) InputSymbols() util.Set[string] {
	symbols := util.Set[string]{}
	for sName := range nfa.states {
		st := nfa.states[sName]

		for a := range st.transitions {
			symbols.Add(a)
		}
	}

	return symbols
}

// MOVE returns the set of states reachable with one transition from some state
// in X on input a. Purple dragon book calls this function MOVE(T, a) and it is
// on page 153 as part of algorithm 3.20.
func (nfa NFA) MOVE(X util.Set[string], a string) util.Set[string] {
	moves := util.Set[string]{}

	for s := range X {
		stateItem, ok := nfa.states[s]
		if !ok {
			continue
		}

		transitions := stateItem.transitions[a]

		for _, t := range transitions {
			moves.Add(t.next)
		}
	}

	return moves
}

// EpsilonClosureOfSet gives the set of states reachable from some state in
// X using one or more ε-moves.
func (nfa NFA) EpsilonClosureOfSet(X util.Set[string]) util.Set[string] {
	allClosures := util.Set[string]{}

	for s := range X {
		closures := nfa.EpsilonClosure(s)
		allClosures.AddAll(closures)
	}

	return allClosures
}

// EpsilonClosure gives the set of states reachable from state using one or more
// ε-moves.
func (nfa NFA) EpsilonClosure(s string) util.Set[string] {
	stateItem, ok := nfa.states[s]
	if !ok {
		return nil
	}

	closure := util.Set[string]{}
	checkingStates := util.Stack[NFAState]{}
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

func (nfa NFA) String() string {
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

func (nfa *NFA) AddState(state string, accepting bool) {
	if _, ok := nfa.states[state]; ok {
		// Gr8! We are done.
		return
	}

	newState := NFAState{
		name:        state,
		transitions: make(map[string][]FATransition),
		accepting:   accepting,
	}

	if nfa.states == nil {
		nfa.states = map[string]NFAState{}
	}

	nfa.states[state] = newState
}

func (nfa *NFA) AddTransition(fromState string, input string, toState string) {
	curFromState, ok := nfa.states[fromState]

	if !ok {
		// Can't let you do that, Starfox
		panic(fmt.Sprintf("add transition from non-existent state %q", fromState))
	}
	if _, ok := nfa.states[toState]; !ok {
		// I'm afraid I can't do that, Dave
		panic(fmt.Sprintf("add transition to non-existent state %q", toState))
	}

	curInputTransitions, ok := curFromState.transitions[input]
	if !ok {
		curInputTransitions = make([]FATransition, 0)
	}

	newTransition := FATransition{
		input: input,
		next:  toState,
	}

	curInputTransitions = append(curInputTransitions, newTransition)

	curFromState.transitions[input] = curInputTransitions
	nfa.states[fromState] = curFromState
}

func NewViablePrefixNDA(g Grammar) NFA {
	// we are about to modify the grammar, get a copy
	g = g.Copy()

	// add the dummy production
	oldStart := g.StartSymbol()
	dummySym := g.GenerateUniqueName(oldStart)
	g.AddRule(dummySym, []string{oldStart})
	g.Start = dummySym

	nfa := NFA{}

	// set the start state
	nfa.start = LR0Item{NonTerminal: dummySym, Right: []string{oldStart}}.String()

	items := g.LRItems()

	// The NFA states are the items of G
	// (including the extra production)

	// add all of them first so we don't accidentally panic on adding
	// transitions
	for i := range items {
		nfa.AddState(items[i].String(), true)
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
		nfa.AddTransition(item.String(), X, toItem.String())

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

				nfa.AddTransition(item.String(), "", prodState.String())
			}
		}
	}

	return nfa
}
