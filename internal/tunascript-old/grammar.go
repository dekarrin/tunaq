package tunascript

import (
	"fmt"
	"strings"

	"github.com/dekarrin/tunaq/internal/ictiobus/grammar"
)

var lang = grammar.Grammar{}

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
