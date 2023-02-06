package tunascript

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dekarrin/rosed"
	"github.com/dekarrin/tunaq/internal/util"
)

// Parse builds an abstract syntax tree by reading the tokens in the provided
// tokenStream. It returns the built up AST that is parsed from it. If any
// issues are encountered, an error is returned (likely a SyntaxError).
func Parse(tokens tokenStream) (AST, error) {
	ast, err := parseExpression(&tokens, 0)
	if err != nil {
		return AST{}, err
	}

	fullTree := AST{
		nodes: []*astNode{ast},
	}

	return fullTree, nil
}

func parseExpression(stream *tokenStream, rbp int) (*astNode, error) {
	// TODO: consider implementing panic mode to parse rest of the system

	var err error

	if stream.Remaining() < 1 {
		return nil, fmt.Errorf("no tokens to parse")
	}

	t := stream.Next()
	left, err := t.nud(stream)
	if err != nil {
		return nil, err
	}
	if left == nil {
		return nil, syntaxErrorFromLexeme(fmt.Sprintf("unexpected %[1]s\n(%[1]s cannot be at the start of an expression)", t.class.human), t)
	}

	for rbp < stream.Peek().class.lbp {
		t = stream.Next()
		left, err = t.led(left, stream)
		if err != nil {
			return nil, err
		}
	}
	return left, nil

}

type parseTree struct {
	terminal bool
	value    string
	source   token
	children []*parseTree
}

// String returns a prettified representation of the entire parse tree suitable
// for use in line-by-line comparisons of tree structure. Two parse trees are
// considered semantcally identical if they produce identical String() output.
func (pt parseTree) String() string {
	return pt.leveledStr("", "")
}

func (pt parseTree) leveledStr(firstPrefix, contPrefix string) string {
	var sb strings.Builder

	sb.WriteString(firstPrefix)
	if pt.terminal {
		sb.WriteString(fmt.Sprintf("(TERM %q)", pt.value))
	} else {
		sb.WriteString(fmt.Sprintf("( %s )", pt.value))
	}

	for i := range pt.children {
		sb.WriteRune('\n')
		var leveledFirstPrefix string
		var leveledContPrefix string
		if i+1 < len(pt.children) {
			leveledFirstPrefix = contPrefix + makeASTTreeLevelPrefix("")
			leveledContPrefix = contPrefix + astTreeLevelOngoing
		} else {
			leveledFirstPrefix = contPrefix + makeASTTreeLevelPrefixLast("")
			leveledContPrefix = contPrefix + astTreeLevelEmpty
		}
		itemOut := pt.children[i].leveledStr(leveledFirstPrefix, leveledContPrefix)
		sb.WriteString(itemOut)
	}

	return sb.String()
}

// Equal returns whether the parseTree is equal to the given object. If the
// given object is not a parseTree, returns false, else returns whether the two
// parse trees have the exact same structure.
func (pt parseTree) Equal(o any) bool {
	other, ok := o.(parseTree)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*parseTree)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if pt.terminal != other.terminal {
		return false
	} else if pt.value != other.value {
		return false
	} else {
		// check every sub tree
		if len(pt.children) != len(other.children) {
			return false
		}

		for i := range pt.children {
			if !pt.children[i].Equal(other.children[i]) {
				return false
			}
		}
	}
	return true
}

// LL1PredictiveParse runse a parse of the input using LL(k) parsing rules on
// the context-free Grammar g (k=1). The grammar must be LL(1); it will not be
// forced to it.
func LL1PredictiveParse(g Grammar, stream tokenStream) (pt parseTree, err error) {
	M, err := g.LLParseTable()
	if err != nil {
		return pt, err
	}

	stack := util.Stack[string]{Of: []string{g.StartSymbol(), "$"}}
	next := stream.Peek()
	X := stack.Peek()
	pt = parseTree{value: g.StartSymbol()}
	ptStack := util.Stack[*parseTree]{Of: []*parseTree{&pt}}

	node := ptStack.Peek()
	for X != "$" { /* stack is not empty */
		if strings.ToLower(X) == X {
			stream.Next()

			// is terminals
			t := g.Term(X)
			if next.class.Equal(t) {
				node.terminal = true
				node.source = next
				stack.Pop()
				X = stack.Peek()
				ptStack.Pop()
				node = ptStack.Peek()
			} else {
				return pt, syntaxErrorFromLexeme(fmt.Sprintf("There should be a %s here, but it was %q!", t.human, next.lexeme), next)
			}

			next = stream.Peek()
		} else {
			nextProd := M.Get(X, g.TermFor(next.class))
			if nextProd.Equal(Error) {
				return pt, syntaxErrorFromLexeme(fmt.Sprintf("It doesn't make any sense to put a %q here!", next.class.human), next)
			}

			stack.Pop()
			ptStack.Pop()
			for i := len(nextProd) - 1; i >= 0; i-- {
				if nextProd[i] != Epsilon[0] {
					stack.Push(nextProd[i])
				}

				child := &parseTree{value: nextProd[i]}
				if nextProd[i] == Epsilon[0] {
					child.terminal = true
				}
				node.children = append([]*parseTree{child}, node.children...)

				if nextProd[i] != Epsilon[0] {
					ptStack.Push(child)
				}
			}

			X = stack.Peek()

			// node stack will always be one smaller than symbol stack bc
			// glub, we dont put a node onto the stack for "$".
			if X != "$" {
				node = ptStack.Peek()
			}
		}
	}

	return pt, nil
}

type LRActionType int

const (
	LRShift LRActionType = iota
	LRReduce
	LRAccept
	LRError
)

type LRAction struct {
	Type LRActionType

	// Production is used when Type is LRReduce. It is the production which
	// should be reduced; the β of A -> β.
	Production Production

	// Symbol is used when Type is LRReduce. It is the symbol to reduce the
	// production to; the A of A -> β.
	Symbol string

	// State is the state to shift to. It is used only when Type is LRShift.
	State string
}

func (act LRAction) String() string {
	switch act.Type {
	case LRAccept:
		return "ACTION<accept>"
	case LRError:
		return "ACTION<error>"
	case LRReduce:
		return fmt.Sprintf("ACTION<reduce %s -> %s>", act.Symbol, act.Production.String())
	case LRShift:
		return fmt.Sprintf("ACTION<shift %s>", act.State)
	default:
		return "ACTION<unknown>"
	}
}

func (act LRAction) Equal(o any) bool {
	other, ok := o.(LRAction)
	if !ok {
		otherPtr := o.(*LRAction)
		if !ok {
			return false
		}
		if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if act.Type != other.Type {
		return false
	} else if !act.Production.Equal(other.Production) {
		return false
	} else if act.State != other.State {
		return false
	} else if act.Symbol != other.Symbol {
		return false
	}

	return true
}

// LRParseTable is a table of information passed to an LR parser. These will be
// generated from a grammar for the purposes of performing bottom-up parsing.
type LRParseTable interface {
	// Shift reads one token of input. For SR parsers that are implemented with
	// a stack, this will push a terminal onto the stack.
	//
	// ABC|xyz => ABCx|yz
	//Shift()

	// Reduce applies an inverse production at the right end of the left string.
	// For SR parsers that are implemented with a stack, this will pop 0 or more
	// terminals off of the stack (production rhs), then will push a
	// non-terminal onto the stack (production lhs).
	//
	// Given A -> xy is a production, then:
	// Cbxy|ijk => CbA|ijk
	//Reduce()

	// Initial returns the initial state of the parse table, if that is
	// applicable for the table.
	Initial() string

	// Action gets the next action to take based on a state i and terminal a.
	Action(state, symbol string) LRAction

	// Goto maps a state and a grammar symbol to some other state.
	Goto(state, symbol string) (string, error)

	// String prints a string representation of the table. If two LRParseTables
	// produce the same String() output, they are considered equal.
	String() string
}

type slrTable struct {
	gPrime    Grammar
	gStart    string
	lr0       DFA[util.Set[string]]
	itemCache map[string]LR0Item
	gTerms    []string
	gNonTerms []string
}

type canonicalLR1Table struct {
	gPrime    Grammar
	gStart    string
	lr1       DFA[util.BSet[string, LR1Item]]
	itemCache map[string]LR1Item
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

func (slr *slrTable) String() string {
	// need mapping of state to indexes
	stateRefs := map[string]string{}

	// need to gaurantee order
	stateNames := slr.lr0.States().Slice()
	sort.Strings(stateNames)

	// put the initial state first
	for i := range stateNames {
		if stateNames[i] == slr.lr0.Start {
			old := stateNames[0]
			stateNames[0] = stateNames[i]
			stateNames[i] = old
			break
		}
	}
	for i := range stateNames {
		stateRefs[stateNames[i]] = fmt.Sprintf("%d", i)
	}

	allTerms := make([]string, len(slr.gTerms))
	copy(allTerms, slr.gTerms)
	allTerms = append(allTerms, "$")

	// okay now do data setup
	data := [][]string{}

	// set up the headers
	headers := []string{"S", "|"}

	for _, t := range allTerms {
		headers = append(headers, fmt.Sprintf("A:%s", t))
	}

	headers = append(headers, "|")

	for _, nt := range slr.gNonTerms {
		headers = append(headers, fmt.Sprintf("G:%s", nt))
	}
	data = append(data, headers)

	// now need to do each state
	for stateIdx := range stateNames {
		i := stateNames[stateIdx]
		row := []string{stateRefs[i], "|"}

		for _, t := range allTerms {
			act := slr.Action(i, t)

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

		for _, nt := range slr.gNonTerms {
			var cell = ""

			gotoState, err := slr.Goto(i, nt)
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

func (slr *slrTable) Initial() string {
	return slr.lr0.Start
}

func (clr1 *canonicalLR1Table) Goto(state, symbol string) (string, error) {
	// step 3 of algorithm 4.56, "Construction of canonicalLR-parsing tables",
	// for reference:

	// 3. The goto transitions for state i are constructed for all nonterminals
	// A using the rule: If GOTO(Iᵢ, A) = Iⱼ, then GOTO[i, A] = j.
	newState := clr1.lr1.Next(state, symbol)
	if newState == "" {
		return "", fmt.Errorf("GOTO[%q, %q] is an error entry", state, symbol)
	}
	return newState, nil
}

func (slr *slrTable) Goto(state, symbol string) (string, error) {
	// as purple  dragon book mentions, "intuitively, the GOTO function is used
	// to define the transitions in the LR(0) automaton for a grammar." We will
	// take advantage of the corollary; we already have the automaton defined,
	// so consequently the transitions of it can be used to derive the value of
	// GOTO(i, a).

	// assume the state is the concatenated items in the set. Up to caller to
	// enshore this is the glubbin case.

	// step 3 of algorithm 4.46, "Constructing an SLR-parsing table", for
	// reference

	// 3. The goto transitions for state i are constructed for all nonterminals
	// A using the rule: If GOTO(Iᵢ, A) = Iⱼ, then GOTO[i, A] = j.

	newState := slr.lr0.Next(state, symbol)

	if newState == "" {
		return "", fmt.Errorf("GOTO[%q, %q] is an error entry", state, symbol)
	}
	return newState, nil
}

func (clr1 *canonicalLR1Table) Action(i, a string) LRAction {
	// step 2 of algorithm 4.56, "Construction of canonicalLR-parsing tables",
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
			newAct := LRAction{Type: LRReduce, Symbol: A, Production: Production(alpha)}
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

func (slr *slrTable) Action(i, a string) LRAction {
	// step 2 of algorithm 4.46, "Constructing an SLR-parsing table", for
	// reference

	// 2. State i is constructed from Iᵢ. The parsing actions for state i are
	// determined as follows:

	// get our set back from current state so we can check it; this is our Iᵢ
	itemSet := slr.lr0.GetValue(i)

	// we have gauranteed that these dont conflict during construction; still,
	// check it so we can panic if it conflicts
	var alreadySet bool
	var act LRAction

	// Okay, "[some random item] is in Iᵢ" is suuuuuuuuper vague. We're
	// basically going to have to check each item and see if it is in the
	// pattern. I *guess* ::::/
	for itemStr := range itemSet {
		item := slr.itemCache[itemStr]

		// given item is [A -> α.β]:
		A := item.NonTerminal
		alpha := item.Left
		beta := item.Right

		followA := util.Set[string]{}
		if A != slr.gPrime.StartSymbol() {
			// we'll need this later, glub 38)
			followA = slr.gPrime.FOLLOW(A)
		}

		// (a) If [A -> α.aβ] is in Iᵢ and GOTO(Iᵢ, a) = Iⱼ, then set
		// ACTION[i, a] to "shift j." Here a must be a terminal.
		//
		// we'll assume α can be ε.
		// β can also be ε but note this β is rly β[1:] from earlier notation
		// used to assign beta (beta := item.Right).
		if slr.gPrime.IsTerminal(a) && len(beta) > 0 && beta[0] == a {
			j, err := slr.Goto(i, a)

			// it's okay if we get an error; it just means there is no
			// transition defined (i think, glub, the purple dragon book's
			// method of constructing GOTO would have it returning an empty
			// set in this case but unshore), so it is not a match.
			if err == nil {
				// match found
				newAct := LRAction{Type: LRShift, State: j}
				if alreadySet && !newAct.Equal(act) {
					panic(fmt.Sprintf("grammar is not SLR: found both %s and %s actions for input %q", act.String(), newAct.String(), a))
				}
				act = newAct
				alreadySet = true
			}
		}

		// (b) If [A -> α.] is in Iᵢ, then set ACTION[i, a] to "reduce A -> α"
		// for all a in FOLLOW(A); here A may not be S'.
		//
		// we'll assume α can be empty.
		// the beta we previously retrieved MUST be empty
		if len(beta) == 0 && followA.Has(a) && A != slr.gPrime.StartSymbol() {
			newAct := LRAction{Type: LRReduce, Symbol: A, Production: Production(alpha)}
			if alreadySet && !newAct.Equal(act) {
				panic(fmt.Sprintf("grammar is not SLR(1): found both %s and %s actions for input %q", act.String(), newAct.String(), a))
			}
			act = newAct
			alreadySet = true
		}

		// (c) If [S' -> S.] is in Iᵢ, then set ACTION[i, $] to "accept".
		if a == "$" && A == slr.gPrime.StartSymbol() && len(alpha) == 1 && alpha[0] == slr.gStart && len(beta) == 0 {
			newAct := LRAction{Type: LRAccept}
			if alreadySet && !newAct.Equal(act) {
				panic(fmt.Sprintf("grammar is not SLR(1): found both %s and %s actions for input %q", act.String(), newAct.String(), a))
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

// ConstructSimpleLRParseTable constructs the SLR(1) table for G. It augments
// grammar G to produce G', then the canonical collection of sets of items of G'
// is used to construct a table with applicable GOTO and ACTION columns.
//
// This is an implementation of Algorithm 4.46, "Constructing an SLR-parsing
// table", from the purple dragon book. In the comments, most of which is lifted
// directly from the textbook, GOTO[i, A] refers to the vaue of the table's
// GOTO column at state i, symbol A, while GOTO(i, A) refers to the "precomputed
// GOTO function for grammar G'".
func ConstructSimpleLRParseTable(g Grammar) (LRParseTable, error) {
	// we will skip a few steps here and simply grab the LR0 DFA for G' which
	// will pretty immediately give us our GOTO() function, since as purple
	// dragon book mentions, "intuitively, the GOTO function is used to define
	// the transitions in the LR(0) automaton for a grammar."
	lr0Automaton := NewLR0ViablePrefixNFA(g).ToDFA()

	table := &slrTable{
		gPrime:    g.Augmented(),
		gStart:    g.StartSymbol(),
		gTerms:    g.Terminals(),
		gNonTerms: g.NonTerminals(),
		lr0:       lr0Automaton,
		itemCache: map[string]LR0Item{},
	}

	for _, item := range table.gPrime.LR0Items() {
		table.itemCache[item.String()] = item
	}

	// check ahead to see if we would get conflicts in ACTION function
	for i := range lr0Automaton.States() {
		for _, a := range table.gPrime.Terminals() {
			itemSet := table.lr0.GetValue(i)
			var matchFound bool
			var act LRAction
			for itemStr := range itemSet {
				item := table.itemCache[itemStr]
				A := item.NonTerminal
				alpha := item.Left
				beta := item.Right

				followA := util.Set[string]{}
				if A != table.gPrime.StartSymbol() {
					// we'll need this later, glub 38)
					followA = table.gPrime.FOLLOW(A)
				}

				if table.gPrime.IsTerminal(a) && len(beta) > 0 && beta[0] == a {
					j, err := table.Goto(i, a)
					if err == nil {
						// match found
						newAct := LRAction{Type: LRShift, State: j}
						if matchFound && !newAct.Equal(act) {
							return nil, fmt.Errorf("grammar is not SLR: found both %s and %s actions for input %q", act.String(), newAct.String(), a)
						}
						act = newAct
						matchFound = true
					}
				}

				if len(beta) == 0 && followA.Has(a) && A != table.gPrime.StartSymbol() {
					newAct := LRAction{Type: LRReduce, Symbol: A, Production: Production(alpha)}
					if matchFound && !newAct.Equal(act) {
						return nil, fmt.Errorf("grammar is not SLR(1): found both %s and %s actions for input %q", act.String(), newAct.String(), a)
					}
					act = newAct
					matchFound = true
				}

				if a == "$" && A == table.gPrime.StartSymbol() && len(alpha) == 1 && alpha[0] == table.gStart && len(beta) == 0 {
					newAct := LRAction{Type: LRAccept}
					if matchFound && !newAct.Equal(act) {
						return nil, fmt.Errorf("grammar is not SLR(1): found both %s and %s actions for input %q", act.String(), newAct.String(), a)
					}
					act = newAct
					matchFound = true
				}
			}
		}
	}

	return table, nil
}

// ConstructCanonicalLR1ParseTable constructs the canonical LR(1) table for G.
// It augments grammar G to produce G', then the canonical collection of sets of
// LR(1) items of G' is used to construct a table with applicable GOTO and
// ACTION columns.
//
// This is an implementation of Algorithm 4.56, "Construction of canonical-LR
// parsing tables", from the purple dragon book. In the comments, most of which
// is lifted directly from the textbook, GOTO[i, A] refers to the vaue of the
// table's GOTO column at state i, symbol A, while GOTO(i, A) refers to the
// "precomputed GOTO function for grammar G'".
func ConstructCanonicalLR1ParseTable(g Grammar) (LRParseTable, error) {
	// we will skip a few steps here and simply grab the LR0 DFA for G' which
	// will pretty immediately give us our GOTO() function, since as purple
	// dragon book mentions, "intuitively, the GOTO function is used to define
	// the transitions in the LR(0) automaton for a grammar."
	lr1Automaton := NewLR1ViablePrefixDFA(g)

	table := &canonicalLR1Table{
		gPrime:    g.Augmented(),
		gStart:    g.StartSymbol(),
		gTerms:    g.Terminals(),
		gNonTerms: g.NonTerminals(),
		lr1:       lr1Automaton,
		itemCache: map[string]LR1Item{},
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
					newAct := LRAction{Type: LRReduce, Symbol: A, Production: Production(alpha)}
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

// LRParse parses the input stream using the provided LRParser.
//
// This is an implementation of Algorithm 4.44, "LR-parsing algorithm", from
// the purple dragon book.
func LRParse(parser LRParseTable, stream tokenStream) (parseTree, error) {
	stateStack := util.Stack[string]{Of: []string{parser.Initial()}}

	// we will use these to build our parse tree
	tokenBuffer := util.Stack[token]{}
	subTreeRoots := util.Stack[*parseTree]{}

	// let a be the first symbol of w$;
	a := stream.Next()

	for { /* repeat forever */
		// let s be the state on top of the stack;
		s := stateStack.Peek()

		ACTION := parser.Action(s, a.class.id)

		switch ACTION.Type {
		case LRShift: // if ( ACTION[s, a] = shift t )
			// add token to our buffer
			tokenBuffer.Push(a)

			t := ACTION.State

			// push t onto the stack
			stateStack.Push(t)

			// let a be the next input symbol
			a = stream.Next()
		case LRReduce: // else if ( ACTION[s, a] = reduce A -> β )
			A := ACTION.Symbol
			beta := ACTION.Production

			// use the reduce to create a node in the parse tree
			node := &parseTree{value: A, children: make([]*parseTree, 0)}
			// we need to go from right to left of the production to pop things
			// from the stacks in the correct order
			for i := len(beta) - 1; i >= 0; i-- {
				sym := beta[i]
				if strings.ToLower(sym) == sym {
					// it is a terminal. read the source from the token buffer
					tok := tokenBuffer.Pop()
					subNode := &parseTree{terminal: true, value: tok.class.id, source: tok}
					node.children = append([]*parseTree{subNode}, node.children...)
				} else {
					// it is a non-terminal. it should be in our stack of
					// current tree roots.
					subNode := subTreeRoots.Pop()
					node.children = append([]*parseTree{subNode}, node.children...)
				}
			}
			// remember it for next time
			subTreeRoots.Push(node)

			// pop |β| symbols off the stack;
			for i := 0; i < len(beta); i++ {
				stateStack.Pop()
			}

			// let state t now be on top of the stack
			t := stateStack.Peek()

			// push GOTO[t, A] onto the stack
			toPush, err := parser.Goto(t, A)
			if err != nil {
				return parseTree{}, syntaxErrorFromLexeme("parsing failed", a)
			}
			stateStack.Push(toPush)

			// output the production A -> β
			// TODO: put it on the parse tree
			fmt.Printf("PUT %q -> %q ON PARSE TREE\n", A, beta)
		case LRAccept: // else if ( ACTION[s, a] = accept )
			// parsing is done. there should be at least one item on the stack
			pt := subTreeRoots.Pop()
			return *pt, nil
		case LRError:
			// call error-recovery routine
			// TODO: error recovery, for now, just report it
			return parseTree{}, syntaxErrorFromLexeme("parsing failed", a)
		}
	}
}
