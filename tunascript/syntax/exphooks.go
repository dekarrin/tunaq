package syntax

import "github.com/dekarrin/ictiobus/trans"

func makeConstHook(v interface{}) trans.Hook {
	return func(info trans.SetterInfo, args []interface{}) (interface{}, error) {
		return v, nil
	}
}

var (
	ExpHooksTable = trans.HookMap{
		"test_const": makeConstHook(ExpansionAST{}),
	}
)
