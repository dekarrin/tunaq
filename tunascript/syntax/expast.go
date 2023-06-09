package syntax

import (
	"fmt"

	"github.com/dekarrin/ictiobus/lex"
	"github.com/dekarrin/rosed"
)

// ExpansionAST is a block of text containing both tunascript
// expansion-legal expressions and regular text. The zero-value of an
// ExpansionAST is not suitable for use and they should only be created by calls
// to AnalyzeExpansion.
type ExpansionAST struct {
	Nodes []ExpNode
}

func (ast ExpansionAST) String() string {
	return ""
}

func (n ExpansionAST) Equal(o any) bool {
	other, ok := o.(ExpansionAST)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*ExpansionAST)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if len(n.Nodes) != len(other.Nodes) {
		return false
	}
	for i := range n.Nodes {
		if !n.Nodes[i].Equal(other.Nodes[i]) {
			return false
		}
	}

	return true
}

type ExpNodeType int

const (
	ExpText ExpNodeType = iota
	ExpFlag
	ExpBranch
	ExpCond
)

// ExpNode is a node in an ExpansionAST.
type ExpNode interface {

	// Type returns the type of the ExpansionAST. This determines which of the As*()
	// functions may be called.
	Type() ExpNodeType

	// Returns this node as an ExpTextNode. Panics if Type() does not return
	// ExpText.
	AsTextNode() ExpTextNode

	// Returns this node as an ExpFlagNode. Panics if Type() does not return
	// ExpFlag.
	AsFlagNode() ExpFlagNode

	// Returns this node as an ExpBranchNode. Panics if Type() does not return
	// ExpBranch.
	AsBranchNode() ExpBranchNode

	// Returns this node as an ExpCondNode. Panics if Type() does not return
	// ExpCond.
	AsCondNode() ExpCondNode

	// String returns a prettified representation of the node suitable for use
	// in line-by-line comparisons of tree structure. Two nodes are considered
	// semantcally identical if they produce identical String() output.
	String() string

	// Equal returns whether a node is equal to another. It will return false
	// if anything besides an ASTNode is passed in. ASTNodes do not consider
	// the result of Source() in their equality; ergo, this returns whether two
	// nodes have the same structure regardless of the exact source that
	// produced them.
	Equal(o any) bool
}

type ExpTextNode struct {
	Text              string
	LeftSpaceTrimmed  string
	RightSpaceTrimmed string
	Source            lex.Token
}

func (n ExpTextNode) Type() ExpNodeType           { return ExpText }
func (n ExpTextNode) AsTextNode() ExpTextNode     { return n }
func (n ExpTextNode) AsFlagNode() ExpFlagNode     { panic("Type() is not ExpFlag") }
func (n ExpTextNode) AsBranchNode() ExpBranchNode { panic("Type() is not ExpBranch") }
func (n ExpTextNode) AsCondNode() ExpCondNode     { panic("Type() is not ExpCond") }

func (n ExpTextNode) String() string {
	s := fmt.Sprintf("[EXP_TEXT ltrim=%t rtrim=%t\n", n.HasLeftTrimmed(), n.HasRightTrimmed())
	wrappedText := rosed.Edit(n.Text).Wrap(60).String()

	titleStart := "    "
	s += titleStart + spaceIndentNewlines(wrappedText, len(titleStart))
	s += "\n]"

	return s
}

func (n ExpTextNode) HasLeftTrimmed() bool {
	if n.Text == "" {
		return false
	}
	return n.LeftSpaceTrimmed != ""
}

func (n ExpTextNode) HasRightTrimmed() bool {
	if n.Text == "" {
		return false
	}
	return n.RightSpaceTrimmed != ""
}

// Does not consider Source.
func (n ExpTextNode) Equal(o any) bool {
	other, ok := o.(ExpTextNode)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*ExpTextNode)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if n.LeftSpaceTrimmed != other.LeftSpaceTrimmed {
		return false
	}
	if n.RightSpaceTrimmed != other.RightSpaceTrimmed {
		return false
	}
	if n.Text != other.Text {
		return false
	}

	return true
}

type ExpFlagNode struct {
	Flag   string
	Source lex.Token
}

func (n ExpFlagNode) Type() ExpNodeType           { return ExpFlag }
func (n ExpFlagNode) AsTextNode() ExpTextNode     { panic("Type() is not ExpText") }
func (n ExpFlagNode) AsFlagNode() ExpFlagNode     { return n }
func (n ExpFlagNode) AsBranchNode() ExpBranchNode { panic("Type() is not ExpBranch") }
func (n ExpFlagNode) AsCondNode() ExpCondNode     { panic("Type() is not ExpCond") }

func (n ExpFlagNode) String() string {
	s := fmt.Sprintf("[EXP_FLAG $%s]", n.Flag)
	return s
}

// Does not consider Source.
func (n ExpFlagNode) Equal(o any) bool {
	other, ok := o.(ExpFlagNode)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*ExpFlagNode)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if n.Flag != other.Flag {
		return false
	}

	return true
}

type ExpBranchNode struct {
	If ExpCondNode

	// ElseIf will be empty if there are no else-if blocks.
	ElseIf []ExpCondNode

	// Else will be nil if there are no else blocks.
	Else *ExpansionAST

	Source lex.Token
}

func (n ExpBranchNode) Type() ExpNodeType           { return ExpFlag }
func (n ExpBranchNode) AsTextNode() ExpTextNode     { panic("Type() is not ExpText") }
func (n ExpBranchNode) AsFlagNode() ExpFlagNode     { panic("Type() is not ExpFlag") }
func (n ExpBranchNode) AsBranchNode() ExpBranchNode { return n }
func (n ExpBranchNode) AsCondNode() ExpCondNode     { panic("Type() is not ExpCond") }

func (n ExpBranchNode) String() string {
	ifStart := " I: "
	elifStart := " EI:"
	elseStart := " E: "

	s := "[EXP_BRANCH\n"

	s += fmt.Sprintf("%s%s\n", ifStart, spaceIndentNewlines(n.If.String(), len(ifStart)))

	for i := range n.ElseIf {
		s += fmt.Sprintf("%s%s\n", elifStart, spaceIndentNewlines(n.ElseIf[i].String(), len(elifStart)))
	}

	if n.Else != nil {
		s += fmt.Sprintf("%s%s\n", elseStart, spaceIndentNewlines(n.Else.String(), len(elseStart)))
	}
	s += "]"
	return s
}

// Does not consider Source.
func (n ExpBranchNode) Equal(o any) bool {
	other, ok := o.(ExpBranchNode)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*ExpBranchNode)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if !n.If.Equal(other.If) {
		return false
	}
	if len(n.ElseIf) != len(other.ElseIf) {
		return false
	}
	for i := range n.ElseIf {
		if !n.ElseIf[i].Equal(other.ElseIf[i]) {
			return false
		}
	}
	if (n.Else == nil && other.Else != nil) || (n.Else != nil && other.Else == nil) {
		return false
	}
	if n.Else != nil && !n.Else.Equal(other.Else) {
		return false
	}

	return true
}

type ExpCondNode struct {
	Cond AST

	// On initial parsing of expansion trees, only this will be set. The
	// contents of this string can be parsed by passing it to the TS frontend.
	RawCond string

	Content ExpansionAST

	Source lex.Token
}

func (n ExpCondNode) Type() ExpNodeType           { return ExpFlag }
func (n ExpCondNode) AsTextNode() ExpTextNode     { panic("Type() is not ExpText") }
func (n ExpCondNode) AsFlagNode() ExpFlagNode     { panic("Type() is not ExpFlag") }
func (n ExpCondNode) AsBranchNode() ExpBranchNode { panic("Type() is not ExpBranch") }
func (n ExpCondNode) AsCondNode() ExpCondNode     { return n }

func (n ExpCondNode) String() string {
	condStart := " IF:"
	contentStart := " C: "

	var condStr string
	if n.Cond.Nodes != nil {
		condStr = spaceIndentNewlines(n.Cond.String(), len(condStart))
	} else {
		condStr = spaceIndentNewlines(n.RawCond, len(condStart))
	}
	contentStr := spaceIndentNewlines(n.Content.String(), len(contentStart))

	return fmt.Sprintf("[EXP_COND\n%s%s\n%s%s\n]", condStart, condStr, contentStart, contentStr)
}

// Does not consider Source.
func (n ExpCondNode) Equal(o any) bool {
	other, ok := o.(ExpCondNode)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*ExpCondNode)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if !n.Cond.Equal(other.Cond) {
		return false
	}
	if !n.Content.Equal(other.Content) {
		return false
	}
	if n.RawCond != other.RawCond {
		return false
	}

	return true
}
