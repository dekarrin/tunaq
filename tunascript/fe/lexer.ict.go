package fe

/*
File automatically generated by the ictiobus compiler. DO NOT EDIT. This was
created by invoking ictiobus with the following command:

    ictcc --slr -l TunaScript -v 1.0 -d tsi --ir github.com/dekarrin/tunaq/tunascript/syntax.AST --hooks ./tunascript/syntax --dest ./tunascript/fe tunascript/tunascript.md --sim-off
*/

import (
	"github.com/dekarrin/ictiobus"
	"github.com/dekarrin/ictiobus/lex"

	"github.com/dekarrin/tunaq/tunascript/fe/fetoken"
)

// Lexer returns the generated ictiobus Lexer for TunaScript.
func Lexer(lazy bool) lex.Lexer {
	var lx lex.Lexer
	if lazy {
		lx = ictiobus.NewLazyLexer()
	} else {
		lx = ictiobus.NewLexer()
	}

	// default state, shared by all
	lx.RegisterClass(fetoken.TCEqualsSign, "")
	lx.RegisterClass(fetoken.TCPlusSignequalsSign, "")
	lx.RegisterClass(fetoken.TCLessThanSignequalsSign, "")
	lx.RegisterClass(fetoken.TCLessThanSign, "")
	lx.RegisterClass(fetoken.TCGreaterThanSignequalsSign, "")
	lx.RegisterClass(fetoken.TCGreaterThanSign, "")
	lx.RegisterClass(fetoken.TCNe, "")
	lx.RegisterClass(fetoken.TCEq, "")
	lx.RegisterClass(fetoken.TCSet, "")
	lx.RegisterClass(fetoken.TCPlusSignplusSign, "")
	lx.RegisterClass(fetoken.TCPlusSign, "")
	lx.RegisterClass(fetoken.TCHyphenMinushyphenMinus, "")
	lx.RegisterClass(fetoken.TCHyphenMinus, "")
	lx.RegisterClass(fetoken.TCAsterisk, "")
	lx.RegisterClass(fetoken.TCSolidus, "")
	lx.RegisterClass(fetoken.TCExclamationMark, "")
	lx.RegisterClass(fetoken.TCAnd, "")
	lx.RegisterClass(fetoken.TCOr, "")
	lx.RegisterClass(fetoken.TCComma, "")
	lx.RegisterClass(fetoken.TCLp, "")
	lx.RegisterClass(fetoken.TCRp, "")
	lx.RegisterClass(fetoken.TCId, "")
	lx.RegisterClass(fetoken.TCBool, "")
	lx.RegisterClass(fetoken.TCNum, "")
	lx.RegisterClass(fetoken.TCCommercialAtstr, "")
	lx.RegisterClass(fetoken.TCStr, "")

	lx.AddPattern(`-=`, lex.LexAs(fetoken.TCEqualsSign.ID()), "", 0)
	lx.AddPattern(`\+=`, lex.LexAs(fetoken.TCPlusSignequalsSign.ID()), "", 0)
	lx.AddPattern(`<=`, lex.LexAs(fetoken.TCLessThanSignequalsSign.ID()), "", 0)
	lx.AddPattern(`<`, lex.LexAs(fetoken.TCLessThanSign.ID()), "", 0)
	lx.AddPattern(`>=`, lex.LexAs(fetoken.TCGreaterThanSignequalsSign.ID()), "", 0)
	lx.AddPattern(`>`, lex.LexAs(fetoken.TCGreaterThanSign.ID()), "", 0)
	lx.AddPattern(`!=`, lex.LexAs(fetoken.TCNe.ID()), "", 0)
	lx.AddPattern(`==`, lex.LexAs(fetoken.TCEq.ID()), "", 0)
	lx.AddPattern(`=`, lex.LexAs(fetoken.TCSet.ID()), "", 0)
	lx.AddPattern(`\+\+`, lex.LexAs(fetoken.TCPlusSignplusSign.ID()), "", 0)
	lx.AddPattern(`\+`, lex.LexAs(fetoken.TCPlusSign.ID()), "", 0)
	lx.AddPattern(`--`, lex.LexAs(fetoken.TCHyphenMinushyphenMinus.ID()), "", 0)
	lx.AddPattern(`-`, lex.LexAs(fetoken.TCHyphenMinus.ID()), "", 0)
	lx.AddPattern(`\*`, lex.LexAs(fetoken.TCAsterisk.ID()), "", 0)
	lx.AddPattern(`/`, lex.LexAs(fetoken.TCSolidus.ID()), "", 0)
	lx.AddPattern(`!`, lex.LexAs(fetoken.TCExclamationMark.ID()), "", 0)
	lx.AddPattern(`&&`, lex.LexAs(fetoken.TCAnd.ID()), "", 0)
	lx.AddPattern(`\|\|`, lex.LexAs(fetoken.TCOr.ID()), "", 0)
	lx.AddPattern(`,`, lex.LexAs(fetoken.TCComma.ID()), "", 0)
	lx.AddPattern(`\(`, lex.LexAs(fetoken.TCLp.ID()), "", 0)
	lx.AddPattern(`\)`, lex.LexAs(fetoken.TCRp.ID()), "", 0)
	lx.AddPattern(`\$[A-Za-z0-9_]+`, lex.LexAs(fetoken.TCId.ID()), "", 0)
	lx.AddPattern(`[Tt][Rr][Uu][Ee]|[Ff][Aa][Ll][Ss][Ee]`, lex.LexAs(fetoken.TCBool.ID()), "", 0)
	lx.AddPattern(`[Oo][Nn]|[Oo][Ff][Ff]`, lex.LexAs(fetoken.TCBool.ID()), "", 0)
	lx.AddPattern(`[Yy][Ee][Ss]|[Nn][Oo]`, lex.LexAs(fetoken.TCBool.ID()), "", 0)
	lx.AddPattern(`(?:\d+(?:\.\d*)?|\.\d+)(?:[Ee]-?\d+)?`, lex.LexAs(fetoken.TCNum.ID()), "", 0)
	lx.AddPattern(`@(?:\\.|[^@\\])*@`, lex.LexAs(fetoken.TCCommercialAtstr.ID()), "", 0)
	lx.AddPattern(`(?:\\.|\S)(?:(?:\\.|[^@,+<>!=*/&|()$-])*(?:\\.|[^\s@,+<>!=*/&|()$-]))?`, lex.LexAs(fetoken.TCStr.ID()), "", 0)
	lx.AddPattern(`\s+`, lex.Discard(), "", 0)

	return lx
}
