package types

import (
	"fmt"
	"strings"

	"github.com/dekarrin/tunaq/internal/ictiobus/lex"
)

const (
	treeLevelEmpty               = "        "
	treeLevelOngoing             = "  |     "
	treeLevelPrefix              = "  |%s: "
	treeLevelPrefixLast          = `  \%s: `
	treeLevelPrefixNamePadChar   = '-'
	treeLevelPrefixNamePadAmount = 3
)

func makeTreeLevelPrefix(msg string) string {
	for len([]rune(msg)) < treeLevelPrefixNamePadAmount {
		msg = string(treeLevelPrefixNamePadChar) + msg
	}
	return fmt.Sprintf(treeLevelPrefix, msg)
}

func makeTreeLevelPrefixLast(msg string) string {
	for len([]rune(msg)) < treeLevelPrefixNamePadAmount {
		msg = string(treeLevelPrefixNamePadChar) + msg
	}
	return fmt.Sprintf(treeLevelPrefixLast, msg)
}

type ParseTree struct {
	// Terminal is whether thie node is for a terminal symbol.
	Terminal bool

	// Value is the symbol at this node.
	Value string

	// Source is only available when Terminal is true.
	Source lex.Token

	// Children is all children of the parse tree.
	Children []*ParseTree
}

// String returns a prettified representation of the entire parse tree suitable
// for use in line-by-line comparisons of tree structure. Two parse trees are
// considered semantcally identical if they produce identical String() output.
func (pt ParseTree) String() string {
	return pt.leveledStr("", "")
}

// Copy returns a duplicate, deeply-copied parse tree.
func (pt ParseTree) Copy() ParseTree {
	newPt := ParseTree{
		Terminal: pt.Terminal,
		Value:    pt.Value,
		Source:   pt.Source,
		Children: make([]*ParseTree, len(pt.Children)),
	}

	for i := range pt.Children {
		if pt.Children[i] != nil {
			newChild := pt.Children[i].Copy()
			newPt.Children[i] = &newChild
		}
	}

	return newPt
}

func (pt ParseTree) leveledStr(firstPrefix, contPrefix string) string {
	var sb strings.Builder

	sb.WriteString(firstPrefix)
	if pt.Terminal {
		sb.WriteString(fmt.Sprintf("(TERM %q)", pt.Value))
	} else {
		sb.WriteString(fmt.Sprintf("( %s )", pt.Value))
	}

	for i := range pt.Children {
		sb.WriteRune('\n')
		var leveledFirstPrefix string
		var leveledContPrefix string
		if i+1 < len(pt.Children) {
			leveledFirstPrefix = contPrefix + makeTreeLevelPrefix("")
			leveledContPrefix = contPrefix + treeLevelOngoing
		} else {
			leveledFirstPrefix = contPrefix + makeTreeLevelPrefixLast("")
			leveledContPrefix = contPrefix + treeLevelEmpty
		}
		itemOut := pt.Children[i].leveledStr(leveledFirstPrefix, leveledContPrefix)
		sb.WriteString(itemOut)
	}

	return sb.String()
}

// Equal returns whether the parseTree is equal to the given object. If the
// given object is not a parseTree, returns false, else returns whether the two
// parse trees have the exact same structure.
func (pt ParseTree) Equal(o any) bool {
	other, ok := o.(ParseTree)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*ParseTree)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if pt.Terminal != other.Terminal {
		return false
	} else if pt.Value != other.Value {
		return false
	} else {
		// check every sub tree
		if len(pt.Children) != len(other.Children) {
			return false
		}

		for i := range pt.Children {
			if !pt.Children[i].Equal(other.Children[i]) {
				return false
			}
		}
	}
	return true
}
