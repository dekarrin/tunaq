package syntax

import (
	"strings"
	"unicode"

	"github.com/dekarrin/ictiobus/trans"
)

var (
	TmplHooksTable = trans.HookMap{
		"identity":         func(info trans.SetterInfo, args []interface{}) (interface{}, error) { return args[0], nil },
		"ast":              tmplHookAST,
		"text":             tmplHookText,
		"flag":             tmplHookFlag,
		"branch":           tmplHookBranch,
		"branch_with_else": tmplHookBranchWithElse,
		"cond_list":        tmplHookCondList,
		"node_list":        tmplHookNodeList,
	}
)

func tmplHookCondList(info trans.SetterInfo, args []interface{}) (interface{}, error) {
	lexedElifText := args[0].(string)
	elifBlocks := args[1].([]Block)
	var list []CondBlock
	if len(args) >= 3 {
		list = args[2].([]CondBlock)
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

	elifCond := CondBlock{
		RawCond: elifExpr,
		Content: elifBlocks,
		Source:  info.FirstToken,
	}

	list = append(list, elifCond)

	return list, nil
}

func tmplHookBranch(info trans.SetterInfo, args []interface{}) (interface{}, error) {
	lexedIfText := args[0].(string)
	ifBlocks := args[1].([]Block)
	var elseIfConds []CondBlock
	if len(args) >= 3 {
		elseIfConds = args[2].([]CondBlock)
	}

	// extract the tunascript from the if token
	ifExpr := strings.TrimPrefix(lexedIfText, "$[[")
	ifExpr = strings.TrimLeftFunc(ifExpr, unicode.IsSpace)
	ifExpr = strings.TrimLeft(ifExpr, "Ii")
	ifExpr = strings.TrimLeft(ifExpr, "Ff")
	ifExpr = strings.TrimSuffix(ifExpr, "]]")

	ifCond := CondBlock{
		RawCond: ifExpr,
		Content: ifBlocks,
		Source:  info.FirstToken,
	}

	return BranchBlock{
		If:     ifCond,
		ElseIf: elseIfConds,
	}, nil
}

func tmplHookBranchWithElse(info trans.SetterInfo, args []interface{}) (interface{}, error) {
	lexedIfText := args[0].(string)
	ifBlocks := args[1].([]Block)
	elseBlocks := args[2].([]Block)
	var elseIfConds []CondBlock
	if len(args) >= 4 {
		elseIfConds = args[3].([]CondBlock)
	}

	// extract the tunascript from the if token
	ifExpr := strings.TrimPrefix(lexedIfText, "$[[")
	ifExpr = strings.TrimLeftFunc(ifExpr, unicode.IsSpace)
	ifExpr = strings.TrimLeft(ifExpr, "Ii")
	ifExpr = strings.TrimLeft(ifExpr, "Ff")
	ifExpr = strings.TrimSuffix(ifExpr, "]]")

	ifCond := CondBlock{
		RawCond: ifExpr,
		Content: ifBlocks,
		Source:  info.FirstToken,
	}

	return BranchBlock{
		If:     ifCond,
		ElseIf: elseIfConds,
		Else:   elseBlocks,
	}, nil
}

func tmplHookFlag(info trans.SetterInfo, args []interface{}) (interface{}, error) {
	lexedIdent := args[0].(string)

	fname := strings.TrimPrefix(strings.ToUpper(lexedIdent), "$")

	return FlagBlock{
		Flag:   fname,
		Source: info.FirstToken,
	}, nil
}

func tmplHookText(info trans.SetterInfo, args []interface{}) (interface{}, error) {
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

	return TextBlock{
		Text:              actualText,
		LeftSpaceTrimmed:  ltrimmed,
		RightSpaceTrimmed: rtrimmed,
		Source:            info.FirstToken,
	}, nil
}

func tmplHookAST(info trans.SetterInfo, args []interface{}) (interface{}, error) {
	nodes := args[0].([]Block)

	ast := Template{
		Blocks: nodes,
	}

	return ast, nil
}

func tmplHookNodeList(info trans.SetterInfo, args []interface{}) (interface{}, error) {
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
