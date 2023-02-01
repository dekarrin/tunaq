package tunascript

import (
	"fmt"
	"math"
	"strings"

	"github.com/dekarrin/rosed"
	"github.com/dekarrin/tunaq/internal/util"
)

type LR0Item struct {
	NonTerminal string
	Left        []string
	Right       []string
}

func (item LR0Item) String() string {
	nonTermPhrase := ""
	if item.NonTerminal != "" {
		nonTermPhrase = fmt.Sprintf("%s -> ", item.NonTerminal)
	}

	return fmt.Sprintf("%s%s.%s", nonTermPhrase, strings.Join(item.Left, " "), strings.Join(item.Right, " "))
}

type Production []string

var (
	Epsilon = Production{""}
	Error   = Production{}
)

// Copy returns a deep-copied duplicate of this production.
func (p Production) Copy() Production {
	p2 := make(Production, len(p))
	copy(p2, p)

	return p2
}

// AllItems returns all LR0 items of the production. Note: a Production does not
// know what non-terminal produces it, so the NonTerminal field of the returned
// LR0Items will be blank.
func (p Production) AllItems() []LR0Item {
	if p.Equal(Epsilon) {
		return []LR0Item{}
	}

	items := []LR0Item{}
	for dot := 0; dot < len(p); dot++ {
		item := LR0Item{
			Left:  p[:dot],
			Right: p[dot:],
		}
		items = append(items, item)
	}

	// finally, add the single dot for the end
	items = append(items, LR0Item{Left: p})

	return items
}

// Equal returns whether Rule is equal to another value. It will not be equal
// if the other value cannot be cast to Production or *Production.
func (p Production) Equal(o any) bool {
	other, ok := o.(Production)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*Production)
		if !ok {
			// also okay if it's a string slice
			otherSlice, ok := o.([]string)

			if !ok {
				// also okay if it's a ptr to string slice
				otherSlicePtr, ok := o.(*[]string)
				if !ok {
					return false
				} else if otherSlicePtr == nil {
					return false
				} else {
					other = Production(*otherSlicePtr)
				}
			} else {
				other = Production(otherSlice)
			}
		} else if otherPtr == nil {
			return false
		} else {
			other = *otherPtr
		}
	}

	if len(p) != len(other) {
		return false
	} else {
		for i := range p {
			if p[i] != other[i] {
				return false
			}
		}
	}

	return true
}

func (p Production) String() string {
	// if it's an epsilon production output that symbol only
	if p.Equal(Epsilon) {
		return "ε"
	}
	// separate each by space and call it good

	var sb strings.Builder

	for i := range p {
		sb.WriteString(p[i])
		if i+1 < len(p) {
			sb.WriteRune(' ')
		}

	}

	return sb.String()
}

// IsUnit returns whether this production is a unit production.
func (p Production) IsUnit() bool {
	return len(p) == 1 && !p.Equal(Epsilon) && strings.ToUpper(p[0]) == p[0]
}

// HasSymbol returns whether the production has the given symbol in it.
func (p Production) HasSymbol(sym string) bool {
	return util.InSlice(sym, p)
}

// terminals will be upper, non-terms will be lower.
type Rule struct {
	NonTerminal string
	Productions []Production
}

// Returns all LRItems in the Rule with their NonTerminal field properly set.
func (r Rule) LRItems() []LR0Item {
	items := []LR0Item{}
	for _, p := range r.Productions {
		prodItems := p.AllItems()
		for i := range prodItems {
			item := prodItems[i]
			item.NonTerminal = r.NonTerminal
			prodItems[i] = item
		}
		items = append(items, prodItems...)
	}
	return items
}

// Copy returns a deep-copy duplicate of the given Rule.
func (r Rule) Copy() Rule {
	r2 := Rule{
		NonTerminal: r.NonTerminal,
		Productions: make([]Production, len(r.Productions)),
	}

	for i := range r.Productions {
		r2.Productions[i] = r.Productions[i].Copy()
	}

	return r2
}

func (r Rule) String() string {
	var sb strings.Builder

	sb.WriteString(r.NonTerminal)
	sb.WriteString(" -> ")

	for i := range r.Productions {
		sb.WriteString(r.Productions[i].String())
		if i+1 < len(r.Productions) {
			sb.WriteString(" | ")
		}
	}

	return sb.String()
}

// ReplaceProduction returns a rule that does not include the given production
// and subsitutes the given production(s) for it. If no productions are given
// the specified production is simply removed. If the specified production
// does not exist, the replacements are added to the end of the rule.
func (r Rule) ReplaceProduction(p Production, replacements ...Production) Rule {
	var addedReplacements bool
	newProds := []Production{}
	for i := range r.Productions {
		if !r.Productions[i].Equal(p) {
			newProds = append(newProds, r.Productions[i])
		} else if len(replacements) > 0 {
			newProds = append(newProds, replacements...)
			addedReplacements = true
		}
	}
	if !addedReplacements {
		newProds = append(newProds, replacements...)
	}

	r.Productions = newProds
	return r
}

// Equal returns whether Rule is equal to another value. It will not be equal
// if the other value cannot be casted to a Rule or *Rule.
func (r Rule) Equal(o any) bool {
	other, ok := o.(Rule)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*Rule)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if r.NonTerminal != other.NonTerminal {
		return false
	} else if !util.EqualSlices(r.Productions, other.Productions) {
		return false
	}

	return true
	// cant do util.EqualSlices here because Productions is a slice of []string
}

// CanProduce returns whether this rule can produce the given Production.
func (r Rule) CanProduce(p Production) bool {
	for _, alt := range r.Productions {
		if alt.Equal(p) {
			return true
		}
	}
	return false
}

// CanProduceSymbol whether any alternative in productions produces the
// given term/non-terminal
func (r Rule) CanProduceSymbol(termOrNonTerm string) bool {
	for _, alt := range r.Productions {
		for _, sym := range alt {
			if sym == termOrNonTerm {
				return true
			}
		}
	}
	return false
}

// HasProduction returns whether the rule has a production of the exact sequence
// of symbols entirely.
func (r Rule) HasProduction(prod Production) bool {
	for _, alt := range r.Productions {
		if len(alt) == len(prod) {
			eq := true
			for i := range alt {
				if alt[i] != prod[i] {
					eq = false
					break
				}
			}
			if eq {
				return true
			}
		}
	}
	return false
}

// UnitProductions returns all productions from the Rule that are unit
// productions; i.e. are of the form A -> B where both A and B are
// non-terminals.
func (r Rule) UnitProductions() []Production {
	prods := []Production{}

	for _, alt := range r.Productions {
		if alt.IsUnit() {
			prods = append(prods, alt)
		}
	}

	return prods
}

// Grammar for tunascript language, used by a parsing algorithm to create a
// parse tree from some input.
type Grammar struct {
	rulesByName map[string]int

	// main rules store, not just doing a simple map bc
	// rules may have order that matters
	rules     []Rule
	terminals map[string]tokenClass

	// name of the start symbol. If not set, assumed to be S.
	Start string
}

// LRItems returns all LR0 Items in the grammar.
func (g Grammar) LRItems() []LR0Item {
	nonTerms := g.NonTerminals()

	items := []LR0Item{}
	for _, nt := range nonTerms {
		r := g.Rule(nt)
		items = append(items, r.LRItems()...)
	}
	return items
}

// Copy makes a duplicate deep copy of the grammar.
func (g Grammar) Copy() Grammar {
	g2 := Grammar{
		rulesByName: make(map[string]int, len(g.rulesByName)),
		rules:       make([]Rule, len(g.rules)),
		terminals:   make(map[string]tokenClass, len(g.terminals)),
		Start:       g.Start,
	}

	for k := range g.rulesByName {
		g2.rulesByName[k] = g.rulesByName[k]
	}

	for i := range g.rules {
		g2.rules[i] = g.rules[i].Copy()
	}

	for k := range g.terminals {
		g2.terminals[k] = g.terminals[k]
	}

	return g2
}

func (g Grammar) StartSymbol() string {
	if g.Start == "" {
		return "S"
	} else {
		return g.Start
	}
}

func (g Grammar) String() string {
	return fmt.Sprintf("(%q, R=%q)", util.OrderedKeys(g.terminals), g.rules)
}

// Rule returns the grammar rule for the given nonterminal symbol.
// If there is no rule defined for that nonterminal, a Rule with an empty
// NonTerminal field is returned; else it will be the same string as the one
// passed in to the function.
func (g Grammar) Rule(nonterminal string) Rule {
	if g.rulesByName == nil {
		return Rule{}
	}

	if curIdx, ok := g.rulesByName[nonterminal]; !ok {
		return Rule{}
	} else {
		return g.rules[curIdx]
	}
}

// Term returns the tokenClass that the given terminal symbol maps to. If the
// given terminal symbol is not defined as a terminal symbol in this grammar,
// the special tokenClass tsUndefined is returned.
func (g Grammar) Term(terminal string) tokenClass {
	if g.terminals == nil {
		return tsUndefined
	}

	if class, ok := g.terminals[terminal]; !ok {
		return tsUndefined
	} else {
		return class
	}
}

// AddTerm adds the given terminal along with the tokenClass that corresponds to
// it; tokens must be of that class in order to match the terminal.
//
// The mapping of terminal symbol IDs to tokenClasses must be 1-to-1; i.e. It is
// an error to map multiple terms to the same tokenClass, and it is an error to
// map the same term to multiple tokenClasses.
//
// As a result, redefining the same term will cause the old one to be removed,
// and during validation if multiple terminals are matched to the same
// tokenClass it will be considered an error.
//
// It is an error to map any terminal to tsUndefined and attempting to do so
// will panic immediately.
func (g *Grammar) AddTerm(terminal string, class tokenClass) {
	if terminal == "" {
		panic("empty terminal not allowed")
	}

	if class.Equal(tsEndOfText) {
		panic("can't add out-of-band signal tsEndOfText as defined terminal")
	}

	// ensure that it isnt an illegal char, only things used should be 'a-z',
	// '_', and '-'
	for _, ch := range terminal {
		if ('a' > ch || ch > 'z') && ch != '_' && ch != '-' {
			panic(fmt.Sprintf("invalid terminal name %q; must only be chars a-z, \"_\", or \"-\"", terminal))
		}
	}

	if class == tsUndefined {
		panic("cannot explicitly map a terminal to tsUndefined")
	}

	if g.terminals == nil {
		g.terminals = map[string]tokenClass{}
	}

	g.terminals[terminal] = class
}

// RemoveRule eliminates all productions of the given nonterminal from the
// grammar. The nonterminal will no longer be considered to be a part of the
// Grammar.
//
// If the grammar already does not contain the given non-terminal this function
// has no effect.
func (g *Grammar) RemoveRule(nonterminal string) {
	// is this rule even present?

	ruleIdx, ok := g.rulesByName[nonterminal]
	if !ok {
		// that was easy
		return
	}

	// delete name -> index mapping
	delete(g.rulesByName, nonterminal)

	// delete from main store
	if ruleIdx+1 < len(g.rules) {
		g.rules = append(g.rules[:ruleIdx], g.rules[ruleIdx+1:]...)

		// Hold on, we just need to adjust the indexes across this quick...
		for i := ruleIdx; i < len(g.rules); i++ {
			r := g.rules[i]
			g.rulesByName[r.NonTerminal] = i
		}
	} else {
		g.rules = g.rules[:ruleIdx]
	}
}

// AddRule adds the given production for a nonterminal. If the nonterminal has
// already been given, the production is added as an alternative for that
// nonterminal with lower priority than all others already added.
//
// All rules require at least one symbol in the production. For episilon
// production, give only the empty string.
//
// TOOD: disallow dupe prods
func (g *Grammar) AddRule(nonterminal string, production []string) {
	if nonterminal == "" {
		panic("empty nonterminal name not allowed for production rule")
	}

	// ensure that it isnt an illegal char, only things used should be 'A-Z',
	// '_', and '-'
	for _, ch := range nonterminal {
		if ('A' > ch || ch > 'Z') && ch != '_' && ch != '-' {
			panic(fmt.Sprintf("invalid nonterminal name %q; must only be chars A-Z, \"_\", or \"-\"", nonterminal))
		}
	}

	if len(production) < 1 {
		panic("for epsilon production give empty string; all rules must have productions")
	}

	// check that epsilon, if given, is by itself
	if len(production) != 1 {
		for _, sym := range production {
			if sym == "" {
				panic("episilon production only allowed as sole production of an alternative")
			}
		}
	}

	if g.rulesByName == nil {
		g.rulesByName = map[string]int{}
	}

	curIdx, ok := g.rulesByName[nonterminal]
	if !ok {
		g.rules = append(g.rules, Rule{NonTerminal: nonterminal})
		curIdx = len(g.rules) - 1
		g.rulesByName[nonterminal] = curIdx
	}

	curRule := g.rules[curIdx]
	curRule.Productions = append(curRule.Productions, production)
	g.rules[curIdx] = curRule
}

// NonTerminals returns list of all the non-terminal symbols. All will be upper
// case.
func (g Grammar) NonTerminals() []string {
	return util.OrderedKeys(g.rulesByName)
}

// ReversePriorityNonTerminals returns list of all the non-terminal symbols in
// reverse order from the order they were defined in. This is handy because it
// can have the effect of causing iteration to do so in a manner that a human
// might do looking at a grammar, reversed.
func (g Grammar) ReversePriorityNonTerminals() []string {
	termNames := []string{}
	for _, r := range g.rules {
		termNames = append([]string{r.NonTerminal}, termNames...)
	}

	return termNames
}

// UnitProductions returns all production rules that are of the form A -> B,
// where A and B are both non-terminals. The returned list contains rules
// mapping the non-terminal to the other non-terminal; all other productions
// from the grammar will not be present.
func (g Grammar) UnitProductions() []Rule {
	allUnitProductions := []Rule{}

	for _, nonTerm := range g.NonTerminals() {
		rule := g.Rule(nonTerm)
		ruleUnitProds := rule.UnitProductions()
		if len(ruleUnitProds) > 0 {
			allUnitProductions = append(allUnitProductions, Rule{NonTerminal: nonTerm, Productions: ruleUnitProds})
		}
	}

	return allUnitProductions
}

// HasUnreachables returns whether the grammar currently has unreachle
// non-terminals.
func (g Grammar) HasUnreachableNonTerminals() bool {
	for _, nonTerm := range g.NonTerminals() {
		if nonTerm == g.StartSymbol() {
			continue
		}

		reachable := false
		for _, otherNonTerm := range g.NonTerminals() {
			if otherNonTerm == nonTerm {
				continue
			}

			r := g.Rule(otherNonTerm)
			if r.CanProduceSymbol(nonTerm) {
				reachable = true
				break
			}
		}

		if !reachable {
			return true
		}

	}

	return false
}

// UnreachableNonTerminals returns all non-terminals (excluding the start
// symbol) that are currently unreachable due to not being produced by any other
// grammar rule.
func (g Grammar) UnreachableNonTerminals() []string {
	unreachables := []string{}

	for _, nonTerm := range g.NonTerminals() {
		if nonTerm == g.StartSymbol() {
			continue
		}

		reachable := false
		for _, otherNonTerm := range g.NonTerminals() {
			if otherNonTerm == nonTerm {
				continue
			}

			r := g.Rule(otherNonTerm)
			if r.CanProduceSymbol(nonTerm) {
				reachable = true
				break
			}
		}

		if !reachable {
			unreachables = append(unreachables, nonTerm)
		}
	}

	return unreachables
}

// RemoveUnitProductions returns a Grammar that derives strings equivalent to
// this one but with all unit production rules removed.
func (g Grammar) RemoveUnitProductions() Grammar {
	for _, nt := range g.NonTerminals() {
		rule := g.Rule(nt)
		resolvedSymbols := map[string]bool{}
		for len(rule.UnitProductions()) > 0 {
			newProds := []Production{}
			for _, p := range rule.Productions {
				if p.IsUnit() && p[0] != nt {
					hoistedRule := g.Rule(p[0])
					includedHoistedProds := []Production{}
					for _, hoistedProd := range hoistedRule.Productions {
						if len(hoistedProd) == 1 && hoistedProd[0] == nt {
							// dont add
						} else if rule.CanProduce(hoistedProd) {
							// dont add
						} else if _, ok := resolvedSymbols[p[0]]; ok {
							// dont add
						} else {
							includedHoistedProds = append(includedHoistedProds, hoistedProd)
						}
					}

					newProds = append(newProds, includedHoistedProds...)
					resolvedSymbols[p[0]] = true
				} else {
					newProds = append(newProds, p)
				}
			}
			rule.Productions = newProds
		}

		g.rules[g.rulesByName[rule.NonTerminal]] = rule
	}

	// okay, now just remove the unreachable ones (not strictly necessary for
	// all interpretations of unit production removal but lets do it anyways for
	// simplicity)
	g = g.RemoveUreachableNonTerminals()

	return g
}

// RemoveUnreachableNonTerminals returns a grammar with all unreachable
// non-terminals removed.
func (g Grammar) RemoveUreachableNonTerminals() Grammar {
	for g.HasUnreachableNonTerminals() {
		for _, nt := range g.UnreachableNonTerminals() {
			g.RemoveRule(nt)
		}
	}
	return g
}

// RemoveEpsilons returns a grammar that derives strings equivalent to the first
// one (with the exception of the empty string) but with all epsilon productions
// automatically eliminated.
//
// Call Validate before this or it may go poorly.
func (g Grammar) RemoveEpsilons() Grammar {
	// run this in a loop until all vars have epsilon propagated out

	propagated := map[string]bool{}
	// first find all of the non-terminals that have epsilon productions

	for {
		// find the first non-terminal with an epsilon production
		toPropagate := ""
		for _, A := range g.NonTerminals() {
			ruleIdx := g.rulesByName[A]
			rule := g.rules[ruleIdx]

			if rule.HasProduction(Epsilon) {
				toPropagate = A
				break
			}
		}

		// if we didn't find any non-terminals with epsilon productions then
		// there are none remaining and we are done.
		if toPropagate == "" {
			break
		}

		// let's call the non-terminal whose epsilons are about to be propegated
		// up 'A'
		A := toPropagate

		// for each of those, remove them from all others
		producesA := map[string]bool{}

		ruleA := g.Rule(A)
		// find all non-terms that produce this, not including self
		for _, B := range g.NonTerminals() {
			ruleIdx := g.rulesByName[B]
			rule := g.rules[ruleIdx]

			// does b produce A?
			if rule.CanProduceSymbol(A) {
				producesA[B] = true
			}
		}

		// okay, now for each production that produces A...
		for B := range producesA {
			ruleB := g.Rule(B)

			if len(ruleA.Productions) == 1 {
				// if A is ONLY an epsilon producer, B can safely eliminate every
				// A from its productions.

				// remove all As from B productions. if it was a unit production,
				// replace it with an epsilon production
				for i, bProd := range ruleB.Productions {
					var newProd Production
					if len(bProd) == 1 && bProd[0] == A {
						newProd = Epsilon
					} else {
						for _, sym := range bProd {
							if sym != A {
								newProd = append(newProd, sym)
							}
						}
					}
					ruleB.Productions[i] = newProd
				}
			} else {
				// general algorithm, summarized in video:
				// https://www.youtube.com/watch?v=j9cNTlGkyZM

				// for each production of b
				var newProds []Production
				for _, bProd := range ruleB.Productions {
					if util.InSlice(A, bProd) {
						// gen all permutations of A being epsi for that
						// production
						// AsA -> AsA, sA, s, As
						// AAsA -> AAsA, AsA, AsA,
						rewrittenEpsilons := getEpsilonRewrites(A, bProd)

						newProds = append(newProds, rewrittenEpsilons...)
					} else {
						// keep it as-is
						newProds = append(newProds, bProd)
					}
				}

				// if B has already propagated epsilons up we can immediately
				// remove any epsilons it just received
				if _, propagatedEpsilons := propagated[B]; propagatedEpsilons {
					newProds = removeEpsilons(newProds)
				}

				ruleB.Productions = newProds
			}

			if A == B {
				// update our A rule if we need to
				ruleA = ruleB
			}

			ruleBIdx := g.rulesByName[B]
			g.rules[ruleBIdx] = ruleB
		}

		// A is now 'covered'; if it would get an epsilon propagated to it
		// it can remove it directly bc it having an epsilon prod has already
		// been propagated up.
		propagated[A] = true
		ruleA.Productions = removeEpsilons(ruleA.Productions)
		g.rules[g.rulesByName[A]] = ruleA
	}

	// did we just make any rules empty? probably should double-check that.

	// A may be unused by this point, may want to fix that
	return g
}

// RemoveLeftRecursion returns a grammar that has no left recursion, suitable
// for operations on by a top-down parsing method.
//
// This will force immediate removal of epsilon-productions and unit-productions
// as well, as this algorithem only works on CFGs without those.
//
// This is an implementation of Algorithm 4.19 from the purple dragon book,
// "Eliminating left recursion".
func (g Grammar) RemoveLeftRecursion() Grammar {
	// precond: grammar must have no epsilon productions or unit productions
	g = g.RemoveEpsilons().RemoveUnitProductions()

	grammarUpdated := true
	for grammarUpdated {
		grammarUpdated = false

		// arrange the nonterminals in some order A₁, A₂, ..., Aₙ.
		A := g.ReversePriorityNonTerminals()
		for i := range A {
			AiRule := g.Rule(A[i])
			for j := 0; j < i; j++ {
				AjRule := g.Rule(A[j])

				// replace each production of the form Aᵢ -> Aⱼγ by the
				// productions Aᵢ -> δ₁γ | δ₂γ | ... | δₖγ, where
				// Aⱼ -> δ₁ | δ₂ | ... | δₖ are all current Aⱼ productions

				newProds := []Production{}
				for k := range AiRule.Productions {
					if AiRule.Productions[k][0] == A[j] { // if rule is Aᵢ -> Aⱼγ (γ may be ε)
						grammarUpdated = true
						gamma := AiRule.Productions[k][1:]
						deltas := AjRule.Productions

						// add replacement rules
						for d := range deltas {
							deltaProd := deltas[d]
							newProds = append(newProds, append(deltaProd, gamma...))
						}
					} else {
						// add it unchanged
						newProds = append(newProds, AiRule.Productions[k])
					}
				}

				// persist the changes
				AiRule.Productions = newProds
				g.rules[g.rulesByName[A[i]]] = AiRule
			}

			// eliminate the immediate left recursion

			// first, group the productions as
			//
			// A -> Aα₁ | Aα₂ | ... | Aαₘ | β₁ | β₂ | βₙ
			//
			// where no βᵢ starts with an A.
			//
			// ^ That was purple dragon book. 8ut transl8ed, *I* say...
			// "put all the immediate left recursive productions first."
			alphas := []Production{}
			betas := []Production{}
			for k := range AiRule.Productions {
				if AiRule.Productions[k][0] == AiRule.NonTerminal {
					alphas = append(alphas, AiRule.Productions[k][1:])
				} else {
					betas = append(betas, AiRule.Productions[k])
				}
			}

			if len(alphas) > 0 {
				grammarUpdated = true

				// then, replace the A-productions by
				//
				// A  -> β₁A' | β₂A' | ... | βₙA'
				// A' -> α₁A' | α₂A' | ... | αₘA' | ε
				//
				// (purple dragon book)

				if len(betas) < 1 {

					// if we have zero betas, we need to have A produce A' only.
					// but if that's the case, then A -> A' becomes a
					// unit production and since we would be creating A' now, we
					// know A is the only non-term that would produce it,
					// therefore there is no point in putting in a new term and
					// we can immediately just shove all the A' rules into A
					newARule := Rule{NonTerminal: AiRule.NonTerminal}

					for _, a := range alphas {
						newARule.Productions = append(newARule.Productions, append(a, AiRule.NonTerminal))
					}
					// also add epsilon
					newARule.Productions = append(newARule.Productions, Epsilon)

					// update A
					AiRule = newARule
					g.rules[g.rulesByName[A[i]]] = AiRule
				} else {
					APrime := g.GenerateUniqueName(AiRule.NonTerminal)
					newARule := Rule{NonTerminal: AiRule.NonTerminal}
					newAprimeRule := Rule{NonTerminal: APrime}

					for _, b := range betas {
						newARule.Productions = append(newARule.Productions, append(b, APrime))
					}
					for _, a := range alphas {
						newAprimeRule.Productions = append(newAprimeRule.Productions, append(a, APrime))
					}
					// also add epsilon to A'
					newAprimeRule.Productions = append(newAprimeRule.Productions, Epsilon)

					// update A
					AiRule = newARule
					g.rules[g.rulesByName[A[i]]] = AiRule

					// insert A' immediately after A (convention)
					// shouldn't be modifying what we are iterating over bc we are
					// iterating over a pre-retrieved list of nonterminals
					AiIndex := g.rulesByName[A[i]]

					g.insertRule(newAprimeRule, AiIndex)
				}
			}
		}
	}

	g = g.RemoveUreachableNonTerminals()

	return g
}

func (g *Grammar) insertRule(r Rule, idx int) {
	// explicitly copy the end of the slice because trying to
	// save a post list and then modifying has lead to aliasing
	// issues in past
	var postList []Rule = make([]Rule, len(g.rules)-(idx+1))
	copy(postList, g.rules[idx+1:])
	g.rules = append(g.rules[:idx+1], r)
	g.rules = append(g.rules, postList...)

	// update indexes
	for i := idx + 1; i < len(g.rules); i++ {
		g.rulesByName[g.rules[i].NonTerminal] = i
	}
}

// LeftFactor returns a new Grammar equivalent to this one but with all unclear
// alternative choices for a top-down parser are left factored to equivalent
// pairs of statements.
//
// This is an implementation of Algorithm 4.21 from the purple dragon book,
// "Left factoring a grammar".
func (g Grammar) LeftFactor() Grammar {
	changes := true
	for changes {
		changes = false
		A := g.NonTerminals()
		for i := range A {
			AiRule := g.Rule(A[i])
			// find the longest common prefix α common to two or more of Aᵢ's
			// alternatives

			alpha := []string{}
			for j := range AiRule.Productions {
				checkingAlt := AiRule.Productions[j]

				for k := j + 1; k < len(AiRule.Productions); k++ {
					againstAlt := AiRule.Productions[k]
					longestPref := util.LongestCommonPrefix(checkingAlt, againstAlt)

					// in this case we will simply always take longest between two
					// because anyfin else would require far more intense searching.
					// if more than one matches that, well awesome we'll pick that
					// up too!! 38D

					if len(longestPref) > len(alpha) {
						alpha = longestPref
					}
				}
			}

			if len(alpha) > 0 && !Epsilon.Equal(alpha) {
				// there is a non-trivial common prefix
				changes = true

				// Replace all of the A-productions A -> αβ₁ | αβ₂ | ... | αβₙ | γ,
				// where γ represents all alternatives that do not begin with α,
				// by:
				//
				// A  -> αA' | γ
				// A' -> β₁ | β₂ | ... | βₙ
				//
				// Where A' is a new-non-terminal.
				gamma := []Production{}
				betas := []Production{}

				for _, alt := range AiRule.Productions {
					if util.HasPrefix(alt, alpha) {
						beta := alt[len(alpha):]
						if len(beta) == 0 {
							beta = Epsilon
						}
						betas = append(betas, beta)
					} else {
						gamma = append(gamma, alt)
					}
				}

				APrime := g.GenerateUniqueName(AiRule.NonTerminal)
				APrimeRule := Rule{NonTerminal: APrime, Productions: betas}

				AiRule.Productions = append([]Production{append(Production(alpha), APrime)}, gamma...)
				// update A
				g.rules[g.rulesByName[A[i]]] = AiRule

				// insert A' immediately after A (convention)
				// shouldn't be modifying what we are iterating over bc we are
				// iterating over a pre-retrieved list of nonterminals
				AiIndex := g.rulesByName[A[i]]
				g.insertRule(APrimeRule, AiIndex)
			}
		}
	}

	return g
}

// recursiveFindFollowSet
func (g Grammar) recursiveFindFollowSet(X string, prevFollowChecks map[string]bool) map[string]bool {
	if X == "" {
		// there is no follow set. return nil.
		return nil
	}
	followSet := map[string]bool{}
	if X == g.StartSymbol() {
		followSet["$"] = true
	}

	A := g.NonTerminals()
	for i := range A {
		AiRule := g.Rule(A[i])

		for _, prod := range AiRule.Productions {
			if prod.HasSymbol(X) {
				// how many occurances of X are there? that says how many times
				// we need to do this, so find them
				var Xcount int
				for k := range prod {
					if prod[k] == X {
						Xcount++
					}
				}

				// do this for each occurance of X
				for Xoccurance := 0; Xoccurance < Xcount; Xoccurance++ {
					alpha := []string{}
					beta := []string{}
					var doneWithAlpha bool
					var Xencounter int
					for k := range prod {
						if prod[k] == X {
							Xencounter++
							if Xencounter > Xoccurance && !doneWithAlpha {
								// only count this as end of alpha if we are at the
								// occurance of X we are looking for
								doneWithAlpha = true
								continue
							}
						}
						if !doneWithAlpha {
							alpha = append(alpha, prod[k])
						} else {
							beta = append(beta, prod[k])
						}
					}

					// we now have our alpha, X, and beta

					// is there a FIRST in beta that isnt exclusively delta,
					// its firsts are in X's FOLLOW. Stop checking at the first
					// in beta that is NOT reducible to eps.
					for b := range beta {
						betaFirst := g.FIRST(beta[b])
						_, epsilonPresent := betaFirst[Epsilon[0]]

						for k := range betaFirst {
							if k != Epsilon[0] {
								followSet[k] = true
							}
						}

						if !epsilonPresent {
							// stop looping
							break
						}
					}

					// if X "can be" at the end of the production (i.e. if
					// either X is the final symbol of the production or if all
					// symbols following X are non-terminals with epsilon in
					// their FIRST sets), then FOLLOW(A) is in FOLLOW(X), where
					// A is the non-terminal producing X.
					canBeAtEnd := true
					for b := range beta {
						betaFirst := g.FIRST(beta[b])
						if _, ok := betaFirst[Epsilon[0]]; !ok {
							canBeAtEnd = false
							break
						}
					}
					if canBeAtEnd {
						// dont infinitely recurse; if the producer is the
						// symbol, there's no need to add the FOLLOW from it bc
						// we are CURRENTLY calculating it.
						//
						// similarly, track the symbols we are going through.
						// don't recheck for the same one.
						if _, ok := prevFollowChecks[A[i]]; A[i] != X && !ok {
							prevFollowChecks[X] = true
							followA := g.recursiveFindFollowSet(A[i], prevFollowChecks)
							for k := range followA {
								followSet[k] = true
							}
						}
					}
				}
			}
		}
	}

	return followSet
}

type LL1Table util.Matrix2[string, string, Production]

func (M LL1Table) Set(A string, a string, alpha Production) {
	util.Matrix2[string, string, Production](M).Set(A, a, alpha)
}

func (M LL1Table) String() string {
	data := [][]string{}

	terms := M.Terminals()
	nts := M.NonTerminals()

	topRow := []string{""}
	for i := range terms {
		topRow = append(topRow, terms[i])
	}
	data = append(data, topRow)

	for i := range nts {
		dataRow := []string{nts[i]}
		for j := range terms {
			prod := M.Get(nts[i], terms[j])
			dataRow = append(dataRow, prod.String())
		}
		data = append(data, dataRow)
	}

	return rosed.Edit("").
		InsertTableOpts(0, data, 80, rosed.Options{
			TableBorders: true,
		}).
		String()
}

// Get returns an empty Production if it does not exist, or the one at the
// given coords.
func (M LL1Table) Get(A string, a string) Production {
	v := util.Matrix2[string, string, Production](M).Get(A, a)
	if v == nil {
		return Error
	}
	return *v
}

// NonTerminals returns all non-terminals used as the X keys for values in this
// table.
func (M LL1Table) NonTerminals() []string {
	return util.OrderedKeys(M)
}

// Terminals returns all terminals used as the Y keys for values in this table.
// Note that the "$" is expected to be present in all LL1 prediction tables.
func (M LL1Table) Terminals() []string {
	termSet := map[string]bool{}

	for k := range M {
		subMap := map[string]map[string]Production(M)[k]

		for term := range subMap {
			termSet[term] = true
		}
	}

	return util.OrderedKeys(termSet)
}

func NewLL1Table() LL1Table {
	return LL1Table(util.NewMatrix2[string, string, Production]())
}

// LLParseTable builds and returns the LL parsing table for the grammar. If it's
// not an LL(1) grammar, returns error.
//
// This is an implementation of Algorithm 4.31, "Construction of a predictive
// parsing table" from the peerple deruuuuugon beeeeeerk. (purple dragon book
// glub)
func (g Grammar) LLParseTable() (M LL1Table, err error) {
	if !g.IsLL1() {
		return nil, fmt.Errorf("not an LL(1) grammar")
	}

	nts := g.NonTerminals()
	M = NewLL1Table()

	// For each production A -> α of the grammar, do the following:
	// -purple dragon book
	for _, A := range nts {
		ARule := g.Rule(A)
		for _, alpha := range ARule.Productions {
			FIRSTalpha := util.Set[string](g.FIRST(alpha[0]))

			// 1. For each terminal a in FIRST(A), add A -> α to M[A, a].
			// -purple dragon book
			//
			// (this LOOKS like a typo in that actually following terminology
			// in these comments, FIRST(A) means "FIRST OF ALL PRODUCTIONS OF
			// A" but specifically in this section of the book, this
			// terminalogy means ONLY the first set of production we are looking
			// at. So really this is a in FIRST(α) by the convention used in
			// these comments, but purple dragon calls it FIRST(A), which is
			// technically correct within the bounds of "For each production
			// A -> α").
			for a := range FIRSTalpha {
				if a != Epsilon[0] {
					M.Set(A, a, alpha)
				}
			}

			// 2. If ε is in FIRST(α), then for each terminal b in FOLLOW(A),
			// add A -> α to M[A, b]. If ε is in FIRST(α) and $ is in FOLLOW(A),
			// add A -> α to M[A, $] as well.
			if FIRSTalpha.Has(Epsilon[0]) {
				for b := range g.FOLLOW(A) {
					// we cover the $ case automatically by not rly caring about
					// them bein glubbin terminals to begin w. W3 SH3LL H4V3
					// 33LQU4L1TY >38]
					M.Set(A, b, alpha)
				}
			}
		}
	}

	return M, nil
}

func mustParseGrammar(gr string) Grammar {
	g, err := parseGrammar(gr)
	if err != nil {
		panic(err.Error())
	}
	return g
}

func parseGrammar(gr string) (Grammar, error) {
	lines := strings.Split(gr, ";")

	var g Grammar
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		rule, err := parseRule(line)
		if err != nil {
			return Grammar{}, err
		}

		for _, p := range rule.Productions {
			for _, sym := range p {
				if strings.ToLower(sym) == sym && sym != "" {
					g.AddTerm(strings.ToLower(sym), tokenClass{id: strings.ToLower(sym), human: sym})
				}
			}
			g.AddRule(rule.NonTerminal, p)
		}
	}

	return g, nil
}

func (g Grammar) TermFor(tc tokenClass) string {
	if tc.Equal(tsEndOfText) {
		return "$"
	}
	for k := range g.terminals {
		if g.terminals[k].Equal(tc) {
			return k
		}
	}
	return ""
}

func (g Grammar) IsLL1() bool {
	nts := g.NonTerminals()
	for _, A := range nts {
		AiRule := g.Rule(A)

		// we'll need this later, glubglub 38)
		followSetA := util.Set[string](g.FOLLOW(A))

		// Whenever A -> α | β are two distinct productions of G:
		// -purple dragon book
		for i := range AiRule.Productions {
			for j := i + 1; j < len(AiRule.Productions); j++ {
				alphaFIRST := g.FIRST(AiRule.Productions[i][0])
				betaFIRST := g.FIRST(AiRule.Productions[j][0])

				aFSet := util.Set[string](alphaFIRST)
				bFSet := util.Set[string](betaFIRST)

				// 1. For no terminal a do both α and β derive strings beginning
				// with a.
				//
				// 2. At most of of α and β derive the empty string.
				//
				//
				// ...or in other words, FIRST(α) and FIRST(β) are disjoint
				// sets.
				// -purple dragon book

				if !aFSet.DisjointWith(bFSet) {
					return false
				}

				// 3. If β =*> ε, then α does not derive any string beginning
				// with a terminal in FOLLOW(A). Likewise, if α =*> ε, then β
				// does not derive any string beginning with a terminal in
				// FOLLOW(A).
				//
				//
				// ...or in other words, if ε is in FIRST(β), then FIRST(α) and
				// FOLLOW(A) are disjoint sets, and likewise if ε is in
				// FIRST(α).
				// -perple dergon berk. (Purple dragon book)
				if bFSet.Has(Epsilon[0]) {
					if !followSetA.DisjointWith(aFSet) {
						return false
					}
				}
				if aFSet.Has(Epsilon[0]) {
					if !followSetA.DisjointWith(bFSet) {
						return false
					}
				}
			}

		}
	}

	return true
}

func (g Grammar) FOLLOW(X string) map[string]bool {
	return g.recursiveFindFollowSet(X, map[string]bool{})
}

func (g Grammar) FIRST(X string) map[string]bool {
	if strings.ToLower(X) == X {
		// terminal or epsilon
		return map[string]bool{X: true}
	} else {
		firsts := map[string]bool{}
		r := g.Rule(X)

		for ntIdx := range r.Productions {
			Y := r.Productions[ntIdx]
			var gotToEnd bool
			for k := 0; k < len(Y); k++ {
				firstY := g.FIRST(Y[k])
				for str := range firstY {
					if str != "" {
						firsts[str] = true
					}
				}
				if len(firstY) == 1 && util.OrderedKeys(firstY)[0] == "" {
					firsts[""] = true
				}
				if _, ok := firstY[Epsilon[0]]; !ok {
					// if its not, then break
					break
				}
				if k+1 >= len(Y) {
					gotToEnd = true
				}
			}
			if gotToEnd {
				firsts[Epsilon[0]] = true
			}
		}
		return firsts
	}
}

// GenerateUniqueName generates a name for a non-terminal gauranteed to be
// unique within the grammar, based on original if one is provided.
func (g Grammar) GenerateUniqueName(original string) string {
	newName := original + "-P"
	existingRule := g.Rule(newName)
	for existingRule.NonTerminal != "" {
		newName += "P"
		existingRule = g.Rule(newName)
	}

	return newName
}

// parseRule parses a Rule from a string like "S -> X | Y"
func parseRule(r string) (Rule, error) {
	sides := strings.Split(r, "->")
	if len(sides) != 2 {
		return Rule{}, fmt.Errorf("not a rule of form 'NONTERM -> SYMBOL SYMBOL | SYMBOL ...': %q", r)
	}
	nonTerminal := strings.TrimSpace(sides[0])

	if nonTerminal == "" {
		return Rule{}, fmt.Errorf("empty nonterminal name not allowed for production rule")
	}

	// ensure that it isnt an illegal char, only things used should be 'A-Z',
	// '_', and '-'
	for _, ch := range nonTerminal {
		if ('A' > ch || ch > 'Z') && ch != '_' && ch != '-' {
			return Rule{}, fmt.Errorf("invalid nonterminal name %q; must only be chars A-Z, \"_\", or \"-\"", nonTerminal)
		}
	}

	parsedRule := Rule{NonTerminal: nonTerminal}

	productionsString := strings.TrimSpace(sides[1])
	prodStrings := strings.Split(productionsString, "|")
	for _, p := range prodStrings {
		parsedProd := Production{}
		// split by spaces
		p = strings.TrimSpace(p)
		symbols := strings.Split(p, " ")
		for _, sym := range symbols {
			sym = strings.TrimSpace(sym)

			if sym == "" {
				return Rule{}, fmt.Errorf("empty symbol not allowed")
			}

			if strings.ToLower(sym) == "ε" {
				// epsilon production
				parsedProd = Epsilon
				continue
			} else {
				// is it a terminal?
				isTerm := strings.ToLower(sym) == sym
				isNonTerm := strings.ToUpper(sym) == sym

				if !isTerm && !isNonTerm {
					return Rule{}, fmt.Errorf("cannot tell if symbol is a terminal or non-terminal: %q", sym)
				}

				for _, ch := range strings.ToLower(sym) {
					if ('a' > ch || ch > 'z') && ch != '_' && ch != '-' {
						return Rule{}, fmt.Errorf("invalid symbol: %q", sym)
					}
				}

				parsedProd = append(parsedProd, sym)
			}
		}

		parsedRule.Productions = append(parsedRule.Productions, parsedProd)
	}

	return parsedRule, nil
}

// mustParseRule is like parseRule but panics if it can't.
func mustParseRule(r string) Rule {
	rule, err := parseRule(r)
	if err != nil {
		panic(err.Error())
	}
	return rule
}

// removeEpsilons removes all epsilon-only productions from a list of
// productions and returns the result.
func removeEpsilons(from []Production) []Production {
	newProds := []Production{}

	for i := range from {
		if !from[i].Equal(Epsilon) {
			newProds = append(newProds, from[i])
		}
	}

	return newProds
}

func getEpsilonRewrites(epsilonableNonterm string, prod Production) []Production {
	// TODO: ensure that if the production consists of ONLY the epsilonable,
	// that we also are adding an epsilon production.

	// how many times does it occur?
	var numOccurances int
	for i := range prod {
		if prod[i] == epsilonableNonterm {
			numOccurances++
		}
	}

	if numOccurances == 0 {
		return []Production{prod}
	}

	// generate all numbers of that binary bitsize

	perms := int(math.Pow(2, float64(numOccurances)))

	// we're using the bitfield of above perms to denote which A should be "on"
	// and which should be "off" in the resulting string.

	newProds := []Production{}

	epsilonablePositions := make([]string, numOccurances)
	for i := perms - 1; i >= 0; i-- {
		// fill positions from the bitfield making up the cur permutation num
		for j := range epsilonablePositions {
			if ((i >> j) & 1) > 0 {
				epsilonablePositions[j] = epsilonableNonterm
			} else {
				epsilonablePositions[j] = ""
			}
		}

		// build a new production
		newProd := Production{}
		var curEpsilonable int
		for j := range prod {
			if prod[j] == epsilonableNonterm {
				pos := epsilonablePositions[curEpsilonable]
				if pos != "" {
					newProd = append(newProd, pos)
				}
				curEpsilonable++
			} else {
				newProd = append(newProd, prod[j])
			}
		}
		if len(newProd) == 0 {
			newProd = Epsilon
		}
		newProds = append(newProds, newProd)
	}

	// now eliminate every production that is a duplicate
	uniqueNewProds := []Production{}
	seenProductions := map[string]bool{}
	for i := range newProds {
		str := strings.Join(newProds[i], " ")

		if _, alreadySeen := seenProductions[str]; alreadySeen {
			continue
		}

		uniqueNewProds = append(uniqueNewProds, newProds[i])
		seenProductions[str] = true
	}

	return uniqueNewProds
}

// Validates that the current rules form a complete grammar with no
// missing definitions. TODO: should also dupe-check rules.
func (g Grammar) Validate() error {
	if g.rulesByName == nil {
		g.rulesByName = map[string]int{}
	}

	// a grammar needs at least one rule and at least one terminal or it makes
	// no sense.
	if len(g.rules) < 1 {
		return fmt.Errorf("no rules defined in grammar")
	} else if len(g.terminals) < 1 {
		return fmt.Errorf("no terminals defined in grammar")
	}

	producedNonTerms := map[string]bool{}
	producedTerms := map[string]bool{}

	// make sure all non-terminals produce either defined
	// non-terminals or defined terminals
	orderedTermKeys := util.OrderedKeys(g.terminals)

	errStr := ""

	for i := range g.rules {
		rule := g.rules[i]
		for _, alt := range rule.Productions {
			for _, sym := range alt {
				// if its empty its the empty non-terminal (episilon production) so skip
				if sym == "" {
					continue
				}
				if strings.ToUpper(sym) == sym {
					// non-terminal
					if _, ok := g.rulesByName[sym]; !ok {
						errStr += fmt.Sprintf("ERR: no production defined for nonterminal %q produced by %q\n", sym, rule.NonTerminal)
					}
					producedNonTerms[sym] = true
				} else {
					// terminal
					if _, ok := g.terminals[sym]; !ok {
						errStr += fmt.Sprintf("ERR: undefined terminal %q produced by %q\n", sym, rule.NonTerminal)
					}
					producedTerms[sym] = true
				}
			}
		}
	}

	// make sure every defined terminal is used and that each maps to a distinct
	// token class
	seenClasses := map[tokenClass]string{}
	for _, term := range orderedTermKeys {
		if _, ok := producedTerms[term]; !ok {
			errStr += fmt.Sprintf("ERR: terminal %q is not produced by any rule\n", term)
		}

		cl := g.terminals[term]
		if mappedBy, alreadySeen := seenClasses[cl]; alreadySeen {
			errStr += fmt.Sprintf("ERR: terminal %q maps to same class %q as terminal %q", term, cl.human, mappedBy)
		}
		seenClasses[cl] = term
	}

	// make sure every non-term is used
	for _, r := range g.rules {
		// S is used by default, don't check that one
		if r.NonTerminal == g.StartSymbol() {
			continue
		}

		if _, ok := producedNonTerms[r.NonTerminal]; !ok {
			errStr += fmt.Sprintf("ERR: non-terminal %q not produced by any rule\n", r.NonTerminal)
		}
	}

	// make sure we HAVE an S
	if _, ok := g.rulesByName[g.StartSymbol()]; !ok {
		errStr += fmt.Sprintf("ERR: no rules defined for productions of start symbol '%s'", g.StartSymbol())
	}

	if len(errStr) > 0 {
		// chop off trailing newline
		errStr = errStr[:len(errStr)-1]
		return fmt.Errorf(errStr)
	}

	return nil
}

var lang = Grammar{}

func init() {

	// add lang rules and terminals based off of CFG in docs/tscfg.md

	// production rules
	lang.AddRule("S", []string{"EXPR"})

	lang.AddRule("EXPR", []string{"BINARY-EXPR"})

	lang.AddRule("BINARY-EXPR", []string{"BINARY-SET-EXPR"})

	//	lang.AddRule("binary-separator-expr", []string{"binary-set-expr", tsSeparator.id, "binary-separator-expr"})
	//	lang.AddRule("binary-separator-expr", []string{"binary-set-expr"})

	lang.AddRule("BINARY-SET-EXPR", []string{"BINARY-SET-EXPR", strings.ToLower(tsOpSet.id), "BINARY-INCSET-EXPR"})
	lang.AddRule("BINARY-SET-EXPR", []string{"BINARY-INCSET-EXPR"})

	lang.AddRule("BINARY-INCSET-EXPR", []string{"BINARY-INCSET-EXPR", strings.ToLower(tsOpIncset.id), "BINARY-DECSET-EXPR"})
	lang.AddRule("BINARY-INCSET-EXPR", []string{"BINARY-DECSET-EXPR"})

	lang.AddRule("BINARY-DECSET-EXPR", []string{"BINARY-DECSET-EXPR", strings.ToLower(tsOpDecset.id), "BINARY-OR-EXPR"})
	lang.AddRule("BINARY-DECSET-EXPR", []string{"BINARY-OR-EXPR"})

	lang.AddRule("BINARY-OR-EXPR", []string{"BINARY-AND-EXPR", strings.ToLower(tsOpOr.id), "BINARY-OR-EXPR"})
	lang.AddRule("BINARY-OR-EXPR", []string{"BINARY-AND-EXPR"})

	lang.AddRule("BINARY-AND-EXPR", []string{"BINARY-EQ-EXPR", strings.ToLower(tsOpAnd.id), "BINARY-AND-EXPR"})
	lang.AddRule("BINARY-AND-EXPR", []string{"BINARY-EQ-EXPR"})

	lang.AddRule("BINARY-EQ-EXPR", []string{"BINARY-NE-EXPR", strings.ToLower(tsOpIs.id), "BINARY-EQ-EXPR"})
	lang.AddRule("BINARY-EQ-EXPR", []string{"BINARY-NE-EXPR"})

	lang.AddRule("BINARY-NE-EXPR", []string{"BINARY-LT-EXPR", strings.ToLower(tsOpIsNot.id), "BINARY-NE-EXPR"})
	lang.AddRule("BINARY-NE-EXPR", []string{"BINARY-LT-EXPR"})

	lang.AddRule("BINARY-LT-EXPR", []string{"BINARY-LE-EXPR", strings.ToLower(tsOpLessThan.id), "BINARY-LT-EXPR"})
	lang.AddRule("BINARY-LT-EXPR", []string{"BINARY-LE-EXPR"})

	lang.AddRule("BINARY-LE-EXPR", []string{"BINARY-GT-EXPR", strings.ToLower(tsOpLessThanIs.id), "BINARY-LE-EXPR"})
	lang.AddRule("BINARY-LE-EXPR", []string{"BINARY-GT-EXPR"})

	lang.AddRule("BINARY-GT-EXPR", []string{"BINARY-GE-EXPR", strings.ToLower(tsOpGreaterThan.id), "BINARY-GT-EXPR"})
	lang.AddRule("BINARY-GT-EXPR", []string{"BINARY-GE-EXPR"})

	lang.AddRule("BINARY-GE-EXPR", []string{"BINARY-ADD-EXPR", strings.ToLower(tsOpGreaterThanIs.id), "BINARY-GE-EXPR"})
	lang.AddRule("BINARY-GE-EXPR", []string{"BINARY-ADD-EXPR"})

	lang.AddRule("BINARY-ADD-EXPR", []string{"BINARY-SUBTRACT-EXPR", strings.ToLower(tsOpPlus.id), "BINARY-ADD-EXPR"})
	lang.AddRule("BINARY-ADD-EXPR", []string{"BINARY-SUBTRACT-EXPR"})

	lang.AddRule("BINARY-SUBTRACT-EXPR", []string{"BINARY-MULT-EXPR", strings.ToLower(tsOpMinus.id), "BINARY-SUBTRACT-EXPR"})
	lang.AddRule("BINARY-SUBTRACT-EXPR", []string{"BINARY-MULT-EXPR"})

	lang.AddRule("BINARY-MULT-EXPR", []string{"BINARY-DIV-EXPR", strings.ToLower(tsOpMultiply.id), "BINARY-MULT-EXPR"})
	lang.AddRule("BINARY-MULT-EXPR", []string{"BINARY-DIV-EXPR"})

	lang.AddRule("BINARY-DIV-EXPR", []string{"UNARY-EXPR", strings.ToLower(tsOpDivide.id), "BINARY-DIV-EXPR"})
	lang.AddRule("BINARY-DIV-EXPR", []string{"UNARY-EXPR"})

	lang.AddRule("UNARY-EXPR", []string{"UNARY-NOT-EXPR"})

	lang.AddRule("UNARY-NOT-EXPR", []string{strings.ToLower(tsOpNot.id), "UNARY-NEGATE-EXPR"})
	lang.AddRule("UNARY-NOT-EXPR", []string{"UNARY-NEGATE-EXPR"})

	lang.AddRule("UNARY-NEGATE-EXPR", []string{strings.ToLower(tsOpMinus.id), "UNARY-INC-EXPR"})
	lang.AddRule("UNARY-NEGATE-EXPR", []string{"UNARY-INC-EXPR"})

	lang.AddRule("UNARY-INC-EXPR", []string{"UNARY-DEC-EXPR", strings.ToLower(tsOpInc.id)})
	lang.AddRule("UNARY-INC-EXPR", []string{"UNARY-DEC-EXPR"})

	lang.AddRule("UNARY-DEC-EXPR", []string{"EXPR-GROUP", strings.ToLower(tsOpDec.id)})
	lang.AddRule("UNARY-DEC-EXPR", []string{"EXPR-GROUP"})

	lang.AddRule("EXPR-GROUP", []string{strings.ToLower(tsGroupOpen.id), "EXPR", strings.ToLower(tsGroupClose.id)})
	lang.AddRule("EXPR-GROUP", []string{"IDENTIFIED-OBJ"})
	lang.AddRule("EXPR-GROUP", []string{"LITERAL"})

	lang.AddRule("IDENTIFIED-OBJ", []string{strings.ToLower(tsIdentifier.id), strings.ToLower(tsGroupOpen.id), "ARG-LIST", strings.ToLower(tsGroupClose.id)})
	lang.AddRule("IDENTIFIED-OBJ", []string{strings.ToLower(tsIdentifier.id)})

	lang.AddRule("ARG-LIST", []string{"EXPR", strings.ToLower(tsSeparator.id), "ARG-LIST"})
	lang.AddRule("ARG-LIST", []string{"EXPR"})
	lang.AddRule("ARG-LIST", []string{""})

	lang.AddRule("LITERAL", []string{strings.ToLower(tsBool.id)})
	lang.AddRule("LITERAL", []string{strings.ToLower(tsNumber.id)})
	lang.AddRule("LITERAL", []string{strings.ToLower(tsUnquotedString.id)})
	lang.AddRule("LITERAL", []string{strings.ToLower(tsQuotedString.id)})

	// terminals
	lang.AddTerm(strings.ToLower(tsBool.id), tsBool)
	lang.AddTerm(strings.ToLower(tsGroupClose.id), tsGroupClose)
	lang.AddTerm(strings.ToLower(tsGroupOpen.id), tsGroupOpen)
	lang.AddTerm(strings.ToLower(tsSeparator.id), tsSeparator)
	lang.AddTerm(strings.ToLower(tsIdentifier.id), tsIdentifier)
	lang.AddTerm(strings.ToLower(tsNumber.id), tsNumber)
	lang.AddTerm(strings.ToLower(tsQuotedString.id), tsQuotedString)
	lang.AddTerm(strings.ToLower(tsUnquotedString.id), tsUnquotedString)
	lang.AddTerm(strings.ToLower(tsOpAnd.id), tsOpAnd)
	lang.AddTerm(strings.ToLower(tsOpDec.id), tsOpDec)
	lang.AddTerm(strings.ToLower(tsOpDecset.id), tsOpDecset)
	lang.AddTerm(strings.ToLower(tsOpDivide.id), tsOpDivide)
	lang.AddTerm(strings.ToLower(tsOpGreaterThan.id), tsOpGreaterThan)
	lang.AddTerm(strings.ToLower(tsOpGreaterThanIs.id), tsOpGreaterThanIs)
	lang.AddTerm(strings.ToLower(tsOpInc.id), tsOpInc)
	lang.AddTerm(strings.ToLower(tsOpIncset.id), tsOpIncset)
	lang.AddTerm(strings.ToLower(tsOpIs.id), tsOpIs)
	lang.AddTerm(strings.ToLower(tsOpIsNot.id), tsOpIsNot)
	lang.AddTerm(strings.ToLower(tsOpLessThan.id), tsOpLessThan)
	lang.AddTerm(strings.ToLower(tsOpLessThanIs.id), tsOpLessThanIs)
	lang.AddTerm(strings.ToLower(tsOpMinus.id), tsOpMinus)
	lang.AddTerm(strings.ToLower(tsOpMultiply.id), tsOpMultiply)
	lang.AddTerm(strings.ToLower(tsOpNot.id), tsOpNot)
	lang.AddTerm(strings.ToLower(tsOpOr.id), tsOpOr)
	lang.AddTerm(strings.ToLower(tsOpPlus.id), tsOpPlus)
	lang.AddTerm(strings.ToLower(tsOpSet.id), tsOpSet)

	err := lang.Validate()
	if err != nil {
		panic(fmt.Sprintf("malformed CFG definition: %s", err.Error()))
	}
}
