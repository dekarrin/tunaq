package parse

import (
	"fmt"
	"sort"

	"github.com/dekarrin/rosed"
	"github.com/dekarrin/tunaq/internal/ictiobus/automaton"
	"github.com/dekarrin/tunaq/internal/ictiobus/grammar"
	"github.com/dekarrin/tunaq/internal/util"
)

// computeLALR1Kernels computes LALR(1) kernels for grammar g, which must NOT be
// an augmented grammar.
//
// This is an implementation of Algorithm 4.63, "Efficient computation of the
// kernels of the LALR(1) collection of sets of items" from purple dragon book.
func computeLALR1Kernels(g grammar.Grammar) util.SVSet[util.SVSet[grammar.LR1Item]] {
	// we'll also need to know what our start rule and augmented start rules are.
	startSym := g.StartSymbol()
	startSymPrime := g.Augmented().StartSymbol()
	gPrimeStartItem := grammar.LR0Item{NonTerminal: startSymPrime, Right: []string{startSym}}
	gPrimeStartKernel := util.NewSVSet[grammar.LR0Item]()
	gPrimeStartKernel.Set(gPrimeStartItem.String(), gPrimeStartItem)

	gTerminals := g.Terminals()

	// 1. Construct the kernels of the sets of LR(O) items for G.
	lr0Kernels := getLR0Kernels(g)

	calcSponts := map[stateAndItemStr]util.StringSet{}
	calcProps := map[stateAndItemStr][]stateAndItemStr{}

	// special case, lookahead $ is always generated spontaneously for the item
	// S' -> .S in the initial set of items
	calcSponts[stateAndItemStr{state: gPrimeStartKernel.String(), item: gPrimeStartItem.String()}] = util.StringSetOf([]string{"$"})

	for _, lr0KernelName := range lr0Kernels.Elements() {
		IKernelSet := lr0Kernels.Get(lr0KernelName)

		if IKernelSet.Equal(util.StringSetOf([]string{"S-P -> . S"})) {
			fmt.Printf("make debugger do thing\n")
		}

		for _, X := range gTerminals {
			// 2. Apply algorithm 4.62 to the kernel of set of LR(0) items and
			// grammar symbol X to determine which lookaheads are spontaneously
			// generated for kernel items in GOTO(I, X), and from which items in
			// I lookaheads are propagated to kernel items in GOTO(I, X).
			sponts, props := determineLookaheads(g.Augmented(), IKernelSet, X)

			// add them to our pre-calced slice for later use in lookahead
			// table
			for k := range sponts {
				sponSet := sponts[k]
				existing, ok := calcSponts[k]
				if !ok {
					existing = util.NewStringSet()
				}
				existing.AddAll(sponSet)
				calcSponts[k] = existing
			}
			for k := range props {
				propSlice := props[k]
				existing, ok := calcProps[k]
				if !ok {
					existing = make([]stateAndItemStr, 0)
				}
				for i := range propSlice {
					existing = append(existing, propSlice[i])
				}
				calcProps[k] = existing
			}
		}
	}

	// 3. Initialize a table that gives, for each kernel item in each set of
	// items, the associated lookaheads. Initially, each item has associated
	// with it only those lookaheads that we determined in step (2) were
	// generated spontaneously

	// this table holds a slice of passes, each of which map a
	// {LR0Item}.OrderedString() to a slice of passes. Each pass is a
	// slice of the lookaheads found on that pass. Pass 0, aka "INIT" pass in
	// purple dragon book, is the spontaneously generated lookaheads for the
	// item; all other passes are the propagation checks.
	lookaheadCalcTable := []map[stateAndItemStr]util.StringSet{}
	initPass := map[stateAndItemStr]util.StringSet{}
	for k := range calcSponts {
		sponts := calcSponts[k]
		elemSet := util.NewStringSet()
		for _, terminal := range sponts.Elements() {
			elemSet.Add(terminal)
		}
		initPass[k] = elemSet
	}
	lookaheadCalcTable = append(lookaheadCalcTable, initPass)

	/*
		// 4. Make repeated passes over the kernel items in all sets. When we visit
		// an item i, we look up the kernel items to which i propagates its
		// lookaheads, using information tabulated in step (2). The current set of
		// lookaheads for i is added to those already associated with each of the
		// items to which i propagates its lookaheads. We continue making passes
		// over the kernel items until no more new lookaheads are propagated.
		updated := true
		passNum := 1
		for updated {
			updated = false

			prevColumn := lookaheadCalcTable[passNum-1]
			curColumn := map[stateAndItemStr]util.StringSet{}

			// initialy set everyfin to prior column
			for k := range prevColumn {
				curColumn[k] = util.NewStringSet(prevColumn[k])
			}

			for _, lr0KernelName := range lr0Kernels.Elements() {
				IKernelSet := lr0Kernels.Get(lr0KernelName)
				// When we visit an item i, we look up the kernel items to which i
				// propagates its lookaheads, using information tabulated in step
				// (2).
				propagateTo := calcProps[IKernelSet.StringOrdered()]

				// The current set of lookaheads for i is added to those already
				// associated with each of the items to which i propagates its
				// lookaheads.
				curLookaheads := prevColumn[IKernelSet.StringOrdered()]
				for _, toName := range propagateTo.Elements() {
					for _, la := range curLookaheads.Elements() {
						if !curColumn[toName].Has(la) {
							propDest := curColumn[toName]
							propDest.Add(la)
							curColumn[toName] = propDest
							updated = true
						}
					}
				}
			}

			lookaheadCalcTable = append(lookaheadCalcTable, curColumn)
			passNum++
		}*/

	// now collect the final table info into the final result
	//finalPass := lookaheadCalcTable[len(lookaheadCalcTable)-1]
	lalrKernels := util.NewSVSet[util.SVSet[grammar.LR1Item]]()

	// TODO: actually convert the table results to this.
	return lalrKernels

}

type stateAndItemStr struct {
	state string
	item  string
}

// determineLookaheads finds the lookaheads spontaneously generated by items in
// I for kernel items in GOTO(I, X) (jello: g.LR1_GOTO) and the items in I from
// which lookaheads are propagated to kernel items in GOTO(I, X).
//
// g must be an augmented grammar.
// K is the kernel of a set of LR(0) items I. X is a grammar symbol. Returns the
// LALR(1) kernel set generated from the LR(0) item kernel set.
//
// This is an implementation of Algorithm 4.62, "Determining lookaheads", from
// purple dragon book.
//
// "There are two ways a lookahead b can get attached to an LR(0) item
// [B -> γ.δ] in some set of LALR(1) items J:"
//
// 1. There is a set of items I, with a kernel item [A -> α.β, a], and J =
// GOTO(I, X), and the construction of
//
//	GOTO(CLOSURE({[A -> α.β, a]}), X)
//
// as given in Fig. 4.40 (jello: implemented in g.LR1_CLOSURE and
// g.LR1_GOTO), contains [B -> γ.δ, b], regardless of a. Such a lookahead is
// said to be generated *spontaneously* for B -> γ.δ.
//
// 2. As a special case, lookahead $ is generated spontaneously for the item
// [S' -> .S] in the initial set of items.
//
// 3. All as (1), but a = b, and GOTO(CLOSURE({[A -> α.β, b]}), X), as given
// in Fig. 4.40 (jello: again, g.LR1_CLOSURE and g.LR1_GOTO), contains
// [B -> γ.δ, b] only because A -> α.β has b as one of its associated
// lookaheads. In such a case, we say that lookaheads *propagate* from
// A -> α.β in the kernel of I to B -> γ.δ in the kernel of J. Note that
// propagation does not depend on the particular lookahead symbol; either
// all lookaheads propagate from one item to another, or none do.
func determineLookaheads(g grammar.Grammar, K util.SVSet[grammar.LR0Item], X string) (spontaneous map[stateAndItemStr]util.StringSet, propagated map[stateAndItemStr][]stateAndItemStr) {
	// note: '#' in notes stands for any symbol not in the grammar at hand. We
	// will use Grammar.GenerateUniqueName to get one not currently used, and as
	// we require g to be augmented, this should give us somefin OTHER than the
	// added start production.
	nonGrammarSym := g.GenerateUniqueTerminal("#")

	if K.Equal(util.StringSetOf([]string{"S-P -> . S"})) {
		fmt.Printf("make debugger do thing\n")
	}

	spontaneous = map[stateAndItemStr]util.StringSet{}
	propagated = map[stateAndItemStr][]stateAndItemStr{}

	// GOTO will be needed elsewhere
	GOTO_I_X := g.LR0_GOTO(g.LR0_CLOSURE(K), X)

	if GOTO_I_X.Empty() {
		return spontaneous, propagated
	}

	// for ( each item A -> α.β in K ) {
	for _, aItemName := range K.Elements() {
		aItem := K.Get(aItemName)

		// J := CLOSURE({[A -> α.β, #]})
		lr1StartItem := grammar.LR1Item{LR0Item: aItem, Lookahead: nonGrammarSym}
		lr1StartKernels := util.NewSVSet[grammar.LR1Item]()
		lr1StartKernels.Set(lr1StartItem.String(), lr1StartItem)
		J := g.LR1_CLOSURE(lr1StartKernels)

		TRUE_GOTO_I_X := g.LR1_GOTO(J, X)

		// next parts tell us to check condition based on some lookahead in
		// [B -> γ.Xδ, a] of J ...soooooooo in other words, check all of the
		// items in J
		for _, bItemName := range J.Elements() {
			bItem := J.Get(bItemName)

			newLeft := make([]string, len(bItem.Left))
			copy(newLeft, bItem.Left)

			var newRight []string
			if len(bItem.Right) > 0 {
				newRight = make([]string, len(bItem.Right)-1)
				copy(newRight, bItem.Right[1:])
				newLeft = append(newLeft, bItem.Right[0])
			}

			// shifted item is our [B -> γX.δ]. note that the dot has moved one
			// symbol to the right
			shiftedLR0Item := grammar.LR0Item{
				NonTerminal: bItem.NonTerminal,
				Left:        newLeft,
				Right:       newRight,
			}

			// slightly more complex logic to go through all of TRUE_GOTO
			// and find all items that have the same LR0 as our shifted one
			prodInGoto := false
			for _, elemName := range TRUE_GOTO_I_X.Elements() {
				lr1Item := TRUE_GOTO_I_X.Get(elemName)
				if lr1Item.LR0Item.Equal(shiftedLR0Item) {
					prodInGoto = true
					break
				}
			}
			if !prodInGoto {
				shiftedItemStr := shiftedLR0Item.String()
				fmt.Println(shiftedItemStr)
				continue
			}

			if bItem.Lookahead != nonGrammarSym {
				// if ( [B -> γ.Xδ, a] is in J, and a is not # )

				// conclude that lookahead a is spontaneously generated for item
				// B -> γX.δ in GOTO(I, X).
				newItem := grammar.LR1Item{
					LR0Item:   shiftedLR0Item,
					Lookahead: bItem.Lookahead,
				}

				key := stateAndItemStr{
					state: GOTO_I_X.StringOrdered(),
					item:  newItem.LR0Item.String(),
				}

				spontSet, ok := spontaneous[key]
				if !ok {
					spontSet = util.NewStringSet()
				}
				spontSet.Add(bItem.Lookahead)

				spontaneous[key] = spontSet
			} else {
				// if ( [B -> γ.Xδ, #] is in J )

				// conclude that lookaheads propagate from A -> α.β in I to
				// B -> γX.δ in GOTO(I, X).

				from := stateAndItemStr{
					state: K.StringOrdered(),
					item:  aItem.String(),
				}

				to := stateAndItemStr{
					state: GOTO_I_X.StringOrdered(),
					item:  shiftedLR0Item.String(),
				}

				existingPropagated, ok := propagated[from]
				if !ok {
					existingPropagated = []stateAndItemStr{}
				}
				existingPropagated = append(existingPropagated, to)
				propagated[from] = existingPropagated
			}

		}
	}

	return spontaneous, propagated
}

// g must NOT be an augmented grammar.
func getLR0Kernels(g grammar.Grammar) util.VSet[string, util.SVSet[grammar.LR0Item]] {
	gPrime := g.Augmented()
	itemSets := gPrime.CanonicalLR0Items()

	kernels := util.SVSet[util.SVSet[grammar.LR0Item]]{}

	// okay, now for each state pull out the kernels
	for _, s := range itemSets.Elements() {
		stateVal := itemSets.Get(s)

		kernelItems := util.SVSet[grammar.LR0Item]{}
		for _, stateItemName := range stateVal.Elements() {
			stateItem := stateVal.Get(stateItemName)
			if len(stateItem.Left) > 0 || (len(stateItem.Right) == 1 && stateItem.Right[0] == g.StartSymbol() && stateItem.NonTerminal == gPrime.StartSymbol()) {
				kernelItems.Set(stateItemName, stateItem)
			}
		}
		kernels.Set(kernelItems.StringOrdered(), kernelItems)
	}

	return kernels
}

// constructLALR1ParseTable constructs the LALR(1) table for G.
// It augments grammar G to produce G', then the canonical collection of sets of
// LR(1) items of G' is used to construct a table with applicable GOTO and
// ACTION columns.
//
// This is an implementation of Algorithm 4.59, "An easy, but space-consuming
// LALR table construction", from the purple dragon book. In the comments, most
// of which is lifted directly from the textbook, GOTO[i, A] refers to the vaue
// of the table's GOTO column at state i, symbol A, while GOTO(i, A) refers to
// the "precomputed GOTO function for grammar G'".
func constructLALR1ParseTable(g grammar.Grammar) (LRParseTable, error) {
	dfa, _ := automaton.NewLALR1ViablePrefixDFA(g)

	table := &lalr1Table{
		gPrime:    g.Augmented(),
		gTerms:    g.Terminals(),
		gStart:    g.StartSymbol(),
		gNonTerms: g.NonTerminals(),
		dfa:       dfa,
		itemCache: map[string]grammar.LR1Item{},
	}

	// collect item cache from the states of our lr1 DFA
	allStates := util.OrderedKeys(table.dfa.States())
	for _, dfaStateName := range allStates {
		itemSet := table.dfa.GetValue(dfaStateName)
		for k := range itemSet {
			table.itemCache[k] = itemSet[k]
		}
	}

	// check that we dont hit conflicts in ACTION
	for i := range dfa.States() {
		for _, a := range table.gPrime.Terminals() {
			itemSet := table.dfa.GetValue(i)
			var matchFound bool
			var act LRAction
			for itemStr := range itemSet {
				item := table.itemCache[itemStr]
				A := item.NonTerminal
				alpha := item.Left
				beta := item.Right
				b := item.Lookahead
				if table.gPrime.IsTerminal(a) && len(beta) > 0 && beta[0] == a {
					j, err := table.Goto(i, a)
					if err == nil {
						// match found
						newAct := LRAction{Type: LRShift, State: j}
						if matchFound && !newAct.Equal(act) {
							return nil, fmt.Errorf("grammar is not LALR(1): found both %s and %s actions for input %q", act.String(), newAct.String(), a)
						}
						act = newAct
						matchFound = true
					}
				}

				if len(beta) == 0 && A != table.gPrime.StartSymbol() && a == b {
					newAct := LRAction{Type: LRReduce, Symbol: A, Production: grammar.Production(alpha)}
					if matchFound && !newAct.Equal(act) {
						return nil, fmt.Errorf("grammar is not LALR(1): found both %s and %s actions for input %q", act.String(), newAct.String(), a)
					}
					act = newAct
					matchFound = true
				}

				if a == "$" && b == "$" && A == table.gPrime.StartSymbol() && len(alpha) == 1 && alpha[0] == table.gStart && len(beta) == 0 {
					newAct := LRAction{Type: LRAccept}
					if matchFound && !newAct.Equal(act) {
						return nil, fmt.Errorf("grammar is not LALR(1): found both %s and %s actions for input %q", act.String(), newAct.String(), a)
					}
					act = newAct
					matchFound = true
				}
			}
		}
	}

	return table, nil
}

type lalr1Table struct {
	gPrime    grammar.Grammar
	gStart    string
	dfa       automaton.DFA[util.SVSet[grammar.LR1Item]]
	itemCache map[string]grammar.LR1Item
	gTerms    []string
	gNonTerms []string
}

func (lalr1 *lalr1Table) Action(i, a string) LRAction {
	// Algorithm 4.59, which we are using for construction of the LALR(1) parse
	// table, explicitly mentions to construct the Action table as it is done
	// in Algorithm 4.56.

	// step 2 of algorithm 4.56, "Construction of canonical-LR parsing tables",
	// for reference:

	// 2. State i is constructed from Iᵢ. The parsing actions for state i are
	// determined as follows:

	// (a) If [A -> α.aβ, b] is in Iᵢ and GOTO(Iᵢ, a) = Iⱼ, then set
	// ACTION[i, a] to "shift j." Here a must be a terminal.

	// (b) If [A -> α., a] is in Iᵢ, A != S', then set ACTION[i, a] to "reduce
	// A -> α".

	// get our set back from current state so we can check it; this is our Iᵢ

	// get our set back from current state so we can check it; this is our Iᵢ
	itemSet := lalr1.dfa.GetValue(i)

	// we have gauranteed that these dont conflict during construction; still,
	// check it so we can panic if it conflicts
	var alreadySet bool
	var act LRAction

	// Okay, "[some random item] is in Iᵢ" is suuuuuuuuper vague. We're
	// basically going to have to check each item and see if it is in the
	// pattern. I *guess* ::::/
	for itemStr := range itemSet {
		item := lalr1.itemCache[itemStr]

		// given item is [A -> α.β, b]:
		A := item.NonTerminal
		alpha := item.Left
		beta := item.Right
		b := item.Lookahead

		// (a) If [A -> α.aβ, b] is in Iᵢ and GOTO(Iᵢ, a) = Iⱼ, then set
		// ACTION[i, a] to "shift j." Here a must be a terminal.
		//
		// we'll assume α can be ε.
		// β can also be ε but note this β is rly β[1:] from earlier notation
		// used to assign beta (beta := item.Right).
		if lalr1.gPrime.IsTerminal(a) && len(beta) > 0 && beta[0] == a {
			j, err := lalr1.Goto(i, a)

			// it's okay if we get an error; it just means there is no
			// transition defined (i think, glub, the purple dragon book's
			// method of constructing GOTO would have it returning an empty
			// set in this case but unshore), so it is not a match.
			if err == nil {
				// match found
				newAct := LRAction{Type: LRShift, State: j}
				if alreadySet && !newAct.Equal(act) {
					panic(fmt.Sprintf("grammar is not LALR(1): found both %s and %s actions for input %q", act.String(), newAct.String(), a))
				}
				act = newAct
				alreadySet = true
			}
		}

		// (b) If [A -> α., a] is in Iᵢ, A != S', then set ACTION[i, a] to
		// "reduce A -> α".
		//
		// we'll assume α can be empty.
		// the beta we previously retrieved MUST be empty.
		// further, lookahead b MUST be a.
		if len(beta) == 0 && A != lalr1.gPrime.StartSymbol() && a == b {
			newAct := LRAction{Type: LRReduce, Symbol: A, Production: grammar.Production(alpha)}
			if alreadySet && !newAct.Equal(act) {
				panic(fmt.Sprintf("grammar is not LR(1): found both %s and %s actions for input %q", act.String(), newAct.String(), a))
			}
			act = newAct
			alreadySet = true
		}

		// (c) If [S' -> S., $] is in Iᵢ, then set ACTION[i, $] to "accept".
		if a == "$" && b == "$" && A == lalr1.gPrime.StartSymbol() && len(alpha) == 1 && alpha[0] == lalr1.gStart && len(beta) == 0 {
			newAct := LRAction{Type: LRAccept}
			if alreadySet && !newAct.Equal(act) {
				panic(fmt.Sprintf("grammar is not LR(1): found both %s and %s actions for input %q", act.String(), newAct.String(), a))
			}
			act = newAct
			alreadySet = true
		}
	}

	// if we haven't found one, error
	if !alreadySet {
		act.Type = LRError
	}

	return act
}

func (lalr1 *lalr1Table) Goto(state, symbol string) (string, error) {
	newState := lalr1.dfa.Next(state, symbol)
	if newState == "" {
		return "", fmt.Errorf("GOTO[%q, %q] is an error entry", state, symbol)
	}
	return newState, nil
}

func (lalr1 *lalr1Table) Initial() string {
	return lalr1.dfa.Start
}

func (lalr1 *lalr1Table) String() string {
	// need mapping of state to indexes
	stateRefs := map[string]string{}

	// need to gaurantee order
	stateNames := lalr1.dfa.States().Elements()
	sort.Strings(stateNames)

	// put the initial state first
	for i := range stateNames {
		if stateNames[i] == lalr1.dfa.Start {
			old := stateNames[0]
			stateNames[0] = stateNames[i]
			stateNames[i] = old
			break
		}
	}
	for i := range stateNames {
		stateRefs[stateNames[i]] = fmt.Sprintf("%d", i)
	}

	allTerms := make([]string, len(lalr1.gTerms))
	copy(allTerms, lalr1.gTerms)
	allTerms = append(allTerms, "$")

	// okay now do data setup
	data := [][]string{}

	// set up the headers
	headers := []string{"S", "|"}

	for _, t := range allTerms {
		headers = append(headers, fmt.Sprintf("A:%s", t))
	}

	headers = append(headers, "|")

	for _, nt := range lalr1.gNonTerms {
		headers = append(headers, fmt.Sprintf("G:%s", nt))
	}
	data = append(data, headers)

	// now need to do each state
	for stateIdx := range stateNames {
		i := stateNames[stateIdx]
		row := []string{stateRefs[i], "|"}

		for _, t := range allTerms {
			act := lalr1.Action(i, t)

			cell := ""
			switch act.Type {
			case LRAccept:
				cell = "acc"
			case LRReduce:
				// reduces to the state that corresponds with the symbol
				cell = fmt.Sprintf("r%s -> %s", act.Symbol, act.Production.String())
			case LRShift:
				cell = fmt.Sprintf("s%s", stateRefs[act.State])
			case LRError:
				// do nothing, err is blank
			}

			row = append(row, cell)
		}

		row = append(row, "|")

		for _, nt := range lalr1.gNonTerms {
			var cell = ""

			gotoState, err := lalr1.Goto(i, nt)
			if err == nil {
				cell = stateRefs[gotoState]
			}

			row = append(row, cell)
		}

		data = append(data, row)
	}

	// This used to be 120 width. Glu88in' *8et* on that. lol.
	return rosed.
		Edit("").
		InsertTableOpts(0, data, 10, rosed.Options{
			TableHeaders:             true,
			NoTrailingLineSeparators: true,
		}).
		String()
}
