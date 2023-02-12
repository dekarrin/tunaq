package translation

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
