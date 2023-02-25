package automaton

import (
	"fmt"
	"strings"

	"github.com/dekarrin/tunaq/internal/ictiobus/grammar"
	"github.com/dekarrin/tunaq/internal/util"
)

type NFA[E any] struct {
	states map[string]NFAState[E]
	Start  string
}

type NFATransitionTo struct {
	from  string
	input string
	index int
}

func (nfa NFA[E]) AcceptingStates() util.StringSet {
	accepting := util.NewStringSet()
	allStates := nfa.States().Elements()
	for i := range allStates {
		if nfa.states[allStates[i]].accepting {
			accepting.Add(allStates[i])
		}
	}

	return accepting
}

// returns a list of 2-tuples that have (fromState, input)
func (nfa NFA[E]) AllTransitionsTo(toState string) []NFATransitionTo {
	if _, ok := nfa.states[toState]; !ok {
		// Gr8! We are done.
		return []NFATransitionTo{}
	}

	transitions := []NFATransitionTo{}

	s := nfa.States()

	for _, sName := range s.Elements() {
		state := nfa.states[sName]
		for k := range state.transitions {
			for i := range state.transitions[k] {
				if state.transitions[k][i].next == toState {
					trans := NFATransitionTo{
						from:  sName,
						input: k,
						index: i,
					}
					transitions = append(transitions, trans)
				}
			}
		}
	}

	return transitions
}

// Copy returns a duplicate of this NFA.
func (nfa NFA[E]) Copy() NFA[E] {
	copied := NFA[E]{
		Start:  nfa.Start,
		states: make(map[string]NFAState[E]),
	}

	for k := range nfa.states {
		copied.states[k] = nfa.states[k].Copy()
	}

	return copied
}

// States returns all states in the dfa.
func (nfa NFA[E]) States() util.StringSet {
	states := util.NewStringSet()

	for k := range nfa.states {
		states.Add(k)
	}

	return states
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

// does a direct conversion of nfa to dfa without joining any states. this is NOT
// a merging algorithm; it will return an error if the given NFA[E] is not
// already de-facto deterministic.
func directNFAToDFA[E any](nfa NFA[E]) (DFA[E], error) {
	dfa := DFA[E]{
		Start:  nfa.Start,
		states: map[string]DFAState[E]{},
	}

	for sName := range nfa.states {
		nState := nfa.states[sName]

		dState := DFAState[E]{
			name:        nState.name,
			value:       nState.value,
			transitions: map[string]FATransition{},
			accepting:   nState.accepting,
		}

		for sym := range nState.transitions {
			nTransList := nState.transitions[sym]

			goesTo := ""
			for i := range nTransList {
				if nTransList[i].next == "" {
					return DFA[E]{}, fmt.Errorf("state %q has empty transition-to for %q", nState.name, sym)
				}
				if goesTo == "" {
					// first time we are seeing this, set it now
					goesTo = nTransList[i].next
					dState.transitions[sym] = FATransition{
						input: sym,
						next:  nTransList[i].next,
					}
				} else {
					// if there's more transitions, they simply need to go to the
					// same place.
					if nTransList[i].next != goesTo {
						return DFA[E]{}, fmt.Errorf("state %q has non-deterministic transition for symbol %q", nState.name, sym)
					}
				}
			}
		}

		dfa.states[sName] = dState
	}

	return dfa, nil
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

// NumberStates renames all states to each have a unique name based on an
// increasing number sequence. The starting state is guaranteed to be numbered
// 0; beyond that, the states are put in alphabetical order.
func (nfa *NFA[E]) NumberStates() {
	if _, ok := nfa.states[nfa.Start]; !ok {
		panic("can't number states of NFA with no start state set")
	}
	origStateNames := util.OrderedKeys(nfa.States())

	// make shore to pull out starting state and place at front
	startIdx := -1
	for i := range origStateNames {
		if origStateNames[i] == nfa.Start {
			startIdx = i
			break
		}
	}
	if startIdx == -1 {
		panic("couldn't find starting state; should never happen")
	}

	origStateNames = append(origStateNames[:startIdx], origStateNames[startIdx+1:]...)
	origStateNames = append([]string{nfa.Start}, origStateNames...)

	numMapping := map[string]string{}
	for i := range origStateNames {
		name := origStateNames[i]
		newName := fmt.Sprintf("%d", i)
		numMapping[name] = newName
	}

	// to keep things simple, instead of searching for every instance of each
	// name which is an expensive operation, we'll just build an entirely new
	// NFA using our mapping rules to adjust names as we go, then steal its
	// states map.

	newNfa := NFA[E]{
		states: make(map[string]NFAState[E]),
		Start:  numMapping[nfa.Start],
	}

	// first, add the initial states
	for _, name := range origStateNames {
		st := nfa.states[name]
		newName := numMapping[name]
		newNfa.AddState(newName, st.accepting)
		newNfa.SetValue(newName, st.value)

		// transitions come later, need to add all states *first*
	}

	// add initial transitions
	for _, name := range origStateNames {
		st := nfa.states[name]
		from := numMapping[name]

		for sym := range st.transitions {
			symTrans := st.transitions[sym]
			for i := range symTrans {
				t := symTrans[i]
				to := numMapping[t.next]
				newNfa.AddTransition(from, sym, to)
			}
		}
	}

	// oh ya, just gonna go ahead and sneeeeeeeak this on away from ya
	nfa.states = newNfa.states
	nfa.Start = newNfa.Start
}

// Join combines two NFAs into a single one. The argument fromToOther gives the
// method of joining the two NFAs; it is a slice of triples, each of which gives
// a state from the original nfa, the symbol to transition on, and a state in
// the provided NFA to go to on receiving that symbol.
//
// The original NFAs are not modified. The resulting NFA's start state is the
// same as the original NFA's start state.
//
// In order to prevent conflicts, all state names in the resulting NFA will be
// named according to a scheme that namespaces them by which NFA they came from;
// states that came from the original NFA will be changed to be called
// '1:ORIGNAL_NAME' in the resulting NFA, and states that came from the provided
// NFA will be changed to be called '2:ORIGINAL_NAME' in the resulting NFA, with
// 'ORIGINAL_NAME' replaced with the actual original name of the state.
//
// After the resulting NFA is created, all state names listed in addAccept will
// be changed to accepting states in the resulting NFA. Likewise, all state
// names listed in removeAccept will be changed to no longer be accepting in the
// resulting DFA.
//
// Note that because addAccept and removeAccept are applied to the resulting NFA
// after creation, they must use the state-naming convention mentioned above,
// while states mentioned in fromToOther should use the original names of the
// states.
func (nfa NFA[E]) Join(other NFA[E], fromToOther [][3]string, otherToFrom [][3]string, addAccept []string, removeAccept []string) (NFA[E], error) {
	if len(fromToOther) < 1 {
		return NFA[E]{}, fmt.Errorf("need to provide at least one mapping in fromToOther")
	}

	joined := NFA[E]{
		states: make(map[string]NFAState[E]),
		Start:  "1:" + nfa.Start,
	}

	addAcceptSet := util.StringSetOf(addAccept)
	removeAcceptSet := util.StringSetOf(removeAccept)

	nfaStateNames := joined.States()

	// first, add the initial states
	for _, stateName := range nfaStateNames.Elements() {
		st := nfa.states[stateName]
		newName := "1:" + stateName

		accept := st.accepting
		if addAcceptSet.Has(newName) {
			accept = true
		} else if removeAcceptSet.Has(newName) {
			accept = false
		}
		joined.AddState(newName, accept)
		joined.SetValue(newName, st.value)

		// transitions come later, need to add all states *first*
	}

	// add initial transitions
	for _, stateName := range nfaStateNames.Elements() {
		st := nfa.states[stateName]
		from := "1:" + stateName

		for sym := range st.transitions {
			symTrans := st.transitions[sym]
			for i := range symTrans {
				t := symTrans[i]
				to := "1:" + t.next
				joined.AddTransition(from, sym, to)
			}
		}
	}

	// next, do the same for the second NFA
	otherStateNames := other.States()

	for _, stateName := range otherStateNames.Elements() {
		st := other.states[stateName]
		newName := "2:" + stateName

		accept := st.accepting
		if addAcceptSet.Has(newName) {
			accept = true
		} else if removeAcceptSet.Has(newName) {
			accept = false
		}
		joined.AddState(newName, accept)
		joined.SetValue(newName, st.value)

		// transitions come later, need to add all states *first*
	}

	// add other transitions
	for _, stateName := range otherStateNames.Elements() {
		st := other.states[stateName]
		from := "2:" + stateName

		for sym := range st.transitions {
			symTrans := st.transitions[sym]
			for i := range symTrans {
				t := symTrans[i]
				to := "2:" + t.next
				joined.AddTransition(from, sym, to)
			}
		}
	}

	// already did accept adjustment on the fly, now it's time to link the
	// states together
	for i := range fromToOther {
		link := fromToOther[i]
		from := "1:" + link[0]
		sym := link[1]
		to := "2:" + link[2]
		joined.AddTransition(from, sym, to)
	}
	for i := range otherToFrom {
		link := otherToFrom[i]
		from := "2:" + link[0]
		sym := link[1]
		to := "1:" + link[2]
		joined.AddTransition(from, sym, to)
	}

	return joined, nil
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
