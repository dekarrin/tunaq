package syntax

import (
	"strings"
	"unicode"

	"github.com/dekarrin/ictiobus/trans"
)

func makeConstHook(v interface{}) trans.Hook {
	return func(info trans.SetterInfo, args []interface{}) (interface{}, error) {
		return v, nil
	}
}

var (
	ExpHooksTable = trans.HookMap{
		"identity":         func(info trans.SetterInfo, args []interface{}) (interface{}, error) { return args[0], nil },
		"ast":              expHookAST,
		"text":             expHookText,
		"flag":             expHookFlag,
		"branch":           expHookBranch,
		"branch_with_else": expHookBranchWithElse,
		"cond_list":        expHookCondList,
		"node_list":        expHookNodeList,
		"test_const":       makeConstHook(Template{}),
	}
)

func expHookCondList(info trans.SetterInfo, args []interface{}) (interface{}, error) {
	lexedElifText := args[0].(string)
	elifBlocks := args[1].([]Block)
	var list []ExpCondNode
	if len(args) >= 3 {
		list = args[2].([]ExpCondNode)
	}

	// extract the tunascript from the elif token. Fairly complicated due to
	// accepting "elseif", "else if", and "elif".
	elifExpr := strings.TrimPrefix(lexedElifText, "$[[")
	elifExpr = strings.TrimLeftFunc(elifExpr, unicode.IsSpace)
	elifExpr = strings.TrimLeft(elifExpr, "Ee")
	elifExpr = strings.TrimLeft(elifExpr, "Ll")
	elifExpr = strings.TrimLeft(elifExpr, "Ss")
	elifExpr = strings.TrimLeft(elifExpr, "Ee")
	elifExpr = strings.TrimLeftFunc(elifExpr, unicode.IsSpace)
	elifExpr = strings.TrimLeft(elifExpr, "Ii")
	elifExpr = strings.TrimLeft(elifExpr, "Ff")
	elifExpr = strings.TrimSuffix(elifExpr, "]]")

	elifCond := ExpCondNode{
		RawCond: elifExpr,
		Content: elifBlocks,
		Source:  info.FirstToken,
	}

	list = append(list, elifCond)

	return list, nil
}

func expHookBranch(info trans.SetterInfo, args []interface{}) (interface{}, error) {
	lexedIfText := args[0].(string)
	ifBlocks := args[1].([]Block)
	var elseIfConds []ExpCondNode
	if len(args) >= 3 {
		elseIfConds = args[2].([]ExpCondNode)
	}

	// extract the tunascript from the if token
	ifExpr := strings.TrimPrefix(lexedIfText, "$[[")
	ifExpr = strings.TrimLeftFunc(ifExpr, unicode.IsSpace)
	ifExpr = strings.TrimLeft(ifExpr, "Ii")
	ifExpr = strings.TrimLeft(ifExpr, "Ff")
	ifExpr = strings.TrimSuffix(ifExpr, "]]")

	ifCond := ExpCondNode{
		RawCond: ifExpr,
		Content: ifBlocks,
		Source:  info.FirstToken,
	}

	return ExpBranchNode{
		If:     ifCond,
		ElseIf: elseIfConds,
	}, nil
}

func expHookBranchWithElse(info trans.SetterInfo, args []interface{}) (interface{}, error) {
	lexedIfText := args[0].(string)
	ifBlocks := args[1].([]Block)
	elseBlocks := args[2].([]Block)
	var elseIfConds []ExpCondNode
	if len(args) >= 4 {
		elseIfConds = args[3].([]ExpCondNode)
	}

	// extract the tunascript from the if token
	ifExpr := strings.TrimPrefix(lexedIfText, "$[[")
	ifExpr = strings.TrimLeftFunc(ifExpr, unicode.IsSpace)
	ifExpr = strings.TrimLeft(ifExpr, "Ii")
	ifExpr = strings.TrimLeft(ifExpr, "Ff")
	ifExpr = strings.TrimSuffix(ifExpr, "]]")

	ifCond := ExpCondNode{
		RawCond: ifExpr,
		Content: ifBlocks,
		Source:  info.FirstToken,
	}

	return ExpBranchNode{
		If:     ifCond,
		ElseIf: elseIfConds,
		Else:   elseBlocks,
	}, nil
}

func expHookFlag(info trans.SetterInfo, args []interface{}) (interface{}, error) {
	lexedIdent := args[0].(string)

	fname := strings.TrimPrefix(strings.ToUpper(lexedIdent), "$")

	return ExpFlagNode{
		Flag:   fname,
		Source: info.FirstToken,
	}, nil
}

func expHookText(info trans.SetterInfo, args []interface{}) (interface{}, error) {
	lexedText := args[0].(string)
	actualText := InterpretEscapes(lexedText)

	var ltrimmed, rtrimmed string

	ltrimmed = strings.TrimLeftFunc(actualText, unicode.IsSpace)
	rtrimmed = strings.TrimRightFunc(actualText, unicode.IsSpace)

	if ltrimmed == actualText {
		ltrimmed = ""
	}
	if rtrimmed == actualText {
		rtrimmed = ""
	}

	return ExpTextNode{
		Text:              actualText,
		LeftSpaceTrimmed:  ltrimmed,
		RightSpaceTrimmed: rtrimmed,
		Source:            info.FirstToken,
	}, nil
}

func expHookAST(info trans.SetterInfo, args []interface{}) (interface{}, error) {
	nodes := args[0].([]Block)

	ast := Template{
		Blocks: nodes,
	}

	return ast, nil
}

func expHookNodeList(info trans.SetterInfo, args []interface{}) (interface{}, error) {
	var list []Block

	var appendNode Block

	if len(args) >= 1 {
		// add item to list
		appendNode = args[0].(Block)
		if len(args) >= 2 {
			list = args[1].([]Block)
		}
	}

	if appendNode != nil {
		list = append(list, appendNode)
	}

	return list, nil
}
