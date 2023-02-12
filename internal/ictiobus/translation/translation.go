// Package translation holds constructs involved in the final stage of langage
// processing. It can also serve as an entrypoint with a full-featured
// translation intepreter engine.
package translation

import (
	"fmt"

	"github.com/dekarrin/tunaq/internal/ictiobus/lex"
)

type SyntaxDirectedTranslator struct {
}

type SDD interface {
	BindInheritedAttribute(ruleHead string, ruleProds []string, bindFunc AttributeSetter)
}

type SDDBinding struct {
	// Synthesized is whether the binding is for a
	Synthesized bool

	// BoundRuleSymbol is the head symbol of the rule the binding is on.
	BoundRuleSymbol string

	// BoundRuleProduction is the list of produced symbols of the rule the
	// binding is on.
	BoundRuleProduction []string

	// Requirements is the attribute references that this binding needs.
	Requirements []AttrRef

	// Dest is the destination.
	Dest AttrRef

	// Setter is the call to calculate a value of the node by the binding.
	Setter AttributeSetter
}

// forAttr can always be taken from Dest.Name
func (bind SDDBinding) Invoke(apt AnnotatedParseTree) NodeAttrValue {
	// sanity checks; can we even call this?
	if bind.Setter == nil {
		panic("attempt to invoke nil attribute setter func")
	}
	if bind.Dest.Relation.Type == RelHead && !bind.Synthesized {
		panic("cannot invoke inherited attribute SDD binding on head of rule")
	} else if bind.Dest.Relation.Type != RelHead && bind.Synthesized {
		panic("cannot invoke synthesized attribute SDD binding on production of rule")
	}

	// symbol of who it is for
	forSymbol, ok := apt.SymbolOf(bind.Dest.Relation)
	if !ok {
		// invalid dest
		panic(fmt.Sprintf("bound-to rule does not contain a %s", bind.Dest.Relation.String()))
	}

	// gather args
	args := []NodeAttrValue{}
	for i := range bind.Requirements {
		req := bind.Requirements[i]
		reqVal, ok := apt.AttributeValueOf(req)
		if !ok {
			// should never happen, creation of Binding should ensure this.
			_, refNodeExists := apt.AttributesOf(req.Relation)
			if !refNodeExists {
				// reference itself was invalid
				panic(fmt.Sprintf("bound-to rule does not contain a %s", req.Relation.String()))
			} else {
				panic(fmt.Sprintf("attribute %s not yet defined for %s in bound-to-rule", req.Name, req.Relation.String()))
			}
		}

		args = append(args, reqVal)
	}

	// call func
	val := bind.Setter(forSymbol, bind.Dest.Name, args)

	return val
}

type SAttrSDD struct {
	Rules map[string][]AttributeSetter
}

type NodeAttrName string
type NodeAttrValue interface{}

type NodeAttrs map[NodeAttrName]NodeAttrValue

func (na NodeAttrs) Copy() NodeAttrs {
	newNa := NodeAttrs{}
	for k := range na {
		newNa[k] = na[k]
	}
	return newNa
}

type NodeValues struct {
	Attributes NodeAttrs

	Terminal bool

	Symbol string
}

type AttributeSetter func(symbol string, name NodeAttrName, args []NodeAttrValue) NodeAttrValue

type AnnotatedParseTree struct {
	// Terminal is whether this node is for a terminal symbol.
	Terminal bool

	// Symbol is the symbol at this node's head.
	Symbol string

	// Source is only available when Terminal is true.
	Source lex.Token

	// Children is all children of the parse tree.
	Children []*AnnotatedParseTree

	// Attributes is the data for attributes at the given position in the parse
	// tree.
	Attributes NodeAttrs
}

// SymbolOf returns the symbol of the node referred to by rel. Additionally, a
// second 'ok' value is returned that specifies whether a node matches rel. Iff
// the second value is false, the first value should not be relied on.
func (apt AnnotatedParseTree) SymbolOf(rel NodeRelation) (symbol string, ok bool) {
	node, ok := apt.RelativeNode(rel)
	if !ok {
		return "", false
	}
	return node.Symbol, true
}

// AttributeValueOf returns the value of the named attribute in the node
// referred to by ref. Additionally, a second 'ok' value is returned that
// specifies whether ref refers to an existing attribute in the node whose
// relation to apt matches that specified in ref; if the returned 'ok' value is
// false, val should be considered a nil value and unsafe to use.
func (apt AnnotatedParseTree) AttributeValueOf(ref AttrRef) (val NodeAttrValue, ok bool) {
	// first get the attributes
	attributes, ok := apt.AttributesOf(ref.Relation)
	if !ok {
		return nil, false
	}

	attrVal, ok := attributes[ref.Name]
	return attrVal, ok
}

// RelativeNode returns the node pointed to by rel. Specifically, it returns the
// node that is related to apt in the way specified by rel, which can be at most
// one node as per the definition of rel's type.
//
// RelHead will cause apt itself to be returned; all others select a child node.
//
// A second 'ok' value is returned. This value is true if rel is a relation that
// exists in apt. If rel specifies a node that does not exist, the ok value will
// be false and the returned related node should not be used.
func (apt AnnotatedParseTree) RelativeNode(rel NodeRelation) (related *AnnotatedParseTree, ok bool) {
	if rel.Type == RelHead {
		return &apt, true
	} else if rel.Type == RelSymbol {
		symIdx := rel.Index
		if symIdx >= len(apt.Children) {
			return nil, false
		}
		return apt.Children[symIdx], true
	} else if rel.Type == RelNonTerminal {
		searchNonTermIdx := rel.Index

		// find the nth non-terminal
		curNonTermIdx := -1
		foundIdx := -1
		for i := range apt.Children {
			childNode := apt.Children[i]

			if childNode.Terminal {
				continue
			} else {
				curNonTermIdx++
				if curNonTermIdx == searchNonTermIdx {
					foundIdx = i
					break
				}
			}
		}
		if foundIdx == -1 {
			return nil, false
		}
		return apt.Children[foundIdx], true
	} else if rel.Type == RelTerminal {
		searchTermIdx := rel.Index

		// find the nth non-terminal
		curTermIdx := -1
		foundIdx := -1
		for i := range apt.Children {
			childNode := apt.Children[i]

			if !childNode.Terminal {
				continue
			} else {
				curTermIdx++
				if curTermIdx == searchTermIdx {
					foundIdx = i
					break
				}
			}
		}
		if foundIdx == -1 {
			return nil, false
		}
		return apt.Children[foundIdx], true
	} else {
		// not a valid AttrRelNode, can't handle it
		return nil, false
	}
}

// AttributesOf gets the Attributes of the node referred to by the given
// AttrRelNode value. For valid relations (those for which apt has a match for
// among itself (for the head symbol) and children (for the produced symbols)),
// the Attributes of the specified node are returned, as well as a second 'ok'
// value which will be true. If rel specifies a node that doesn't exist relative
// to apt, then the second value will be false and the returned node attributes
// will be nil.
func (apt AnnotatedParseTree) AttributesOf(rel NodeRelation) (attributes NodeAttrs, ok bool) {
	node, ok := apt.RelativeNode(rel)
	if !ok {
		return nil, false
	}
	return node.Attributes, true
}
