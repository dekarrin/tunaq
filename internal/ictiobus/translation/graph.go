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
// InEdge leading back to dg from other.
func (dg *DirectedGraph[V]) LinkTo(other *DirectedGraph[V]) {
	dg.Edges = append(dg.Edges, other)
	other.InEdges = append(other.InEdges, dg)
}

// LinkFrom creates an out edge from the other graph to dg, and also adds an
// InEdge leading back to other from dg.
func (dg *DirectedGraph[V]) LinkFrom(other *DirectedGraph[V]) {
	other.LinkTo(dg)
}

// Copy creates a duplicate of this graph. Note that data is copied by value and
// is *not* deeply copied.
func (dg *DirectedGraph[V]) Copy() *DirectedGraph[V] {
	return dg.recursiveCopy(map[*DirectedGraph[V]]bool{})
}

func (dg *DirectedGraph[V]) recursiveCopy(visited map[*DirectedGraph[V]]bool) *DirectedGraph[V] {
	visited[dg] = true
	dgCopy := &DirectedGraph[V]{Data: dg.Data}

	// check out edges
	for i := range dg.Edges {
		to := dg.Edges[i]
		if _, alreadyVisited := visited[to]; alreadyVisited {
			continue
		}

		toCopy := to.recursiveCopy(visited)
		dgCopy.LinkTo(toCopy)
	}

	// check back edges
	for i := range dg.InEdges {
		from := dg.InEdges[i]
		if _, alreadyVisited := visited[from]; alreadyVisited {
			continue
		}
		fromCopy := from.recursiveCopy(visited)
		dgCopy.LinkFrom(fromCopy)
	}

	return dgCopy
}

// Contains returns whether the graph that the given node is in contains at any
// point the given node. Note that the contains check will check SPECIFICALLY if
// the given address is contained.
func (dg *DirectedGraph[V]) Contains(other *DirectedGraph[V]) bool {
	return dg.any(func(dg *DirectedGraph[V]) bool {
		return dg == other
	}, map[*DirectedGraph[V]]bool{})
}

func (dg *DirectedGraph[V]) forEachNode(action func(dg *DirectedGraph[V]), visited map[*DirectedGraph[V]]bool) {
	visited[dg] = true
	action(dg)

	// check out edges
	for i := range dg.Edges {
		to := dg.Edges[i]
		if _, alreadyVisited := visited[to]; alreadyVisited {
			continue
		}
		to.forEachNode(action, visited)
	}

	// check back edges
	for i := range dg.InEdges {
		from := dg.InEdges[i]
		if _, alreadyVisited := visited[from]; alreadyVisited {
			continue
		}
		from.forEachNode(action, visited)
	}
}

// AllNodes returns all nodes in the graph, in no particular order.
func (dg *DirectedGraph[V]) AllNodes() []*DirectedGraph[V] {
	gathered := new([]*DirectedGraph[V])
	*gathered = make([]*DirectedGraph[V], 0)

	onVisit := func(dg *DirectedGraph[V]) {
		*gathered = append(*gathered, dg)
	}
	dg.forEachNode(onVisit, map[*DirectedGraph[V]]bool{})

	return *gathered
}

// KahnSort takes the given graph and constructs a topological ordering for its
// nodes such that every node is placed after all nodes that eventually lead
// into it. Fails immediately if there are any cycles in the graph.
//
// This is an implementation of the algorithm published by Arthur B. Kahn in
// "Topological sorting of large networks" in Communications of the ACM, 5 (11),
// in 1962, glub! 38O
func KahnSort[V any](dg *DirectedGraph[V]) ([]*DirectedGraph[V], error) {
	// detect cycles first or we may enter an infinite loop
	if dg.HasCycles() {
		return nil, fmt.Errorf("can't apply kahn's algorithm to a graph with cycles")
	}

	// this algorithm involves modifying the graph, which we absolutely do not
	// intend to do, so make a copy and operate on that instead.
	dg = dg.Copy()

	sortedL := []*DirectedGraph[V]{}
	noIncomingS := util.NewKeySet[*DirectedGraph[V]]()

	// find all start nodes
	allNodes := dg.AllNodes()
	for i := range allNodes {
		n := allNodes[i]
		if len(n.InEdges) == 0 {
			noIncomingS.Add(n)
		}
	}

	for !noIncomingS.Empty() {
		var n *DirectedGraph[V]
		// just need to get literally any value from the set
		for nodeInSet := range noIncomingS {
			n = nodeInSet
			break
		}

		sortedL = append(sortedL, n)

		for i := range n.Edges {
			m := n.Edges[i]
			// remove all edges from n to m (instead of just 'the one' bc we
			// have no way of associated a *particular* edge with the in edge
			// on m side and there COULD be dupes)
			newNEdges := []*DirectedGraph[V]{}
			newMInEdges := []*DirectedGraph[V]{}
			for j := range n.Edges {
				if n.Edges[j] != m {
					newNEdges = append(newNEdges, n.Edges[j])
				}
			}
			for j := range m.InEdges {
				if m.InEdges[j] != n {
					newMInEdges = append(newMInEdges, m.InEdges[j])
				}
			}
			n.Edges = newNEdges
			m.InEdges = newMInEdges

			if len(m.InEdges) == 0 {
				noIncomingS.Add(m)
			}
		}
	}

	return sortedL, nil
}

// HasCycles returns whether the graph has a cycle in it at any point.
func (dg *DirectedGraph[V]) HasCycles() bool {
	// if there are no cycles, there is at least one node with no other
	finished := map[*DirectedGraph[V]]bool{}
	visited := map[*DirectedGraph[V]]bool{}

	var dfsCycleCheck func(n *DirectedGraph[V]) bool
	dfsCycleCheck = func(n *DirectedGraph[V]) bool {
		_, alreadyFinished := finished[n]
		_, alreadyVisited := visited[n]
		if alreadyFinished {
			return false
		}
		if alreadyVisited {
			return true
		}
		visited[n] = true

		for i := range n.Edges {
			cycleFound := dfsCycleCheck(n.Edges[i])
			if cycleFound {
				return true
			}
		}
		finished[n] = true
		return false
	}

	toCheck := dg.AllNodes()
	for i := range toCheck {
		n := toCheck[i]
		if dfsCycleCheck(n) {
			return true
		}
	}

	return false
}

func (dg *DirectedGraph[V]) any(predicate func(*DirectedGraph[V]) bool, visited map[*DirectedGraph[V]]bool) bool {
	visited[dg] = true
	if predicate(dg) {
		return true
	}

	// check out edges
	for i := range dg.Edges {
		to := dg.Edges[i]
		if _, alreadyVisited := visited[to]; alreadyVisited {
			continue
		}
		toMatches := to.any(predicate, visited)
		if toMatches {
			return true
		}
	}

	// check back edges
	for i := range dg.InEdges {
		from := dg.InEdges[i]
		if _, alreadyVisited := visited[from]; alreadyVisited {
			continue
		}
		fromMatches := from.any(predicate, visited)
		if fromMatches {
			return true
		}
	}

	return false
}

type DepNode struct {
	Parent    *AnnotatedParseTree
	Tree      *AnnotatedParseTree
	Synthetic bool
	Dest      AttrRef
}

// Info on this func from 5.2.1 dragon book, purple.
//
// Returns one node from each of the connected sub-graphs of the dependency
// tree. If the entire dependency graph is connected, there will be only 1 item
// in the returned slice.
func DepGraph(aptRoot AnnotatedParseTree, sdd *sddImpl) []*DirectedGraph[DepNode] {
	type treeAndParent struct {
		Tree   *AnnotatedParseTree
		Parent *AnnotatedParseTree
	}
	// no parent set on first node; it's the root
	treeStack := util.Stack[treeAndParent]{Of: []treeAndParent{{Tree: &aptRoot}}}

	// TODO: yeahhhhhhhhhhh this should probs be map of APTNodeID -> map[AttrRef]*DG
	// or SOMEFIN that isn't subject to attribute name collision, which this is
	// atm.
	depNodes := map[APTNodeID]map[AttrRef]*DirectedGraph[DepNode]{}
	for treeStack.Len() > 0 {
		curTreeAndParent := treeStack.Pop()
		curTree := curTreeAndParent.Tree
		curParent := curTreeAndParent.Parent

		// what semantic rule would apply to this?
		ruleHead, ruleProd := curTree.Rule()
		binds := sdd.Bindings(ruleHead, ruleProd)

		// sanity check each node on visit to be shore it's got a non-empty ID.
		if curTree.ID() == IDZero {
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
				relNode, ok := curTree.RelativeNode(req.Relation)
				if !ok {
					panic(fmt.Sprintf("relative address cannot be followed: %v", req.Relation.String()))
				}
				relNodeID := relNode.ID()
				relNodeDepNodes, ok := depNodes[relNodeID]
				if !ok {
					relNodeDepNodes = map[AttrRef]*DirectedGraph[DepNode]{}
				}
				// specifically, need to address the one for the desired attribute
				fromDepNode, ok := relNodeDepNodes[req]
				if !ok {
					relParent := curParent
					if relNode != curTree {
						// then relNode MUST be a child of curTreeNode
						relParent = curTree
					}
					fromDepNode = &DirectedGraph[DepNode]{Data: DepNode{
						// we simply have no idea whether this is a synthetic
						// attribute or not at this time
						Parent: relParent, Tree: relNode,
					}}
				}

				// get the TARGET node
				targetNode, ok := curTree.RelativeNode(binding.Dest.Relation)
				if !ok {
					panic(fmt.Sprintf("relative address cannot be followed: %v", req.Relation.String()))
				}
				targetNodeID := targetNode.ID()
				targetNodeDepNodes, ok := depNodes[targetNodeID]
				if !ok {
					targetNodeDepNodes = map[AttrRef]*DirectedGraph[DepNode]{}
				}
				targetParent := curParent
				synthTarget := true
				if targetNode != curTree {
					// then targetNode MUST be a child of curTreeNode
					targetParent = curTree

					// additionally, it cannot be synthetic because it is not
					// being set at the head of a production
					synthTarget = false
				}
				// specifically, need to address the one for the desired attribute
				toDepNode, ok := targetNodeDepNodes[binding.Dest]
				if !ok {
					toDepNode = &DirectedGraph[DepNode]{Data: DepNode{
						Parent: targetParent, Tree: targetNode, Dest: binding.Dest, Synthetic: synthTarget,
					}}
				}
				// but also, if it DOES already exist we might have created it
				// without knowing whether it is a synthetic attr; either way,
				// check it now
				toDepNode.Data.Synthetic = synthTarget
				toDepNode.Data.Dest = binding.Dest

				// create the edge; this will modify BOTH dep nodes
				fromDepNode.LinkTo(toDepNode)

				// make shore to assign after modification (shouldn't NEED
				// to due to attrDepNode being ptr-to but do it just to be
				// safe)
				relNodeDepNodes[req] = fromDepNode
				targetNodeDepNodes[binding.Dest] = toDepNode
				depNodes[relNodeID] = relNodeDepNodes
				depNodes[targetNodeID] = targetNodeDepNodes
			}
		}

		// put child nodes on stack in reverse order to get left-first
		for i := len(curTree.Children) - 1; i >= 0; i-- {
			treeStack.Push(treeAndParent{Parent: curTree, Tree: curTree.Children[i]})
		}
	}

	// TODO: go through the entire graph and enshore it is totally connected
	var connectedSubGraphs []*DirectedGraph[DepNode]

	for k := range depNodes {
		idDepNodes := depNodes[k]
		for attrRef := range idDepNodes {
			node := idDepNodes[attrRef]
			if len(node.Edges) > 0 || len(node.InEdges) > 0 {
				// we found a non-empty node, include that

				// first, is this already in a graph we've grabbed? no need to
				// keep it if so
				var alreadyHaveGraph bool
				for i := range connectedSubGraphs {
					prevSub := connectedSubGraphs[i]
					if prevSub.Contains(node) {
						alreadyHaveGraph = true
						break
					}
				}
				if !alreadyHaveGraph {
					connectedSubGraphs = append(connectedSubGraphs, node)
				}
			}
		}
	}

	return connectedSubGraphs
}
