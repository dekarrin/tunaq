package tunascript

import (
	"fmt"
	"math"
	"strings"

	"github.com/dekarrin/tunaq/internal/util"
)

type Production []string

// Equal returns whether Rule is equal to another value. It will not be equal
// if the other value cannot be cast to Production or *Production.
func (p Production) Equal(o any) bool {
	other, ok := o.(Production)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*Production)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
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
	if len(p) == 1 && p[0] == "" {
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
	return len(p) == 1 && p[0] != "" && strings.ToUpper(p[0]) == p[0]
}

// terminals will be upper, non-terms will be lower. 'S' is reserved for use as
// the start symbol.
type Rule struct {
	NonTerminal string
	Productions []Production
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

// HasEpsilonProduction is shorthand for HasProduction(Production{""})
func (r Rule) HasEpsilonProduction() bool {
	return r.HasProduction(Production{""})
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

type RuleSet []Rule

// Grammar for tunascript language, used by a parsing algorithm to create a
// parse tree from some input.
type Grammar struct {
	rulesByName map[string]int

	// main rules store, not just doing a simple map bc
	// rules may have order that matters
	rules     []Rule
	terminals map[string]tokenClass
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
			panic(fmt.Sprintf("invalid nonterminal name %q; must only be chars A-Z, \"_\", \"-\", or else the start symbol \"S\"", nonterminal))
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

// NonTerminals returns list of all the non-terminal symbols. All will be lower
// case with the exception of the start symbol S.
func (g Grammar) NonTerminals() []string {
	return util.OrderedKeys(g.rulesByName)
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
		if nonTerm == "S" {
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

// UnreachableNonTerminals returns all non-terminals (excluding the start symbol
// "S") that are currently unreachable due to not being produced by any other
// grammar rule.
func (g Grammar) UnreachableNonTerminals() []string {
	unreachables := []string{}

	for _, nonTerm := range g.NonTerminals() {
		if nonTerm == "S" {
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
	/*// old algo:
	prods := g.UnitProductions()

	for _, ruleWithUnitProds := range prods {
		fullRule := g.Rule(ruleWithUnitProds.NonTerminal)

		for _, unitProd := range ruleWithUnitProds.Productions {

			// hoist all productions of that symbol up.

			// by definition unit production has exactly one non-terminal and
			nonTermToHoist := unitProd[0]
			hoistedRule := g.Rule(nonTermToHoist)
			fullRule = fullRule.ReplaceProduction(unitProd, hoistedRule.Productions...)
		}

		g.rules[g.rulesByName[fullRule.NonTerminal]] = fullRule
	}*/

	// okay, now just remove the unreachable ones (not strictly necessary for
	// all interpretations of unit production removal but lets do it anyways for
	// simplicity)
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

			if rule.HasEpsilonProduction() {
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
					var newProd []string
					if len(bProd) == 1 && bProd[0] == A {
						newProd = append(newProd, "")
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
			return Rule{}, fmt.Errorf("invalid nonterminal name %q; must only be chars A-Z, \"_\", \"-\", or else the start symbol \"S\"", nonTerminal)
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
				parsedProd = Production{""}
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
		if len(from[i]) != 1 || from[i][0] != "" {
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
		newProd := []string{}
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
			newProd = []string{""}
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
		if r.NonTerminal == "S" {
			continue
		}

		if _, ok := producedNonTerms[r.NonTerminal]; !ok {
			errStr += fmt.Sprintf("ERR: non-terminal %q not produced by any rule\n", r.NonTerminal)
		}
	}

	// make sure we HAVE an S
	if _, ok := g.rulesByName["S"]; !ok {
		errStr += "ERR: no rules defined for productions of start symbol 'S'"
	}

	if len(errStr) > 0 {
		// chop off trailing newline
		errStr = errStr[:len(errStr)-1]
		return fmt.Errorf(errStr)
	}

	return nil
}

type parseTree struct {
	terminal bool
	value    string
	children []parseTree
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
