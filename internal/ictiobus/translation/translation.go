// Package translation holds constructs involved in the final stage of langage
// processing. It can also serve as an entrypoint with a full-featured
// translation intepreter engine.
package translation

import (
	"fmt"
	"strings"

	"github.com/dekarrin/tunaq/internal/ictiobus/types"
	"github.com/dekarrin/tunaq/internal/util"
)

type sddImpl struct {
	bindings map[string]map[string][]SDDBinding
}

func (sdd *sddImpl) BindingsFor(head string, prod []string, attrRef AttrRef) []SDDBinding {
	allForRule := sdd.Bindings(head, prod)

	matchingBindings := []SDDBinding{}

	for i := range allForRule {
		if allForRule[i].Dest == attrRef {
			matchingBindings = append(matchingBindings, allForRule[i])
		}
	}

	return matchingBindings
}

func (sdd *sddImpl) Evaluate(tree types.ParseTree, attributes ...NodeAttrName) ([]NodeAttrValue, error) {
	// first get an annotated parse tree
	root := AddAttributes(tree)
	depGraphs := DepGraph(root, sdd)
	if len(depGraphs) > 1 {
		return nil, fmt.Errorf("applying SDD to tree results in evaluation dependency graph with disconnected segments")
	}
	visitOrder, err := KahnSort(depGraphs[0])
	if err != nil {
		return nil, fmt.Errorf("sorting SDD dependency graph: %w", err)
	}

	for i := range visitOrder {
		depNode := visitOrder[i].Data

		tree := depNode.Tree
		synthetic := depNode.Synthetic
		treeParent := depNode.Parent

		var invokeOn *AnnotatedParseTree
		if synthetic {
			invokeOn = tree
		} else {
			invokeOn = treeParent
		}

		nodeRuleHead, nodeRuleProd := tree.Rule()

		bindingsToExec := sdd.BindingsFor(nodeRuleHead, nodeRuleProd, depNode.Dest)
		for j := range bindingsToExec {
			binding := bindingsToExec[j]
			binding.Invoke(invokeOn)
		}
	}

	return nil, nil
}

func (sdd *sddImpl) Bindings(head string, prod []string) []SDDBinding {
	forHead, ok := sdd.bindings[head]
	if !ok {
		return nil
	}

	forProd, ok := forHead[strings.Join(prod, " ")]
	if !ok {
		return nil
	}

	targetBindings := make([]SDDBinding, len(forProd))
	copy(targetBindings, forProd)

	return targetBindings
}

func (sdd *sddImpl) BindSynthesizedAttribute(head string, prod []string, attrName NodeAttrName, bindFunc AttributeSetter, forAttr string, withArgs []AttrRef) error {
	// sanity checks; can we even call this?
	if bindFunc == nil {
		return fmt.Errorf("cannot bind nil bindFunc")
	}

	// check args
	argErrs := ""
	for i := range withArgs {
		req := withArgs[i]
		if !req.Relation.ValidFor(head, prod) {
			argErrs += fmt.Sprintf("\n* bound-to-rule does not have a %s", req.Relation.String())
		}
	}
	if len(argErrs) > 0 {
		return fmt.Errorf("bad arguments:%s", argErrs)
	}

	// get storage slice
	bindingsForHead, ok := sdd.bindings[head]
	if !ok {
		bindingsForHead = map[string][]SDDBinding{}
	}
	defer func() { sdd.bindings[head] = bindingsForHead }()

	prodStr := strings.Join(prod, " ")
	existingBindings, ok := bindingsForHead[prodStr]
	if !ok {
		existingBindings = make([]SDDBinding, 0)
	}
	defer func() { bindingsForHead[prodStr] = existingBindings }()

	// build the binding
	bind := SDDBinding{
		Synthesized:         true,
		BoundRuleSymbol:     head,
		BoundRuleProduction: make([]string, len(prod)),
		Requirements:        make([]AttrRef, len(withArgs)),
		Setter:              bindFunc,
		Dest:                AttrRef{Relation: NodeRelation{Type: RelHead}, Name: attrName},
	}

	copy(bind.BoundRuleProduction, prod)
	copy(bind.Requirements, withArgs)
	existingBindings = append(existingBindings, bind)

	// defers will assign back up to map

	return nil
}

func (sdd *sddImpl) BindInheritedAttribute(head string, prod []string, attrName NodeAttrName, bindFunc AttributeSetter, withArgs []AttrRef, forProd NodeRelation) error {
	// sanity checks; can we even call this?
	if bindFunc == nil {
		return fmt.Errorf("cannot bind nil bindFunc")
	}

	// check forProd
	if forProd.Type == RelHead {
		return fmt.Errorf("inherited attributes not allowed to be defined on production heads")
	}
	if !forProd.ValidFor(head, prod) {
		return fmt.Errorf("bad target symbol: bound-to-rule does not have a %s", forProd.String())
	}

	// check args
	argErrs := ""
	for i := range withArgs {
		req := withArgs[i]
		if !req.Relation.ValidFor(head, prod) {
			argErrs += fmt.Sprintf("\n* bound-to-rule does not have a %s", req.Relation.String())
		}
	}
	if len(argErrs) > 0 {
		return fmt.Errorf("bad arguments:%s", argErrs)
	}

	// get storage slice
	bindingsForHead, ok := sdd.bindings[head]
	if !ok {
		bindingsForHead = map[string][]SDDBinding{}
	}
	defer func() { sdd.bindings[head] = bindingsForHead }()

	prodStr := strings.Join(prod, " ")
	existingBindings, ok := bindingsForHead[prodStr]
	if !ok {
		existingBindings = make([]SDDBinding, 0)
	}
	defer func() { bindingsForHead[prodStr] = existingBindings }()

	// build the binding
	bind := SDDBinding{
		Synthesized:         true,
		BoundRuleSymbol:     head,
		BoundRuleProduction: make([]string, len(prod)),
		Requirements:        make([]AttrRef, len(withArgs)),
		Setter:              bindFunc,
		Dest:                AttrRef{Relation: forProd, Name: attrName},
	}

	copy(bind.BoundRuleProduction, prod)
	copy(bind.Requirements, withArgs)
	existingBindings = append(existingBindings, bind)

	// defers will assign back up to map

	return nil
}

func NewSDD() *sddImpl {
	impl := sddImpl{
		map[string]map[string][]SDDBinding{},
	}
	return &impl
}

type APTNodeID uint64

const (
	IDZero APTNodeID = APTNodeID(0)
)

// IDGenerator should not be used directly, use NewIDGenerator. This will
// generate one that avoids the zero-value of APTNodeID.
type IDGenerator struct {
	avoidVals []APTNodeID
	seed      APTNodeID
	last      APTNodeID
	started   bool
}

func NewIDGenerator(seed int64) IDGenerator {
	return IDGenerator{
		seed:      APTNodeID(seed),
		avoidVals: []APTNodeID{IDZero},
	}
}

func (idGen *IDGenerator) Next() APTNodeID {
	var next APTNodeID
	var valid bool

	for !valid {
		if !idGen.started {
			// then next is set to seed-value
			idGen.started = true
			next = idGen.seed
		} else {
			next = idGen.last + 1
		}
		idGen.last = next

		valid = true
		for i := range idGen.avoidVals {
			if idGen.avoidVals[i] == next {
				valid = false
				break
			}
		}
	}

	return next
}

type NodeAttrName string

func (nan NodeAttrName) String() string {
	return string(nan)
}

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

// AddAttributes adds annotation fields to the given parse tree. Returns an
// AnnotatedParseTree with only auto fields set ('$text' for terminals, '$id'
// for all nodes).
func AddAttributes(root types.ParseTree) AnnotatedParseTree {
	treeStack := util.Stack[*types.ParseTree]{Of: []*types.ParseTree{&root}}
	annoRoot := AnnotatedParseTree{}
	annotatedStack := util.Stack[*AnnotatedParseTree]{Of: []*AnnotatedParseTree{&annoRoot}}

	idGen := NewIDGenerator(0)

	for treeStack.Len() > 0 {
		curTreeNode := treeStack.Pop()
		curAnnoNode := annotatedStack.Pop()

		curAnnoNode.Terminal = curTreeNode.Terminal
		curAnnoNode.Symbol = curTreeNode.Value
		curAnnoNode.Source = curTreeNode.Source
		curAnnoNode.Children = make([]*AnnotatedParseTree, len(curAnnoNode.Children))
		curAnnoNode.Attributes = NodeAttrs{
			NodeAttrName("$id"): NodeAttrValue(idGen.Next()),
		}

		if curTreeNode.Terminal {
			curAnnoNode.Attributes[NodeAttrName("$text")] = curAnnoNode.Source.Lexeme()
		}

		// put child nodes on stack in reverse order to get left-first
		for i := len(curTreeNode.Children) - 1; i >= 0; i-- {
			newAnnoNode := &AnnotatedParseTree{}
			curAnnoNode.Children[i] = newAnnoNode
			treeStack.Push(curTreeNode.Children[i])
			annotatedStack.Push(newAnnoNode)
		}
	}

	return annoRoot
}

type AnnotatedParseTree struct {
	// Terminal is whether this node is for a terminal symbol.
	Terminal bool

	// Symbol is the symbol at this node's head.
	Symbol string

	// Source is only available when Terminal is true.
	Source types.Token

	// Children is all children of the parse tree.
	Children []*AnnotatedParseTree

	// Attributes is the data for attributes at the given position in the parse
	// tree.
	Attributes NodeAttrs
}

// Returns the ID of this node in the parse tree. All nodes have an ID
// accessible via the special predefined attribute '$id'; this function serves
// as a shortcut to getting the value from the node attributes with casting and
// sanity checking handled.
//
// If for whatever reason the ID has not been set on this node, IDZero is
// returned.
func (apt AnnotatedParseTree) ID() APTNodeID {
	var id APTNodeID
	untyped, ok := apt.Attributes["$id"]
	if !ok {
		return id
	}

	id, ok = untyped.(APTNodeID)
	if !ok {
		panic(fmt.Sprintf("$id attribute set to non-APTNodeID typed value: %v", untyped))
	}

	return id
}

// Rule returns the head and production of the grammar rule associated with the
// creation of this node in the parse tree. If apt is for a terminal, prod will
// be empty.
func (apt AnnotatedParseTree) Rule() (head string, prod []string) {
	if apt.Terminal {
		return apt.Symbol, nil
	}

	// need to gather symbol names from created nodes
	prod = []string{}
	for i := range apt.Children {
		prod = append(prod, apt.Children[i].Symbol)
	}

	return apt.Symbol, prod
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
