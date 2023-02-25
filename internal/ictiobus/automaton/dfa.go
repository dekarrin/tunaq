package automaton

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dekarrin/tunaq/internal/ictiobus/grammar"
	"github.com/dekarrin/tunaq/internal/util"
)

// DFA is a deterministic finite automaton.
type DFA[E any] struct {
	order  uint64
	states map[string]DFAState[E]
	Start  string
}

// Copy returns a duplicate of this DFA.
func (dfa DFA[E]) Copy() DFA[E] {
	copied := DFA[E]{
		Start:  dfa.Start,
		states: make(map[string]DFAState[E]),
		order:  dfa.order,
	}

	for k := range dfa.states {
		copied.states[k] = dfa.states[k].Copy()
	}

	return copied
}

func TransformDFA[E1, E2 any](dfa DFA[E1], transform func(old E1) E2) DFA[E2] {
	copied := DFA[E2]{
		states: make(map[string]DFAState[E2]),
		Start:  dfa.Start,
		order:  dfa.order,
	}

	for k := range dfa.states {
		oldState := dfa.states[k]
		copiedState := DFAState[E2]{
			name:        oldState.name,
			value:       transform(oldState.value),
			transitions: make(map[string]FATransition),
			accepting:   oldState.accepting,
			ordering:    oldState.ordering,
		}

		for sym := range oldState.transitions {
			copiedState.transitions[sym] = oldState.transitions[sym]
		}
	}

	return copied
}

// DFAToNFA converts the DFA into an equivalent non-deterministic finite automaton
// type. Note that the type change doesn't suddenly make usage non-deterministic
// but it does allow for non-deterministic transitions to be added.
//
// TODO: generics hell if trying to make this a method on DFA. need to figure
// that out.
func DFAToNFA[E any](dfa DFA[E]) NFA[E] {
	nfa := NFA[E]{
		Start:  dfa.Start,
		states: map[string]NFAState[E]{},
		order:  dfa.order,
	}

	for sName := range dfa.states {
		dState := dfa.states[sName]

		nState := NFAState[E]{
			ordering:    dState.ordering,
			name:        dState.name,
			value:       dState.value,
			transitions: map[string][]FATransition{},
			accepting:   dState.accepting,
		}

		for sym := range dState.transitions {
			dTrans := dState.transitions[sym]
			nState.transitions[sym] = []FATransition{{input: dTrans.input, next: dTrans.next}}
		}

		nfa.states[sName] = nState
	}

	return nfa
}

// NumberStates renames all states to each have a unique name based on an
// increasing number sequence. The starting state is guaranteed to be numbered
// 0; beyond that, the states are put in alphabetical order.
func (dfa *DFA[E]) NumberStates() {
	if _, ok := dfa.states[dfa.Start]; !ok {
		panic("can't number states of DFA with no start state set")
	}
	origStateNames := util.OrderedKeys(dfa.States())

	// make shore to pull out starting state and place at front
	startIdx := -1
	for i := range origStateNames {
		if origStateNames[i] == dfa.Start {
			startIdx = i
			break
		}
	}
	if startIdx == -1 {
		panic("couldn't find starting state; should never happen")
	}

	origStateNames = append(origStateNames[:startIdx], origStateNames[startIdx+1:]...)
	origStateNames = append([]string{dfa.Start}, origStateNames...)

	numMapping := map[string]string{}
	for i := range origStateNames {
		name := origStateNames[i]
		newName := fmt.Sprintf("%d", i)
		numMapping[name] = newName
	}

	// to keep things simple, instead of searching for every instance of each
	// name which is an expensive operation, we'll just build an entirely new
	// DFA using our mapping rules to adjust names as we go, then steal its
	// states map.

	newDfa := &DFA[E]{
		states: make(map[string]DFAState[E]),
		Start:  numMapping[dfa.Start],
	}

	// first, add the initial states
	for _, name := range origStateNames {
		st := dfa.states[name]
		newName := numMapping[name]
		newDfa.AddState(newName, st.accepting)

		newSt := newDfa.states[newName]
		newSt.ordering = st.ordering
		newDfa.states[newName] = newSt

		newDfa.SetValue(newName, st.value)

		// transitions come later, need to add all states *first*
	}

	// add initial transitions
	for _, name := range origStateNames {
		st := dfa.states[name]
		from := numMapping[name]

		for sym := range st.transitions {
			t := st.transitions[sym]
			to := numMapping[t.next]
			newDfa.AddTransition(from, sym, to)
		}
	}

	// oh ya, just gonna go ahead and sneeeeeeeak this on away from ya
	dfa.states = newDfa.states
	dfa.Start = newDfa.Start
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

// Validate immediately returns an error if it finds the following:
//
// Any state impossible to reach (no transitions to it).
// Any transition leading to a state that doesn't exist.
// A start that isn't a state that exists.
func (dfa DFA[E]) Validate() error {
	errs := ""
	// all states must be reachable somehow. Must be reachable by some other
	// state if not the start state.
	for sName := range dfa.states {
		if sName == dfa.Start {
			continue
		}

		atLeastOneTransitionTo := false
		for otherName := range dfa.states {
			if otherName == sName {
				continue
			}

			st := dfa.states[otherName]

			for i := range st.transitions {
				if st.transitions[i].next == sName {
					atLeastOneTransitionTo = true
					break
				}
			}

			if atLeastOneTransitionTo {
				break
			}
		}
		if !atLeastOneTransitionTo {
			errs += fmt.Sprintf("\nno transitions to non-start state %q", sName)
		}
	}

	// all transitions must lead to an existing state
	for sName := range dfa.states {
		// dont skip if the starting state; this applies to that state too
		st := dfa.states[sName]

		for symbol := range st.transitions {
			nextState := st.transitions[symbol].next

			if _, ok := dfa.states[nextState]; !ok {
				errs += fmt.Sprintf("\nstate %q transitions to non-existing state: %q", sName, st.transitions[symbol])
			}
		}
	}

	// finally, start must be a reel state that exists
	if _, ok := dfa.states[dfa.Start]; !ok {
		errs += fmt.Sprintf("\nstart state does not exist: %q", dfa.Start)
	}

	if len(errs) > 0 {
		errs = errs[1:]
		return fmt.Errorf(errs)
	}

	return nil
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

// returns a list of 2-tuples that have (fromState, input)
func (dfa DFA[E]) AllTransitionsTo(toState string) [][2]string {
	if _, ok := dfa.states[toState]; !ok {
		// Gr8! We are done.
		return [][2]string{}
	}

	transitions := [][2]string{}

	s := dfa.States()

	for _, sName := range s.Elements() {
		state := dfa.states[sName]
		for k := range state.transitions {
			if state.transitions[k].next == toState {
				trans := [2]string{sName, k}
				transitions = append(transitions, trans)
			}
		}
	}

	return transitions
}

func (dfa *DFA[E]) RemoveState(state string) {
	_, ok := dfa.states[state]
	if !ok {
		// Gr8! We are done.
		return
	}

	// is this allowed?
	transitionsTo := dfa.AllTransitionsTo(state)

	if len(transitionsTo) > 0 {
		panic("can't remove state that is currently traversed to")
	}

	delete(dfa.states, state)
}

func (dfa *DFA[E]) AddState(state string, accepting bool) {
	if _, ok := dfa.states[state]; ok {
		// Gr8! We are done.
		return
	}

	newState := DFAState[E]{
		ordering:    dfa.order,
		name:        state,
		transitions: make(map[string]FATransition),
		accepting:   accepting,
	}
	dfa.order++

	if dfa.states == nil {
		dfa.states = map[string]DFAState[E]{}
	}

	dfa.states[state] = newState
}

func (dfa *DFA[E]) RemoveTransition(fromState string, input string, toState string) {
	curFromState, ok := dfa.states[fromState]
	if !ok {
		// Gr8! We are done.
		return
	}

	curTrans, ok := curFromState.transitions[input]
	if !ok {
		// Done early
		return
	}

	if curTrans.next != toState {
		// already not here
		return
	}

	// otherwise, remove the relation
	delete(curFromState.transitions, input)
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

// g must be non-augmented
func NewLALR1ViablePrefixDFA(g grammar.Grammar) (DFA[util.SVSet[grammar.LR1Item]], error) {
	lr1Dfa := NewLR1ViablePrefixDFA(g)

	// get an NFA so we can start fixing things
	lalrNfa := DFAToNFA(lr1Dfa)

	// counter for unique state name
	newStateNum := 0

	// now start merging states
	updated := true
	for updated {
		updated = false

		alreadyMerged := util.NewStringSet()
		states := lalrNfa.States()
		stateVals := map[string]util.SVSet[grammar.LR1Item]{}
		orderedStateElements := states.Elements()
		sort.Strings(orderedStateElements)
		for _, name := range orderedStateElements {
			stateVals[name] = lalrNfa.GetValue(name)
		}

		for _, stateName := range orderedStateElements {
			if alreadyMerged.Has(stateName) {
				continue
			}

			mergeWith := []string{}
			coreSet := grammar.CoreSet(stateVals[stateName])

			// need to find ALL to merge w or this is gonna get wild REEL quick
			for _, otherStateName := range orderedStateElements {
				if stateName == otherStateName {
					continue
				}

				otherCoreSet := grammar.CoreSet(stateVals[otherStateName])

				// Note: we do NOT enforce an ordering in general on which
				// states are merged first. this could cause issues; doing them
				// in an arbitrary order

				// check their cores
				if coreSet.Equal(otherCoreSet) {
					mergeWith = append(mergeWith, otherStateName)
				}
			}

			// now we merge any that have been queued to do so
			if len(mergeWith) > 0 {
				updated = true
				alreadyMerged.Add(stateName)
				destState := lalrNfa.states[stateName]
				mergedStateSet := util.NewSVSet(stateVals[stateName])

				for i := range mergeWith {
					alreadyMerged.Add(mergeWith[i])
					mergedStateSet.AddAll(stateVals[mergeWith[i]])
				}

				// We COULD tell what new name of state would be NOW, but to keep
				// things from overlapping during the process we will be setting
				// to a unique number and updating after all merges are complete
				// (at which point there should be 0 conflicting state names).
				newStateName := fmt.Sprintf("%d", newStateNum)
				newStateNum++
				destState.name = mergedStateSet.StringOrdered()
				destState.value = mergedStateSet

				// and so we can rewrite transitions from the old states to the
				// new one
				for i := range mergeWith {
					transitionsToMerged := lalrNfa.AllTransitionsTo(mergeWith[i])

					for j := range transitionsToMerged {
						trans := transitionsToMerged[j]
						from := trans.from
						sym := trans.input
						idx := trans.index

						// rewrite the transition to new state
						lalrNfa.states[from].transitions[sym][idx] = FATransition{input: sym, next: newStateName}
					}

					// also, check to see if we need to update start
					if lalrNfa.Start == mergeWith[i] {
						lalrNfa.Start = newStateName
					}
				}

				// also rewrite any transitions to the merged-to state
				transitionsToDestState := lalrNfa.AllTransitionsTo(stateName)
				for j := range transitionsToDestState {
					trans := transitionsToDestState[j]
					from := trans.from
					sym := trans.input
					idx := trans.index

					// rewrite the transition to new state
					lalrNfa.states[from].transitions[sym][idx] = FATransition{input: sym, next: newStateName}
				}

				// also, check to see if we need to update start
				if lalrNfa.Start == stateName {
					lalrNfa.Start = newStateName
				}

				// finally, enshore that any transitions we lose by deleting the
				// old state are added to the new state. this SHOULD collapse to
				// a single state by the time that things are done if it is
				// indeed an LALR(1) grammar
				for i := range mergeWith {
					lostTransitions := lalrNfa.states[mergeWith[i]].transitions
					for sym := range lostTransitions {
						transForSym := lostTransitions[sym]
						destTransForSym, ok := destState.transitions[sym]
						if !ok {
							destTransForSym = []FATransition{}
						}

						for j := range transForSym {
							// is this already in the dest? don't add it if so
							faTrans := transForSym[j]

							inDestTrans := false
							for k := range destTransForSym {
								destFATrans := destTransForSym[k]
								if destFATrans == faTrans {
									inDestTrans = true
									break
								}
							}
							if !inDestTrans {
								destTransForSym = append(destTransForSym, faTrans)
							}
						}
						destState.transitions[sym] = destTransForSym
					}
				}

				// with those updated, we can now delete the old states from
				// the DFA
				for i := range mergeWith {
					delete(lalrNfa.states, mergeWith[i])
				}

				// unshore if this condition is proven not to happen, either
				// way it's 8AD so checking
				if _, ok := lalrNfa.states[newStateName]; ok {
					panic(fmt.Sprintf("merged state name conflicts w state %q already in DFA", newStateName))
				}

				// enshore the updated new state is stored...
				lalrNfa.states[newStateName] = destState

				// ...and, finally, remove the old version of it
				delete(lalrNfa.states, stateName)
			}

			// did we just update? if so, all of the pre-cached info on states
			// and names and such is invalid due to modifying the DFA, and
			// therefore must be regenerated before checking anyfin else.
			//
			// they will be auto-regenerated by the parent loop
			if updated {
				break
			}
		}
	}

	// prior to conversion to dfa, go through and update the auto-numbered states
	lalrStates := lalrNfa.States().Elements()
	for _, stateName := range lalrStates {
		st := lalrNfa.states[stateName]

		// we keep the name pre-calculated in .name, so check if there's a mismatch
		if st.name != stateName {
			newStateName := st.name
			transitionsToMerged := lalrNfa.AllTransitionsTo(stateName)

			for j := range transitionsToMerged {
				trans := transitionsToMerged[j]
				from := trans.from
				sym := trans.input
				idx := trans.index

				// rewrite the transition to new state
				lalrNfa.states[from].transitions[sym][idx] = FATransition{input: sym, next: newStateName}
			}

			// also, check to see if we need to update start
			if lalrNfa.Start == stateName {
				lalrNfa.Start = newStateName
			}

			// and now, swap the name for the reel one
			lalrNfa.states[newStateName] = st
			delete(lalrNfa.states, stateName)
		}
	}

	lalrDfa, err := directNFAToDFA(lalrNfa)
	if err != nil {
		return DFA[util.SVSet[grammar.LR1Item]]{}, fmt.Errorf("grammar is not LALR(1); resulted in inconsistent state merges")
	}

	return lalrDfa, nil
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
