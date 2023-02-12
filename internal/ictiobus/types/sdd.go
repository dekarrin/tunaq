package types

// SyntaxDirectedDefinition is a function that gives the value for a node, based
// on the left and right siblings (if they exist).
type SyntaxDirectedDefinition func(node ParseTree, leftSiblings []ParseTree, rightSiblings []ParseTree) any
