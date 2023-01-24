package tunascript

import (
	"fmt"
	"math"
	"strings"

	"github.com/dekarrin/tunaq/internal/util"
)

type Production []string

// Equal returns whether Rule is equal to another value. It will not be equal
// if the other value cannot be cast to a []string or *[]string.
func (p Production) Equal(o any) bool {
	other, ok := o.([]string)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*[]string)
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

// HasEpsilonProduction is shorthand for HasProduction([]string{""})
func (r Rule) HasEpsilonProduction() bool {
	return r.HasProduction([]string{""})
}

// HasProduction returns whether the rule has a production of the exact sequence
// of symbols entirely.
func (r Rule) HasProduction(prod []string) bool {
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

// Grammar for tunascript language, used by a parsing algorithm to create a
// parse tree from some input.
type Grammar struct {
	rulesByName map[string]int

	// main rules store, not just doing a simple map bc
	// rules may have order that matters
	rules     []Rule
	terminals map[string]tokenClass
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
		if ('A' > ch || ch > 'Z') && ch != '_' && ch != '-' {
			panic("terminal name must only be chars A-Z, \"_\", or \"-\"")
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
	if nonterminal != "S" {
		for _, ch := range nonterminal {
			if ('a' > ch || ch > 'z') && ch != '_' && ch != '-' {
				panic("nonterminal name must only be chars a-z, \"_\", \"-\", or else the start symbol \"S\"")
			}
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

// RemoveEpsilons returns a grammar that derives strings equivalent to the first
// one (with the exception of the empty string) but with all epsilon productions
// automatically eliminated.
//
// Call Validate before this or it may go poorly.
func (g Grammar) RemoveEpsilons() Grammar {
	// run this in a loop until all vars have episilon propagated out

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
			if B == A {
				// unit production cycle. will be addressed in later funcs
				continue
			}

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
	// special case, if it occurs exactly once as a unit production we must keep
	// both the unit AND add an epsilon production
	if len(prod) == 1 && prod[0] == epsilonableNonterm {
		return []Production{prod, {epsilonableNonterm}}
	}

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
	for i := 0; i < perms; i++ {
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
				if strings.ToLower(sym) == sym || sym == "S" {
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
	lang.AddRule("S", []string{"expr"})

	lang.AddRule("expr", []string{"binary-expr"})

	lang.AddRule("binary-expr", []string{"binary-set-expr"})

	//	lang.AddRule("binary-separator-expr", []string{"binary-set-expr", tsSeparator.id, "binary-separator-expr"})
	//	lang.AddRule("binary-separator-expr", []string{"binary-set-expr"})

	lang.AddRule("binary-set-expr", []string{"binary-set-expr", tsOpSet.id, "binary-incset-expr"})
	lang.AddRule("binary-set-expr", []string{"binary-incset-expr"})

	lang.AddRule("binary-incset-expr", []string{"binary-incset-expr", tsOpIncset.id, "binary-decset-expr"})
	lang.AddRule("binary-incset-expr", []string{"binary-decset-expr"})

	lang.AddRule("binary-decset-expr", []string{"binary-decset-expr", tsOpDecset.id, "binary-or-expr"})
	lang.AddRule("binary-decset-expr", []string{"binary-or-expr"})

	lang.AddRule("binary-or-expr", []string{"binary-and-expr", tsOpOr.id, "binary-or-expr"})
	lang.AddRule("binary-or-expr", []string{"binary-and-expr"})

	lang.AddRule("binary-and-expr", []string{"binary-eq-expr", tsOpAnd.id, "binary-and-expr"})
	lang.AddRule("binary-and-expr", []string{"binary-eq-expr"})

	lang.AddRule("binary-eq-expr", []string{"binary-ne-expr", tsOpIs.id, "binary-eq-expr"})
	lang.AddRule("binary-eq-expr", []string{"binary-ne-expr"})

	lang.AddRule("binary-ne-expr", []string{"binary-lt-expr", tsOpIsNot.id, "binary-ne-expr"})
	lang.AddRule("binary-ne-expr", []string{"binary-lt-expr"})

	lang.AddRule("binary-lt-expr", []string{"binary-le-expr", tsOpLessThan.id, "binary-lt-expr"})
	lang.AddRule("binary-lt-expr", []string{"binary-le-expr"})

	lang.AddRule("binary-le-expr", []string{"binary-gt-expr", tsOpLessThanIs.id, "binary-le-expr"})
	lang.AddRule("binary-le-expr", []string{"binary-gt-expr"})

	lang.AddRule("binary-gt-expr", []string{"binary-ge-expr", tsOpGreaterThan.id, "binary-gt-expr"})
	lang.AddRule("binary-gt-expr", []string{"binary-ge-expr"})

	lang.AddRule("binary-ge-expr", []string{"binary-add-expr", tsOpGreaterThanIs.id, "binary-ge-expr"})
	lang.AddRule("binary-ge-expr", []string{"binary-add-expr"})

	lang.AddRule("binary-add-expr", []string{"binary-subtract-expr", tsOpPlus.id, "binary-add-expr"})
	lang.AddRule("binary-add-expr", []string{"binary-subtract-expr"})

	lang.AddRule("binary-subtract-expr", []string{"binary-mult-expr", tsOpMinus.id, "binary-subtract-expr"})
	lang.AddRule("binary-subtract-expr", []string{"binary-mult-expr"})

	lang.AddRule("binary-mult-expr", []string{"binary-div-expr", tsOpMultiply.id, "binary-mult-expr"})
	lang.AddRule("binary-mult-expr", []string{"binary-div-expr"})

	lang.AddRule("binary-div-expr", []string{"unary-expr", tsOpDivide.id, "binary-div-expr"})
	lang.AddRule("binary-div-expr", []string{"unary-expr"})

	lang.AddRule("unary-expr", []string{"unary-not-expr"})

	lang.AddRule("unary-not-expr", []string{tsOpNot.id, "unary-negate-expr"})
	lang.AddRule("unary-not-expr", []string{"unary-negate-expr"})

	lang.AddRule("unary-negate-expr", []string{tsOpMinus.id, "unary-inc-expr"})
	lang.AddRule("unary-negate-expr", []string{"unary-inc-expr"})

	lang.AddRule("unary-inc-expr", []string{"unary-dec-expr", tsOpInc.id})
	lang.AddRule("unary-inc-expr", []string{"unary-dec-expr"})

	lang.AddRule("unary-dec-expr", []string{"expr-group", tsOpDec.id})
	lang.AddRule("unary-dec-expr", []string{"expr-group"})

	lang.AddRule("expr-group", []string{tsGroupOpen.id, "expr", tsGroupClose.id})
	lang.AddRule("expr-group", []string{"identified-obj"})
	lang.AddRule("expr-group", []string{"literal"})

	lang.AddRule("identified-obj", []string{tsIdentifier.id, tsGroupOpen.id, "arg-list", tsGroupClose.id})
	lang.AddRule("identified-obj", []string{tsIdentifier.id})

	lang.AddRule("arg-list", []string{"expr", tsSeparator.id, "arg-list"})
	lang.AddRule("arg-list", []string{"expr"})
	lang.AddRule("arg-list", []string{""})

	lang.AddRule("literal", []string{tsBool.id})
	lang.AddRule("literal", []string{tsNumber.id})
	lang.AddRule("literal", []string{tsUnquotedString.id})
	lang.AddRule("literal", []string{tsQuotedString.id})

	// terminals
	lang.AddTerm(tsBool.id, tsBool)
	lang.AddTerm(tsGroupClose.id, tsGroupClose)
	lang.AddTerm(tsGroupOpen.id, tsGroupOpen)
	lang.AddTerm(tsSeparator.id, tsSeparator)
	lang.AddTerm(tsIdentifier.id, tsIdentifier)
	lang.AddTerm(tsNumber.id, tsNumber)
	lang.AddTerm(tsQuotedString.id, tsQuotedString)
	lang.AddTerm(tsUnquotedString.id, tsUnquotedString)
	lang.AddTerm(tsOpAnd.id, tsOpAnd)
	lang.AddTerm(tsOpDec.id, tsOpDec)
	lang.AddTerm(tsOpDecset.id, tsOpDecset)
	lang.AddTerm(tsOpDivide.id, tsOpDivide)
	lang.AddTerm(tsOpGreaterThan.id, tsOpGreaterThan)
	lang.AddTerm(tsOpGreaterThanIs.id, tsOpGreaterThanIs)
	lang.AddTerm(tsOpInc.id, tsOpInc)
	lang.AddTerm(tsOpIncset.id, tsOpIncset)
	lang.AddTerm(tsOpIs.id, tsOpIs)
	lang.AddTerm(tsOpIsNot.id, tsOpIsNot)
	lang.AddTerm(tsOpLessThan.id, tsOpLessThan)
	lang.AddTerm(tsOpLessThanIs.id, tsOpLessThanIs)
	lang.AddTerm(tsOpMinus.id, tsOpMinus)
	lang.AddTerm(tsOpMultiply.id, tsOpMultiply)
	lang.AddTerm(tsOpNot.id, tsOpNot)
	lang.AddTerm(tsOpOr.id, tsOpOr)
	lang.AddTerm(tsOpPlus.id, tsOpPlus)
	lang.AddTerm(tsOpSet.id, tsOpSet)

	err := lang.Validate()
	if err != nil {
		panic(fmt.Sprintf("malformed CFG definition: %s", err.Error()))
	}
}
