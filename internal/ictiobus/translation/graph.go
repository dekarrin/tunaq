package translation

import (
	"fmt"

	"github.com/dekarrin/tunaq/internal/util"
)

// DirectedGraph is a node in a graph whose edges point in one direction to
// another node. This implementation can carry data in the nodes of the graph.
//
// DirectedGraph's zero-value can be used directly.
type DirectedGraph[V any] struct {
	// Edges is a set of references to nodes this node goes to (that it has an
	// edge pointing to).
	Edges []*DirectedGraph[V]

	// InEdges is a set of back-references to nodes that go to (have an edge
	// that point towards) this node.
	InEdges []*DirectedGraph[V]

	// Data is the value held at this node of the graph.
	Data V
}

// LinkTo creates an out edge from dg to the other graph, and also adds an
// InEdge leading back to dg to other.
func (dg *DirectedGraph[V]) LinkTo(other *DirectedGraph[V]) {
	dg.Edges = append(dg.Edges, other)
	other.InEdges = append(other.InEdges, dg)
}

// LinkFrom creates an out edge from the other graph to dg, and also adds an
// InEdge leading back to other to dg.
func (dg *DirectedGraph[V]) LinkFrom(other *DirectedGraph[V]) {
	other.LinkTo(dg)
}

type DepNode struct {
	Tree      *AnnotatedParseTree
	Attribute NodeAttrName
}

// Info on this func from 5.2.1 dragon book, purple.
func DepGraph(aptRoot AnnotatedParseTree, sdd SDD) *DirectedGraph[DepNode] {
	treeStack := util.Stack[*AnnotatedParseTree]{Of: []*AnnotatedParseTree{&aptRoot}}

	depNodes := map[APTNodeID]map[NodeAttrName]*DirectedGraph[DepNode]{}

	for treeStack.Len() > 0 {
		curTreeNode := treeStack.Pop()

		// what semantic rule would apply to this?
		ruleHead, ruleProd := curTreeNode.Rule()
		binds := sdd.Bindings(ruleHead, ruleProd)

		// sanity check each node on visit to be shore it's got a non-empty ID.
		if curTreeNode.ID() == IDZero {
			panic("ID not set on APT node")
		}

		for i := range binds {
			binding := binds[i]
			if len(binding.Requirements) < 1 {
				continue
			}
			for j := range binding.Requirements {
				req := binding.Requirements[j]

				// get the related node:
				relNode, ok := curTreeNode.RelativeNode(req.Relation)
				if !ok {
					panic(fmt.Sprintf("relative address cannot be followed: %v", req.Relation.String()))
				}
				relNodeID := relNode.ID()
				relNodeDepNodes, ok := depNodes[relNodeID]
				if !ok {
					relNodeDepNodes = map[NodeAttrName]*DirectedGraph[DepNode]{}
				}
				// specifically, need to address the one for the desired attribute
				fromDepNode, ok := relNodeDepNodes[req.Name]
				if !ok {
					fromDepNode = &DirectedGraph[DepNode]{Data: DepNode{
						Tree: relNode, Attribute: req.Name,
					}}
				}

				// get the TARGET node
				targetNode, ok := curTreeNode.RelativeNode(binding.Dest.Relation)
				if !ok {
					panic(fmt.Sprintf("relative address cannot be followed: %v", req.Relation.String()))
				}
				targetNodeID := targetNode.ID()
				targetNodeDepNodes, ok := depNodes[targetNodeID]
				if !ok {
					targetNodeDepNodes = map[NodeAttrName]*DirectedGraph[DepNode]{}
				}
				// specifically, need to address the one for the desired attribute
				toDepNode, ok := targetNodeDepNodes[binding.Dest.Name]
				if !ok {
					toDepNode = &DirectedGraph[DepNode]{Data: DepNode{
						Tree: targetNode, Attribute: binding.Dest.Name,
					}}
				}

				// create the edge; this will modify BOTH dep nodes
				fromDepNode.LinkTo(toDepNode)

				// make shore to assign after modification (shouldn't NEED
				// to due to attrDepNode being ptr-to but do it just to be
				// safe)
				relNodeDepNodes[req.Name] = fromDepNode
				targetNodeDepNodes[binding.Dest.Name] = toDepNode
				depNodes[relNodeID] = relNodeDepNodes
				depNodes[targetNodeID] = targetNodeDepNodes
			}
		}

		// put child nodes on stack in reverse order to get left-first
		for i := len(curTreeNode.Children) - 1; i >= 0; i-- {
			treeStack.Push(curTreeNode.Children[i])
		}
	}

	// TODO: go through the entire graph and enshore it is totally connected

	// honestly, any node returning will do
	var anyNode *DirectedGraph[DepNode]

	for k := range depNodes {
		idDepNodes := depNodes[k]
		for attrName := range idDepNodes {
			node := idDepNodes[attrName]
			if len(node.Edges) > 0 || len(node.InEdges) > 0 {
				// we found a non-empty node, use that as the return value as
				// from there we should be able to get to the entire graph
				// (ASSUMPTION: check that with above TODO)
				anyNode = node
				break
			}
		}
		if anyNode != nil {
			break
		}
	}

	return anyNode
}
