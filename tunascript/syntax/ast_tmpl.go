package syntax

import (
	"fmt"
	"strings"

	"github.com/dekarrin/ictiobus/lex"
	"github.com/dekarrin/rosed"
)

// Template is a parsed tunaquest template containing both tunascript
// template-legal expressions and regular text. The zero-value of an Template is
// an empty Template. Text can be parsed into a Template by calling Analyze on
// the template frontend.
//
// Templates cannot be expanded on their own and require the help of an engine
// that provides a suitable execution environment. Typically, this is done with
// an Interpreter from the tunascript package.
type Template struct {
	// Blocks is the template blocks that make up the complete template. They
	// are arranged in the order they appear in the finished template.
	Blocks []Block
}

// String returns a debug-tailored string that represents the Template. Two
// templates are considered semantically identical and will produce the same
// text if their String() methods produce the same output.
func (tmpl Template) String() string {
	var sb strings.Builder

	sb.WriteString("Template")
	if len(tmpl.Blocks) < 1 {
		sb.WriteString("(empty)")
		return sb.String()
	}

	const stmtStart = " B: "
	for i := range tmpl.Blocks {
		sb.WriteRune('\n')

		stmtStr := spaceIndentNewlines(tmpl.Blocks[i].String(), len(stmtStart))

		sb.WriteString(stmtStart)
		sb.WriteString(stmtStr)
	}

	return sb.String()
}

// Template returns the string that, if pased, would produce a Template
// identical to this one. It does *not* return, necessarily, the exact text that
// was parsed to create it, as some non-semantic elements such as whitespace
// within control-flow statements may be slightly altered.
func (tmpl Template) Template() string {
	return ""
}

// Equal returns whether this Template is equal to another value. This will
// return true only if o is another Template or Template pointer that has the
// same members as tmpl.
func (tmpl Template) Equal(o any) bool {
	other, ok := o.(Template)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*Template)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if len(tmpl.Blocks) != len(other.Blocks) {
		return false
	}
	for i := range tmpl.Blocks {
		if !tmpl.Blocks[i].Equal(other.Blocks[i]) {
			return false
		}
	}

	return true
}

// BlockType is the type of a template Block. Every Block will be one of these
// types, and it dictates which of its As*() functions can be called.
type BlockType int

const (
	// TmplText is the type of a TextBlock, which contains literal text which
	// will not be expanded further.
	TmplText BlockType = iota

	// TmplFlag is the type of a FlagBlock, which contains a flag that will be
	// replaced with its actual value at the time it is expanded.
	TmplFlag

	// TmplBranch is the type of a BranchBlock, which contains flow-control
	// statements that will be replaced with the text in the applicable branch
	// at the time it is expanded.
	TmplBranch

	// TmplCond is the type of a CondBlock, which contains both a TunaScript
	// condition and the content that the block should be expanded to if it is
	// selected as the branch from within a BranchBlock.
	TmplCond
)

// Block is a block of parsed template code in a Template. It represents the
// smallest abstract unit that a template can be divided into.
type Block interface {

	// Type returns the type of the Block. This determines which of the As*()
	// functions may be called.
	Type() BlockType

	// Returns this node as an ExpTextNode. Panics if Type() does not return
	// ExpText.
	AsText() TextBlock

	// Returns this node as an ExpFlagNode. Panics if Type() does not return
	// ExpFlag.
	AsFlag() FlagBlock

	// Returns this node as an ExpBranchNode. Panics if Type() does not return
	// ExpBranch.
	AsBranch() BranchBlock

	// Returns this node as an ExpCondNode. Panics if Type() does not return
	// ExpCond.
	AsCond() CondBlock

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

	// Template returns the string that, if pased, would produce a Block
	// identical to this one. It does *not* return, necessarily, the exact text
	// that was parsed to create it, as some non-semantic elements such as
	// whitespace within control-flow statements may be slightly altered.
	Template() string
}

// TextBlock is a block of unexpandable text in a template. This text will be
// returned as-is when expanded.
type TextBlock struct {

	// Text is the literal text in the block.
	Text              string
	LeftSpaceTrimmed  string
	RightSpaceTrimmed string

	// Source is the lexed token that was used to produce this TextBlock. If
	// this TextBlock was not created by parsing template code, this will be
	// nil.
	Source lex.Token
}

func (n TextBlock) Type() BlockType       { return TmplText }
func (n TextBlock) AsText() TextBlock     { return n }
func (n TextBlock) AsFlag() FlagBlock     { panic("Type() is not ExpFlag") }
func (n TextBlock) AsBranch() BranchBlock { panic("Type() is not ExpBranch") }
func (n TextBlock) AsCond() CondBlock     { panic("Type() is not ExpCond") }

func (n TextBlock) String() string {
	s := fmt.Sprintf("[TEXT ltrim=%t rtrim=%t\n", n.HasLeftTrimmed(), n.HasRightTrimmed())
	wrappedText := rosed.Edit(n.Text).Wrap(60).String()

	titleStart := "    "
	s += titleStart + spaceIndentNewlines(wrappedText, len(titleStart))
	s += "\n]"

	return s
}

func (n TextBlock) HasLeftTrimmed() bool {
	if n.Text == "" {
		return false
	}
	return n.LeftSpaceTrimmed != ""
}

func (n TextBlock) HasRightTrimmed() bool {
	if n.Text == "" {
		return false
	}
	return n.RightSpaceTrimmed != ""
}

// Does not consider Source.
func (n TextBlock) Equal(o any) bool {
	other, ok := o.(TextBlock)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*TextBlock)
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

func (n TextBlock) Template() string {
	// is any escaping required? escape backslashes and dollars just to be on
	// the safe side.
	tmplText := strings.ReplaceAll(n.Text, "\\", "\\\\")
	tmplText = strings.ReplaceAll(tmplText, "$", "\\$")
	return tmplText
}

// FlagBlock holds a variable (flag) within a template. During expansion, this
// will be replaced with the actual value of the flag at that time, converted to
// a string for display.
type FlagBlock struct {
	Flag   string
	Source lex.Token
}

func (n FlagBlock) Type() BlockType       { return TmplFlag }
func (n FlagBlock) AsText() TextBlock     { panic("Type() is not ExpText") }
func (n FlagBlock) AsFlag() FlagBlock     { return n }
func (n FlagBlock) AsBranch() BranchBlock { panic("Type() is not ExpBranch") }
func (n FlagBlock) AsCond() CondBlock     { panic("Type() is not ExpCond") }

func (n FlagBlock) String() string {
	s := fmt.Sprintf("[FLAG $%s]", n.Flag)
	return s
}

// Does not consider Source.
func (n FlagBlock) Equal(o any) bool {
	other, ok := o.(FlagBlock)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*FlagBlock)
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

func (n FlagBlock) Template() string {
	return "$" + n.Flag
}

// BranchBlock is a series of control-flow statements and their contents within
// a template. This may include the opening IF statement, any ELSE-IFs, and the
// ELSE attached to the IF, if any is present. This block is expanded to the
// first applicable branch's contents at the time that it is expanded.
type BranchBlock struct {
	If CondBlock

	// ElseIf will be empty if there are no else-if blocks.
	ElseIf []CondBlock

	// Else will be nil if there are no else blocks.
	Else []Block

	Source lex.Token
}

func (n BranchBlock) Type() BlockType       { return TmplBranch }
func (n BranchBlock) AsText() TextBlock     { panic("Type() is not ExpText") }
func (n BranchBlock) AsFlag() FlagBlock     { panic("Type() is not ExpFlag") }
func (n BranchBlock) AsBranch() BranchBlock { return n }
func (n BranchBlock) AsCond() CondBlock     { panic("Type() is not ExpCond") }

func (n BranchBlock) String() string {
	ifStart := " I: "
	elifStart := " EI:"
	elseStart := " E: "

	s := "[BRANCH\n"

	s += fmt.Sprintf("%s%s\n", ifStart, spaceIndentNewlines(n.If.String(), len(ifStart)))

	for i := range n.ElseIf {
		s += fmt.Sprintf("%s%s\n", elifStart, spaceIndentNewlines(n.ElseIf[i].String(), len(elifStart)))
	}

	if n.Else != nil {
		for i := range n.Else {
			s += fmt.Sprintf("%s%s\n", elseStart, spaceIndentNewlines(n.Else[i].String(), len(elseStart)))
		}
	}
	s += "]"
	return s
}

// Does not consider Source.
func (n BranchBlock) Equal(o any) bool {
	other, ok := o.(BranchBlock)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*BranchBlock)
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
	if len(n.Else) != len(other.Else) {
		return false
	}
	for i := range n.Else {
		if !n.Else[i].Equal(other.Else[i]) {
			return false
		}
	}

	return true
}

func (n BranchBlock) Template() string {
	var sb strings.Builder

	// if-block
	sb.WriteString("$[[IF")
	if n.If.Cond.Nodes != nil {
		sb.WriteRune(' ')
		sb.WriteString(n.If.Cond.Tunascript())
	} else if n.If.RawCond != "" {
		sb.WriteRune(' ')
		sb.WriteString(n.If.RawCond)
	}
	sb.WriteString("]]")

	for _, cont := range n.If.Content {
		sb.WriteString(cont.Template())
	}

	// any else-ifs?
	for _, elif := range n.ElseIf {
		sb.WriteString("$[[ELSE IF")
		if elif.Cond.Nodes != nil {
			sb.WriteRune(' ')
			sb.WriteString(elif.Cond.Tunascript())
		} else if elif.RawCond != "" {
			sb.WriteRune(' ')
			sb.WriteString(elif.RawCond)
		}
		sb.WriteString("]]")
		for _, cont := range elif.Content {
			sb.WriteString(cont.Template())
		}
	}

	// finally, do we have an else?
	if len(n.Else) > 0 {
		sb.WriteString("$[[ELSE]]")
		for _, cont := range n.Else {
			sb.WriteString(cont.Template())
		}
	}

	// close the branch
	sb.WriteString("$[[ENDIF]]")
	return sb.String()
}

// CondBlock is a single condition from a BranchBlock. It holds both the
// TunaScript code that makes up its condition, which may or may not already be
// parsed (it will be parsed if this CondBlock was in a Template returned by an
// Interpreter).
type CondBlock struct {
	Cond AST

	// On initial parsing of template trees, only this will be set. The
	// contents of this string can be parsed by passing it to the TS frontend.
	RawCond string

	Content []Block

	Source lex.Token
}

func (n CondBlock) Type() BlockType       { return TmplFlag }
func (n CondBlock) AsText() TextBlock     { panic("Type() is not ExpText") }
func (n CondBlock) AsFlag() FlagBlock     { panic("Type() is not ExpFlag") }
func (n CondBlock) AsBranch() BranchBlock { panic("Type() is not ExpBranch") }
func (n CondBlock) AsCond() CondBlock     { return n }

func (n CondBlock) String() string {
	condStart := " IF:"
	contentStart := " C: "

	var condStr string
	if n.Cond.Nodes != nil {
		condStr = spaceIndentNewlines(n.Cond.String(), len(condStart))
	} else {
		condStr = spaceIndentNewlines("(raw) "+n.RawCond, len(condStart))
	}

	s := fmt.Sprintf("[COND\n%s%s", condStart, condStr)

	for i := range n.Content {
		contentStr := spaceIndentNewlines(n.Content[i].String(), len(contentStart))
		s += fmt.Sprintf("\n%s%s", contentStart, contentStr)
	}

	s += "]"

	return s
}

// Does not consider Source.
func (n CondBlock) Equal(o any) bool {
	other, ok := o.(CondBlock)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*CondBlock)
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
	if n.RawCond != other.RawCond {
		return false
	}
	if len(n.Content) != len(other.Content) {
		return false
	}
	for i := range n.Content {
		if !n.Content[i].Equal(other.Content[i]) {
			return false
		}
	}

	return true
}

func (n CondBlock) Template() string {
	var sb strings.Builder

	sb.WriteString("$[[IF")
	if n.Cond.Nodes != nil {
		sb.WriteRune(' ')
		sb.WriteString(n.Cond.Tunascript())
	} else if n.RawCond != "" {
		sb.WriteRune(' ')
		sb.WriteString(n.RawCond)
	}
	sb.WriteString("]]")

	for _, cont := range n.Content {
		sb.WriteString(cont.Template())
	}

	sb.WriteString("$[[ENDIF]]")
	return sb.String()
}
