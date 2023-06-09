package fe

/*
File automatically generated by the ictiobus compiler. DO NOT EDIT. This was
created by invoking ictiobus with the following command:

    ictcc --slr -l TunaScript -v 1.0 -d tsi --ir github.com/dekarrin/tunaq/tunascript/syntax.AST --hooks ./tunascript/syntax --dest ./tunascript/fe tunascript/tunascript.md --sim-off
*/

import (
	"github.com/dekarrin/ictiobus"
	"github.com/dekarrin/ictiobus/trans"

	"fmt"
	"strings"
)

// SDTS returns the generated ictiobus syntax-directed translation scheme for
// TunaScript.
func SDTS() trans.SDTS {
	sdts := ictiobus.NewSDTS()

	sdtsBindTCTunascript(sdts)
	sdtsBindTCExpr(sdts)
	sdtsBindTCBoolOp(sdts)
	sdtsBindTCEquality(sdts)
	sdtsBindTCComparison(sdts)
	sdtsBindTCSum(sdts)
	sdtsBindTCProduct(sdts)
	sdtsBindTCNegation(sdts)
	sdtsBindTCTerm(sdts)
	sdtsBindTCArgList(sdts)
	sdtsBindTCArgs(sdts)
	sdtsBindTCValue(sdts)

	return sdts
}

func sdtsBindTCTunascript(sdts trans.SDTS) {
	var err error
	err = sdts.Bind(
		"TUNASCRIPT", []string{"EXPR"},
		"ast",
		"ast",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"EXPR"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "TUNASCRIPT", prodStr, err.Error()))
	}
}

func sdtsBindTCExpr(sdts trans.SDTS) {
	var err error
	err = sdts.Bind(
		"EXPR", []string{"id", "set", "EXPR"},
		"node",
		"assign_set",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "$text"},
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 2}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"id", "set", "EXPR"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "EXPR", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"EXPR", []string{"id", "+=", "EXPR"},
		"node",
		"assign_incset",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "$text"},
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 2}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"id", "+=", "EXPR"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "EXPR", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"EXPR", []string{"id", "-=", "EXPR"},
		"node",
		"assign_decset",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "$text"},
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 2}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"id", "-=", "EXPR"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "EXPR", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"EXPR", []string{"BOOL-OP"},
		"node",
		"identity",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"BOOL-OP"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "EXPR", prodStr, err.Error()))
	}
}

func sdtsBindTCBoolOp(sdts trans.SDTS) {
	var err error
	err = sdts.Bind(
		"BOOL-OP", []string{"BOOL-OP", "or", "EQUALITY"},
		"node",
		"bin_or",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 2}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"BOOL-OP", "or", "EQUALITY"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "BOOL-OP", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"BOOL-OP", []string{"BOOL-OP", "and", "EQUALITY"},
		"node",
		"bin_and",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 2}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"BOOL-OP", "and", "EQUALITY"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "BOOL-OP", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"BOOL-OP", []string{"EQUALITY"},
		"node",
		"identity",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"EQUALITY"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "BOOL-OP", prodStr, err.Error()))
	}
}

func sdtsBindTCEquality(sdts trans.SDTS) {
	var err error
	err = sdts.Bind(
		"EQUALITY", []string{"EQUALITY", "eq", "COMPARISON"},
		"node",
		"bin_eq",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 2}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"EQUALITY", "eq", "COMPARISON"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "EQUALITY", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"EQUALITY", []string{"EQUALITY", "ne", "COMPARISON"},
		"node",
		"bin_ne",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 2}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"EQUALITY", "ne", "COMPARISON"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "EQUALITY", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"EQUALITY", []string{"COMPARISON"},
		"node",
		"identity",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"COMPARISON"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "EQUALITY", prodStr, err.Error()))
	}
}

func sdtsBindTCComparison(sdts trans.SDTS) {
	var err error
	err = sdts.Bind(
		"COMPARISON", []string{"COMPARISON", "<", "SUM"},
		"node",
		"bin_lt",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 2}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"COMPARISON", "<", "SUM"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "COMPARISON", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"COMPARISON", []string{"COMPARISON", ">", "SUM"},
		"node",
		"bin_gt",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 2}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"COMPARISON", ">", "SUM"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "COMPARISON", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"COMPARISON", []string{"COMPARISON", "<=", "SUM"},
		"node",
		"bin_le",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 2}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"COMPARISON", "<=", "SUM"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "COMPARISON", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"COMPARISON", []string{"COMPARISON", ">=", "SUM"},
		"node",
		"bin_ge",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 2}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"COMPARISON", ">=", "SUM"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "COMPARISON", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"COMPARISON", []string{"SUM"},
		"node",
		"identity",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"SUM"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "COMPARISON", prodStr, err.Error()))
	}
}

func sdtsBindTCSum(sdts trans.SDTS) {
	var err error
	err = sdts.Bind(
		"SUM", []string{"SUM", "+", "PRODUCT"},
		"node",
		"bin_add",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 2}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"SUM", "+", "PRODUCT"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "SUM", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"SUM", []string{"SUM", "-", "PRODUCT"},
		"node",
		"bin_sub",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 2}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"SUM", "-", "PRODUCT"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "SUM", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"SUM", []string{"PRODUCT"},
		"node",
		"identity",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"PRODUCT"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "SUM", prodStr, err.Error()))
	}
}

func sdtsBindTCProduct(sdts trans.SDTS) {
	var err error
	err = sdts.Bind(
		"PRODUCT", []string{"PRODUCT", "*", "NEGATION"},
		"node",
		"bin_mult",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 2}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"PRODUCT", "*", "NEGATION"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "PRODUCT", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"PRODUCT", []string{"PRODUCT", "/", "NEGATION"},
		"node",
		"bin_div",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 2}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"PRODUCT", "/", "NEGATION"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "PRODUCT", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"PRODUCT", []string{"NEGATION"},
		"node",
		"identity",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"NEGATION"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "PRODUCT", prodStr, err.Error()))
	}
}

func sdtsBindTCNegation(sdts trans.SDTS) {
	var err error
	err = sdts.Bind(
		"NEGATION", []string{"!", "NEGATION"},
		"node",
		"unary_not",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 1}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"!", "NEGATION"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "NEGATION", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"NEGATION", []string{"-", "NEGATION"},
		"node",
		"unary_neg",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 1}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"-", "NEGATION"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "NEGATION", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"NEGATION", []string{"TERM"},
		"node",
		"identity",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"TERM"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "NEGATION", prodStr, err.Error()))
	}
}

func sdtsBindTCTerm(sdts trans.SDTS) {
	var err error
	err = sdts.Bind(
		"TERM", []string{"lp", "EXPR", "rp"},
		"node",
		"group",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 1}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"lp", "EXPR", "rp"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "TERM", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"TERM", []string{"id", "ARG-LIST"},
		"node",
		"func",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "$text"},
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 1}, Name: "args"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"id", "ARG-LIST"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "TERM", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"TERM", []string{"VALUE"},
		"node",
		"identity",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"VALUE"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "TERM", prodStr, err.Error()))
	}
}

func sdtsBindTCArgList(sdts trans.SDTS) {
	var err error
	err = sdts.Bind(
		"ARG-LIST", []string{"lp", "ARGS", "rp"},
		"args",
		"identity",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 1}, Name: "args"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"lp", "ARGS", "rp"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "ARG-LIST", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"ARG-LIST", []string{"lp", "rp"},
		"args",
		"args_list",
		nil,
	)
	if err != nil {
		prodStr := strings.Join([]string{"lp", "rp"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "ARG-LIST", prodStr, err.Error()))
	}
}

func sdtsBindTCArgs(sdts trans.SDTS) {
	var err error
	err = sdts.Bind(
		"ARGS", []string{"ARGS", "comma", "EXPR"},
		"args",
		"args_list",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 2}, Name: "node"},
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "args"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"ARGS", "comma", "EXPR"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "ARGS", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"ARGS", []string{"EXPR"},
		"args",
		"args_list",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "node"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"EXPR"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "ARGS", prodStr, err.Error()))
	}
}

func sdtsBindTCValue(sdts trans.SDTS) {
	var err error
	err = sdts.Bind(
		"VALUE", []string{"id", "++"},
		"node",
		"assign_inc",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "$text"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"id", "++"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "VALUE", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"VALUE", []string{"id", "--"},
		"node",
		"assign_dec",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "$text"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"id", "--"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "VALUE", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"VALUE", []string{"id"},
		"node",
		"flag",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "$text"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"id"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "VALUE", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"VALUE", []string{"num"},
		"node",
		"lit_num",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "$text"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"num"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "VALUE", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"VALUE", []string{"@str"},
		"node",
		"lit_text",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "$text"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"@str"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "VALUE", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"VALUE", []string{"str"},
		"node",
		"lit_text",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "$text"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"str"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "VALUE", prodStr, err.Error()))
	}

	err = sdts.Bind(
		"VALUE", []string{"bool"},
		"node",
		"lit_binary",
		[]trans.AttrRef{
			{Rel: trans.NodeRelation{Type: trans.RelSymbol, Index: 0}, Name: "$text"},
		},
	)
	if err != nil {
		prodStr := strings.Join([]string{"bool"}, " ")
		panic(fmt.Sprintf("binding %s -> [%s]: %s", "VALUE", prodStr, err.Error()))
	}
}
