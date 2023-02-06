package parse

import (
	"fmt"
	"sort"

	"github.com/dekarrin/rosed"
	"github.com/dekarrin/tunaq/internal/ictiobus/automaton"
	"github.com/dekarrin/tunaq/internal/ictiobus/grammar"
	"github.com/dekarrin/tunaq/internal/util"
)

// GenerateCanonicalLR1Parser returns a parser that uses the set of canonical
// LR(1) items from g to parse input in language g. The provided language must
// be in LR(1) or else the a non-nil error is returned.
func GenerateCanonicalLR1Parser(g grammar.Grammar) (lrParser, error) {
	table, err := constructCanonicalLR1ParseTable(g)
	if err != nil {
		return lrParser{}, err
	}

	return lrParser{table: table}, nil
}

// constructCanonicalLR1ParseTable constructs the canonical LR(1) table for G.
// It augments grammar G to produce G', then the canonical collection of sets of
// LR(1) items of G' is used to construct a table with applicable GOTO and
// ACTION columns.
//
// This is an implementation of Algorithm 4.56, "Construction of canonical-LR
// parsing tables", from the purple dragon book. In the comments, most of which
// is lifted directly from the textbook, GOTO[i, A] refers to the vaue of the
// table's GOTO column at state i, symbol A, while GOTO(i, A) refers to the
// "precomputed GOTO function for grammar G'".
func constructCanonicalLR1ParseTable(g grammar.Grammar) (LRParseTable, error) {
	// we will skip a few steps here and simply grab the LR0 DFA for G' which
	// will pretty immediately give us our GOTO() function, since as purple
	// dragon book mentions, "intuitively, the GOTO function is used to define
	// the transitions in the LR(0) automaton for a grammar."
	lr1Automaton := automaton.NewLR1ViablePrefixDFA(g)

	table := &canonicalLR1Table{
		gPrime:    g.Augmented(),
		gStart:    g.StartSymbol(),
		gTerms:    g.Terminals(),
		gNonTerms: g.NonTerminals(),
		lr1:       lr1Automaton,
		itemCache: map[string]grammar.LR1Item{},
	}

	// collect item cache from the states of our lr1 DFA
	allStates := util.OrderedKeys(table.lr1.States())
	for _, dfaStateName := range allStates {
		itemSet := table.lr1.GetValue(dfaStateName)
		for k := range itemSet {
			table.itemCache[k] = itemSet[k]
		}
	}

	// check that we dont hit conflicts in ACTION
	for i := range lr1Automaton.States() {
		for _, a := range table.gPrime.Terminals() {
			itemSet := table.lr1.GetValue(i)
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
							return nil, fmt.Errorf("grammar is not LR(1): found both %s and %s actions for input %q", act.String(), newAct.String(), a)
						}
						act = newAct
						matchFound = true
					}
				}

				if len(beta) == 0 && A != table.gPrime.StartSymbol() && a == b {
					newAct := LRAction{Type: LRReduce, Symbol: A, Production: grammar.Production(alpha)}
					if matchFound && !newAct.Equal(act) {
						return nil, fmt.Errorf("grammar is not LR(1): found both %s and %s actions for input %q", act.String(), newAct.String(), a)
					}
					act = newAct
					matchFound = true
				}

				if a == "$" && b == "$" && A == table.gPrime.StartSymbol() && len(alpha) == 1 && alpha[0] == table.gStart && len(beta) == 0 {
					newAct := LRAction{Type: LRAccept}
					if matchFound && !newAct.Equal(act) {
						return nil, fmt.Errorf("grammar is not LR(1): found both %s and %s actions for input %q", act.String(), newAct.String(), a)
					}
					act = newAct
					matchFound = true
				}
			}
		}
	}

	return table, nil
}

type canonicalLR1Table struct {
	gPrime    grammar.Grammar
	gStart    string
	lr1       automaton.DFA[util.BSet[string, grammar.LR1Item]]
	itemCache map[string]grammar.LR1Item
	gTerms    []string
	gNonTerms []string
}

func (clr1 *canonicalLR1Table) String() string {
	// need mapping of state to indexes
	stateRefs := map[string]string{}

	// need to gaurantee order
	stateNames := clr1.lr1.States().Slice()
	sort.Strings(stateNames)

	// put the initial state first
	for i := range stateNames {
		if stateNames[i] == clr1.lr1.Start {
			old := stateNames[0]
			stateNames[0] = stateNames[i]
			stateNames[i] = old
			break
		}
	}
	for i := range stateNames {
		stateRefs[stateNames[i]] = fmt.Sprintf("%d", i)
	}

	allTerms := make([]string, len(clr1.gTerms))
	copy(allTerms, clr1.gTerms)
	allTerms = append(allTerms, "$")

	// okay now do data setup
	data := [][]string{}

	// set up the headers
	headers := []string{"S", "|"}

	for _, t := range allTerms {
		headers = append(headers, fmt.Sprintf("A:%s", t))
	}

	headers = append(headers, "|")

	for _, nt := range clr1.gNonTerms {
		headers = append(headers, fmt.Sprintf("G:%s", nt))
	}
	data = append(data, headers)

	// now need to do each state
	for stateIdx := range stateNames {
		i := stateNames[stateIdx]
		row := []string{stateRefs[i], "|"}

		for _, t := range allTerms {
			act := clr1.Action(i, t)

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

		for _, nt := range clr1.gNonTerms {
			var cell = ""

			gotoState, err := clr1.Goto(i, nt)
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

func (clr1 *canonicalLR1Table) Initial() string {
	return clr1.lr1.Start
}

func (clr1 *canonicalLR1Table) Goto(state, symbol string) (string, error) {
	// step 3 of algorithm 4.56, "Construction of canonical-LR parsing tables",
	// for reference:

	// 3. The goto transitions for state i are constructed for all nonterminals
	// A using the rule: If GOTO(Iᵢ, A) = Iⱼ, then GOTO[i, A] = j.
	newState := clr1.lr1.Next(state, symbol)
	if newState == "" {
		return "", fmt.Errorf("GOTO[%q, %q] is an error entry", state, symbol)
	}
	return newState, nil
}

func (clr1 *canonicalLR1Table) Action(i, a string) LRAction {
	// step 2 of algorithm 4.56, "Construction of canonical-LR parsing tables",
	// for reference:

	// 2. State i is constructed from Iᵢ. The parsing actions for state i are
	// determined as follows:

	// (a) If [A -> α.aβ, b] is in Iᵢ and GOTO(Iᵢ, a) = Iⱼ, then set
	// ACTION[i, a] to "shift j." Here a must be a terminal.

	// (b) If [A -> α., a] is in Iᵢ, A != S', then set ACTION[i, a] to "reduce
	// A -> α".

	// get our set back from current state so we can check it; this is our Iᵢ
	itemSet := clr1.lr1.GetValue(i)

	// we have gauranteed that these dont conflict during construction; still,
	// check it so we can panic if it conflicts
	var alreadySet bool
	var act LRAction

	// Okay, "[some random item] is in Iᵢ" is suuuuuuuuper vague. We're
	// basically going to have to check each item and see if it is in the
	// pattern. I *guess* ::::/
	for itemStr := range itemSet {
		item := clr1.itemCache[itemStr]

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
		if clr1.gPrime.IsTerminal(a) && len(beta) > 0 && beta[0] == a {
			j, err := clr1.Goto(i, a)

			// it's okay if we get an error; it just means there is no
			// transition defined (i think, glub, the purple dragon book's
			// method of constructing GOTO would have it returning an empty
			// set in this case but unshore), so it is not a match.
			if err == nil {
				// match found
				newAct := LRAction{Type: LRShift, State: j}
				if alreadySet && !newAct.Equal(act) {
					panic(fmt.Sprintf("grammar is not LR(1): found both %s and %s actions for input %q", act.String(), newAct.String(), a))
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
		if len(beta) == 0 && A != clr1.gPrime.StartSymbol() && a == b {
			newAct := LRAction{Type: LRReduce, Symbol: A, Production: grammar.Production(alpha)}
			if alreadySet && !newAct.Equal(act) {
				panic(fmt.Sprintf("grammar is not LR(1): found both %s and %s actions for input %q", act.String(), newAct.String(), a))
			}
			act = newAct
			alreadySet = true
		}

		// (c) If [S' -> S., $] is in Iᵢ, then set ACTION[i, $] to "accept".
		if a == "$" && b == "$" && A == clr1.gPrime.StartSymbol() && len(alpha) == 1 && alpha[0] == clr1.gStart && len(beta) == 0 {
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
