package fetmpl

/*
File automatically generated by the ictiobus compiler. DO NOT EDIT. This was
created by invoking ictiobus with the following command:

    ictcc --slr -l TunaQuest Template -v 1.0 -d tte --sim-off --ir github.com/dekarrin/tunaq/tunascript/syntax.Template --hooks ./tunascript/syntax --hooks-table TmplHooksTable --dest ./tunascript/fetmpl --pkg fetmpl tunascript/expansion.md
*/

import (
	_ "embed"

	"github.com/dekarrin/ictiobus/grammar"
	"github.com/dekarrin/ictiobus/parse"

	"github.com/dekarrin/tunaq/tunascript/fetmpl/fetmpltoken"
)

var (
	//go:embed parser.cff
	parserData []byte
)

// Grammar returns the grammar accepted by the generated ictiobus parser for
// TunaQuest Template. This grammar will also be included with with the parser itself,
// but it is included here as well for convenience.
func Grammar() grammar.CFG {
	g := grammar.CFG{}

	g.AddTerm(fetmpltoken.TCElse.ID(), fetmpltoken.TCElse)
	g.AddTerm(fetmpltoken.TCElseif.ID(), fetmpltoken.TCElseif)
	g.AddTerm(fetmpltoken.TCEndif.ID(), fetmpltoken.TCEndif)
	g.AddTerm(fetmpltoken.TCFlag.ID(), fetmpltoken.TCFlag)
	g.AddTerm(fetmpltoken.TCIf.ID(), fetmpltoken.TCIf)
	g.AddTerm(fetmpltoken.TCText.ID(), fetmpltoken.TCText)

	g.AddRule("EXPANSION", []string{"BLOCKS"})

	g.AddRule("BLOCKS", []string{"BLOCKS", "BLOCK"})
	g.AddRule("BLOCKS", []string{"BLOCK"})

	g.AddRule("BLOCK", []string{"text"})
	g.AddRule("BLOCK", []string{"flag"})
	g.AddRule("BLOCK", []string{"BRANCH"})

	g.AddRule("BRANCH", []string{"if", "BLOCKS", "endif"})
	g.AddRule("BRANCH", []string{"if", "BLOCKS", "ELSEIFS", "endif"})
	g.AddRule("BRANCH", []string{"if", "BLOCKS", "else", "BLOCKS", "endif"})
	g.AddRule("BRANCH", []string{"if", "BLOCKS", "ELSEIFS", "else", "BLOCKS", "endif"})

	g.AddRule("ELSEIFS", []string{"ELSEIFS", "elseif", "BLOCKS"})
	g.AddRule("ELSEIFS", []string{"elseif", "BLOCKS"})

	return g
}

// Parser returns the generated ictiobus Parser for TunaQuest Template.
func Parser() parse.Parser {
	p, err := parse.DecodeBytes(parserData)
	if err != nil {
		panic("corrupted parser.cff file: " + err.Error())
	}

	return p
}