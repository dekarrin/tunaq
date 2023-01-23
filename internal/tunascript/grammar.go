package tunascript

import (
	"fmt"
	"strings"

	"github.com/dekarrin/tunaq/internal/util"
)

// terminals will be upper, non-terms will be lower. 'S' is reserved for use as
// the start symbol.
type Rule struct {
	NonTerminal string
	Productions [][]string
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

	if terminal == "S" {
		panic("start non-terminal 'S' cannot be used as a terminal")
	}

	if strings.ToUpper(terminal) != terminal {
		panic("terminal must be all uppercase")
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
func (g *Grammar) AddRule(nonterminal string, production []string) {
	if nonterminal == "" {
		panic("empty nonterminal name not allowed for production rule")
	}
	if strings.ToLower(nonterminal) != nonterminal && nonterminal != "S" {
		panic("nonterminal must be all lowercase or special start symbole \"S\"")
	}

	if len(production) < 1 {
		panic("for epsilon production give empty string; all rules must have productions")
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

// Validates that the current rules form a complete grammar with no
// missing definitions.
func (g Grammar) Validate() error {
	if g.rulesByName == nil {
		g.rulesByName = map[string]int{}
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
				if sym == "S" || strings.ToLower(sym) == sym {
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
		if _, ok := producedNonTerms[r.NonTerminal]; !ok {
			errStr += fmt.Sprintf("ERR: non-terminal %q not produced by any rule\n", r.NonTerminal)
		}
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

	lang.AddRule("binary-decset-expr", []string{"binary-decset-expr", tsOpDec.id, "binary-or-expr"})
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

	lang.AddRule("unary-not-expr", []string{tsOpNot.id, "unary-negat-expr"})
	lang.AddRule("unary-not-expr", []string{"unary-negate-expr"})

	lang.AddRule("unary-negate-expr", []string{tsOpMinus.id, "unary-inc-expr"})
	lang.AddRule("unary-negate-expr", []string{"unary-inc-expr"})

	lang.AddRule("unary-inc-expr", []string{"unary-dec-expr", tsOpInc.id})
	lang.AddRule("unary-inc-expr", []string{"unary-dec-expr"})

	lang.AddRule("unary-dec-expr", []string{"expr-group", tsOpDec.id})
	lang.AddRule("unary-dec-expr", []string{"unary-group"})

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
