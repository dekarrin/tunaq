package translation

import (
	"fmt"

	"github.com/dekarrin/tunaq/internal/util"
)

type AttrRef struct {
	Relation NodeRelation
	Name     NodeAttrName
}

type NodeRelationType int

const (
	RelHead NodeRelationType = iota
	RelTerminal
	RelNonTerminal
	RelSymbol
)

func (nrt NodeRelationType) String() string {
	if nrt == RelHead {
		return "head symbol"
	} else if nrt == RelTerminal {
		return "terminal symbol"
	} else if nrt == RelNonTerminal {
		return "non-terminal symbol"
	} else if nrt == RelSymbol {
		return "symbol"
	} else {
		return fmt.Sprintf("NodeRelationType<%d>", int(nrt))
	}
}

type NodeRelation struct {
	// Type is the type of the relation.
	Type NodeRelationType

	// Index specifies which of the nodes of the given type that the relation
	// points to. If it is RelHead, this will be 0.
	Index int
}

func (arl NodeRelation) String() string {
	if arl.Type == RelHead {
		return arl.Type.String()
	}

	humanIndex := arl.Index + 1
	return fmt.Sprintf("%d%s %s", humanIndex, util.OrdinalSuf(humanIndex), arl.Type.String())
}
