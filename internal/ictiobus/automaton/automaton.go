package automaton

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dekarrin/tunaq/internal/ictiobus/grammar"
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

type DFAState[E any] struct {
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

type NFAState[E any] struct {
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

type DFA[E any] struct {
	states map[string]DFAState[E]
	Start  string
}

func (dfa *DFA[E]) SetValue(state string, v E) {
	s, ok := dfa.states[state]
	if !ok {
		panic(fmt.Sprintf("setting value on non-existing state: %q", state))
	}
	s.value = v
	dfa.states[state] = s
}

func (dfa *DFA[E]) GetValue(state string) E {
	s, ok := dfa.states[state]
	if !ok {
		panic(fmt.Sprintf("getting value on non-existing state: %q", state))
	}
	return s.value
}

// IsAccepting returns whether the given state is an accepting (terminating)
// state. Returns false if the state does not exist.
func (dfa DFA[E]) IsAccepting(state string) bool {
	s, ok := dfa.states[state]
	if !ok {
		return false
	}

	return s.accepting
}

// States returns all states in the dfa.
func (dfa DFA[E]) States() util.StringSet {
	states := util.NewStringSet()

	for k := range dfa.states {
		states.Add(k)
	}

	return states
}

// Next returns the next state of the DFA, given a current state and an input.
// Will return "" if state is not an existing state or if there is no transition
// from the given state on the given input.
func (dfa DFA[E]) Next(fromState string, input string) string {
	state, ok := dfa.states[fromState]
	if !ok {
		return ""
	}

	transition, ok := state.transitions[input]
	if !ok {
		return ""
	}

	return transition.next
}

func (dfa *DFA[E]) AddState(state string, accepting bool) {
	if _, ok := dfa.states[state]; ok {
		// Gr8! We are done.
		return
	}

	newState := DFAState[E]{
		name:        state,
		transitions: make(map[string]FATransition),
		accepting:   accepting,
	}

	if dfa.states == nil {
		dfa.states = map[string]DFAState[E]{}
	}

	dfa.states[state] = newState
}

func (dfa *DFA[E]) AddTransition(fromState string, input string, toState string) {
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

func (dfa DFA[E]) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("<START: %q, STATES:", dfa.Start))

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

type NFA[E any] struct {
	states map[string]NFAState[E]
	Start  string
}

// ToDFA converts the NFA into a deterministic finite automaton accepting the
// same strings.
//
// This is an implementation of algorithm 3.20 from the purple dragon book.
func (nfa NFA[E]) ToDFA() DFA[util.SVSet[E]] {
	inputSymbols := nfa.InputSymbols()

	Dstart := nfa.EpsilonClosure(nfa.Start)

	markedStates := util.NewStringSet()
	Dstates := map[string]util.StringSet{}
	Dstates[Dstart.StringOrdered()] = Dstart

	// these are Dstates but represented in actual format for placement into
	// our implement8ion of DFAs, which is also where transition function info
	// and acceptance info is stored.
	dfa := DFA[util.SVSet[E]]{
		states: map[string]DFAState[util.SVSet[E]]{},
	}

	// initially, ε-closure(s₀) is the only state in Dstates, and it is unmarked
	for {
		// get unmarked states in Dstates
		DstateNames := util.StringSetOf(util.OrderedKeys(Dstates))
		unmarkedStates := DstateNames.Difference(markedStates)

		if unmarkedStates.Len() < 1 {
			break
		}
		// while ( there is an unmarked state T in Dstates )
		for _, Tname := range unmarkedStates.Elements() {
			T := Dstates[Tname]

			// mark T
			markedStates.Add(Tname)

			// (need to get the value of every item to get a set of them)
			stateValues := util.NewSVSet[E]()
			for nfaStateName := range T {
				val := nfa.GetValue(nfaStateName)
				stateValues.Set(nfaStateName, val)
			}

			newDFAState := DFAState[util.SVSet[E]]{name: Tname, value: stateValues, transitions: map[string]FATransition{}}

			if T.Any(func(v string) bool {
				return nfa.states[v].accepting
			}) {
				newDFAState.accepting = true
			}

			// for ( each input symbol a )
			for a := range inputSymbols {
				// (but like, glub, not the epsilon symbol itself)
				if a == grammar.Epsilon[0] {
					continue
				}

				U := nfa.EpsilonClosureOfSet(nfa.MOVE(T, a))

				// if its not a symbol that the state can transition on, U will
				// be empty, skip it
				if U.Empty() {
					continue
				}

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

			if dfa.Start == "" {
				// then T is our starting state.
				dfa.Start = Tname
			}
		}

	}
	return dfa
}

// InputSymbols returns the set of all input symbols processed by some
// transition in the NFA.
func (nfa NFA[E]) InputSymbols() util.StringSet {
	symbols := util.NewStringSet()
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
func (nfa NFA[E]) MOVE(X util.ISet[string], a string) util.StringSet {
	moves := util.NewStringSet()

	for _, s := range X.Elements() {
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
func (nfa NFA[E]) EpsilonClosureOfSet(X util.ISet[string]) util.StringSet {
	allClosures := util.NewStringSet()

	for _, s := range X.Elements() {
		closures := nfa.EpsilonClosure(s)
		allClosures.AddAll(closures)
	}

	return allClosures
}

// EpsilonClosure gives the set of states reachable from state using one or more
// ε-moves.
func (nfa NFA[E]) EpsilonClosure(s string) util.StringSet {
	stateItem, ok := nfa.states[s]
	if !ok {
		return nil
	}

	closure := util.NewStringSet()
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

	sb.WriteString(fmt.Sprintf("<START: %q, STATES:", nfa.Start))

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

func (nfa *NFA[E]) AddState(state string, accepting bool) {
	if _, ok := nfa.states[state]; ok {
		// Gr8! We are done.
		return
	}

	newState := NFAState[E]{
		name:        state,
		transitions: make(map[string][]FATransition),
		accepting:   accepting,
	}

	if nfa.states == nil {
		nfa.states = map[string]NFAState[E]{}
	}

	nfa.states[state] = newState
}

func (nfa *NFA[E]) SetValue(state string, v E) {
	s, ok := nfa.states[state]
	if !ok {
		panic(fmt.Sprintf("setting value on non-existing state: %q", state))
	}
	s.value = v
	nfa.states[state] = s
}

func (nfa *NFA[E]) GetValue(state string) E {
	s, ok := nfa.states[state]
	if !ok {
		panic(fmt.Sprintf("getting value on non-existing state: %q", state))
	}
	return s.value
}

func (nfa *NFA[E]) AddTransition(fromState string, input string, toState string) {
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

func NewLR1ViablePrefixDFA(g grammar.Grammar) DFA[util.SVSet[grammar.LR1Item]] {
	oldStart := g.StartSymbol()
	g = g.Augmented()

	initialItem := grammar.LR1Item{
		LR0Item: grammar.LR0Item{
			NonTerminal: g.StartSymbol(),
			Right:       []string{oldStart},
		},
		Lookahead: "$",
	}

	startSet := g.LR1_CLOSURE(util.SVSet[grammar.LR1Item]{initialItem.String(): initialItem})

	stateSets := util.NewSVSet[util.SVSet[grammar.LR1Item]]()
	stateSets.Set(startSet.StringOrdered(), startSet)
	transitions := map[string]map[string]FATransition{}

	// following algo from http://www.cs.ecu.edu/karl/5220/spr16/Notes/Bottom-up/lr1.html
	updates := true
	for updates {
		updates = false

		// suppose that state q contains set I of LR(1) items
		for _, I := range stateSets {

			for _, item := range I {
				if len(item.Right) == 0 || item.Right[0] == grammar.Epsilon[0] {
					continue // no epsilons, deterministic finite state
				}
				// For each symbol s (either a token or a nonterminal) that
				// immediately follows a dot in an LR(1) item [A → α ⋅ sβ, t] in
				// set I...
				s := item.Right[0]

				// ...let Is be the set of all LR(1) items in I where s
				// immediately follows the dot.
				Is := util.NewSVSet[grammar.LR1Item]()
				for _, checkItem := range I {
					if len(checkItem.Right) >= 1 && checkItem.Right[0] == s {
						newItem := checkItem.Copy()

						// Move the dot to the other side of s in each of them.
						newItem.Left = append(newItem.Left, s)
						newItem.Right = make([]string, len(checkItem.Right)-1)
						copy(newItem.Right, checkItem.Right[1:])

						Is.Set(newItem.String(), newItem)
					}
				}

				// That set [Is] becomes the kernel of state q', and you make a
				// transition from q to q′ on s. As usual, form the closure of
				// the set of LR(1) items in state q'.
				newSet := g.LR1_CLOSURE(Is)

				// add to states if not already in it
				if !stateSets.Has(newSet.StringOrdered()) {
					updates = true
					stateSets.Set(newSet.StringOrdered(), newSet)
				}

				// add to transitions if not already in it
				stateTransitions, ok := transitions[I.StringOrdered()]
				if !ok {
					stateTransitions = map[string]FATransition{}
				}
				trans, ok := stateTransitions[s]
				if !ok {
					trans = FATransition{}
				}
				if trans.next != newSet.StringOrdered() {
					updates = true
					trans.input = s
					trans.next = newSet.StringOrdered()
					stateTransitions[s] = trans
					transitions[I.StringOrdered()] = stateTransitions
				}
			}
		}
	}

	// okay, we've actually pre-calculated all DFA items so we can now add them.
	// might be able to optimize to add on-the-fly during above loop but this is
	// easier for the moment.
	dfa := DFA[util.SVSet[grammar.LR1Item]]{}

	// add states
	for sName, state := range stateSets {
		dfa.AddState(sName, true)
		dfa.SetValue(sName, state)
	}

	// transitions
	for onState, stateTrans := range transitions {
		for _, t := range stateTrans {
			dfa.AddTransition(onState, t.input, t.next)
		}
	}

	// and start
	dfa.Start = startSet.StringOrdered()

	return dfa
}

// Creates an NDA for all LR0 items of augmented grammar g'. The augmented
// grammar is created by taking the start symbol S of g and adding a new
// production, S' -> S, as the new start symbol.
//
// The value at each state will be the string encoding of the LR0 item it
// represents. To get a DFA whose states and values at each are the epsilon
// closures of the transitions, call ToDFA on the output of this function.
//
// To get a DFA whose values are
func NewLR0ViablePrefixNFA(g grammar.Grammar) NFA[grammar.LR0Item] {
	// add the dummy production
	oldStart := g.StartSymbol()
	g = g.Augmented()

	nfa := NFA[grammar.LR0Item]{}

	// set the start state
	nfa.Start = grammar.LR0Item{NonTerminal: g.StartSymbol(), Right: []string{oldStart}}.String()

	items := g.LR0Items()

	// The NFA states are the items of G
	// (including the extra production)

	// add all of them first so we don't accidentally panic on adding
	// transitions
	for i := range items {
		nfa.AddState(items[i].String(), true)
		nfa.SetValue(items[i].String(), items[i])
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
		toItem := grammar.LR0Item{
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
				prodState := grammar.LR0Item{
					NonTerminal: X,
					Right:       gamma,
				}

				nfa.AddTransition(item.String(), "", prodState.String())
			}
		}
	}

	return nfa
}
