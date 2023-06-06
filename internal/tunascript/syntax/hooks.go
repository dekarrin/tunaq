package syntax

import "github.com/dekarrin/ictiobus/trans"

var (
	HooksTable = trans.HookMap{
		"test_const": func(info trans.SetterInfo, args []interface{}) (interface{}, error) {
			return 1, nil
		},
	}
)
