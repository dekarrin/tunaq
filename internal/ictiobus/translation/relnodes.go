package translation

import (
	"fmt"
	"strings"

	"github.com/dekarrin/tunaq/internal/util"
)

// AttrRef contains no uncomparable attributes and can be assigned/copied
// directly.
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

func (nr NodeRelation) String() string {
	if nr.Type == RelHead {
		return nr.Type.String()
	}

	humanIndex := nr.Index + 1
	return fmt.Sprintf("%d%s %s", humanIndex, util.OrdinalSuf(humanIndex), nr.Type.String())
}

// ValidFor returns whether the given node relation refers to a valid and
// existing node when applied to a node in parse tree that is the result of
// parsing production head -> production.
func (nr NodeRelation) ValidFor(head string, prod []string) bool {
	// Refering to the head is refering to the node itself, so is always valid.
	if nr.Type == RelHead {
		return true
	} else if nr.Type == RelSymbol {
		return nr.Index < len(prod) && nr.Index >= 0
	} else if nr.Type == RelTerminal {
		searchTermIdx := nr.Index

		// find the nth terminal
		curTermIdx := -1
		foundIdx := -1
		for i := range prod {
			sym := prod[i]

			if strings.ToLower(sym) != sym {
				continue
			} else {
				curTermIdx++
				if curTermIdx == searchTermIdx {
					foundIdx = i
					break
				}
			}
		}
		return foundIdx != -1
	} else if nr.Type == RelNonTerminal {
		searchNonTermIdx := nr.Index

		// find the nth non-terminal
		curNonTermIdx := -1
		foundIdx := -1
		for i := range prod {
			sym := prod[i]

			if strings.ToLower(sym) != sym {
				continue
			} else {
				curNonTermIdx++
				if curNonTermIdx == searchNonTermIdx {
					foundIdx = i
					break
				}
			}
		}
		return foundIdx != -1
	} else {
		return false
	}
}
