// Package buffalo contains parsers and parser-generator constructs used as part
// of research into compiling techniques. It is the tunascript compilers pulled
// out after it turned from "small knowledge gaining side-side project" into
// full-blown compilers and translators research.
//
// It's based off of the name for the buffalo fish due to the buffalo's relation
// with bison. Naturally, bison due to its popularity as a parser-generator
// tool.
//
// This will probably never be as good as bison, so consider using that. This is
// for research and does not seek to replace existing toolchains in any
// practical fashion.
package ictiobus

// HACKING NOTE:
//
// https://jsmachines.sourceforge.net/machines/lalr1.html is an AMAZING tool for
// validating LALR(1) grammars quickly.

import (
	"io"

	"github.com/dekarrin/tunaq/internal/ictiobus/grammar"
	"github.com/dekarrin/tunaq/internal/ictiobus/lex"
	"github.com/dekarrin/tunaq/internal/ictiobus/parse"
	"github.com/dekarrin/tunaq/internal/ictiobus/translation"
	"github.com/dekarrin/tunaq/internal/ictiobus/types"
)

type Parser interface {
	// Parse parses input text and returns the parse tree built from it, or a
	// SyntaxError with the description of the problem.
	Parse(stream types.TokenStream) (types.ParseTree, error)
}

type Lexer interface {
	// Lex returns a token stream. The tokens may be lexed in a lazy fashion or
	// an immediate fashion; if it is immediate, errors will be returned at that
	// point. If it is lazy, then error token productions will be returned to
	// the callers of the returned TokenStream at the point where the error
	// occured.
	Lex(input io.Reader) (types.TokenStream, error)
	RegisterClass(cl types.TokenClass, forState string)
	AddPattern(pat string, action lex.Action, forState string) error

	SetStartingState(s string)
	StartingState() string
}

// SDD is a series of syntax-directed definitions bound to syntactic rules of
// a grammar. It is used for evaluation of a parse tree into an intermediate
// representation, or for direct execution.
type SDD interface {

	// BindInheritedAttribute creates a new SDD binding for setting the value of
	// an inherited attribute with name attrName. The production that the
	// inherited attribute is set on is specified with forProd, which must have
	// its Type set to something other than RelHead (inherited attributes can be
	// set only on production symbols).
	//
	// The binding applies only on nodes in the parse tree created by parsing
	// the grammar rule productions with head symbol head and production symbols
	// prod.
	//
	// The AttributeSetter bindFunc is called when the inherited value attrName
	// is to be set, in order to calculate the new value. Attribute values to
	// pass in as arguments are specified by passing references to the node and
	// attribute name whose value to retrieve in the withArgs slice. Explicitly
	// giving the referenced attributes in this fashion makes it easy to
	// determine the dependency graph for later execution.
	BindInheritedAttribute(head string, prod []string, attrName translation.NodeAttrName, bindFunc translation.AttributeSetter, withArgs []translation.AttrRef, forProd translation.NodeRelation) error

	// BindSynthesizedAttribute creates a new SDD binding for setting the value
	// of a synthesized attribute with name attrName. The attribute is set on
	// the symbol at the head of the rule that the binding is being created for.
	//
	// The binding applies only on nodes in the parse tree created by parsing
	// the grammar rule productions with head symbol head and production symbols
	// prod.
	//
	// The AttributeSetter bindFunc is called when the synthesized value
	// attrName is to be set, in order to calculate the new value. Attribute
	// values to pass in as arguments are specified by passing references to the
	// node and attribute name whose value to retrieve in the withArgs slice.
	// Explicitly giving the referenced attributes in this fashion makes it easy
	// to determine the dependency graph for later execution.
	BindSynthesizedAttribute(head string, prod []string, attrName translation.NodeAttrName, bindFunc translation.AttributeSetter, forAttr string, withArgs []translation.AttrRef) error

	// Bindings returns all bindings defined to apply when at a node in a parse
	// tree created by the rule production with head as its head symbol and prod
	// as its produced symbols. They will be returned in the order they were
	// defined.
	Bindings(head string, prod []string) []translation.SDDBinding

	BindingsFor(head string, prod []string, dest translation.AttrRef) []translation.SDDBinding

	// Evaluate takes a parse tree and executes the semantic actions defined as
	// SDDBindings for a node for each node in the tree and on completion,
	// returns the requested attributes values from the root node. Execution
	// order is automatically determined by taking the dependency graph of the
	// SDD; cycles are not supported. Do note that this does not require the SDD
	// to be S-attributed or L-attributed, only that it not have cycles in its
	// value dependency graph.
	Evaluate(tree types.ParseTree, attributes ...translation.NodeAttrName) ([]translation.NodeAttrValue, error)
}

// NewLexer returns a lexer whose Lex method will immediately lex the entire
// input source, finding errors and reporting them and stopping as soon as the
// first lexing error is encountered or the input has been completely lexed.
//
// The TokenStream returned by the Lex function is guaranteed to not have any
// error tokens.
func NewLexer() Lexer {
	return lex.NewLexer(false)
}

// NewLazyLexer returns a Lexer whose Lex method will return a TokenStream that
// is lazily executed; that is to say, calling Next() on the token stream will
// perform only enough lexical analysis to produce the next token. Additionally,
// that TokenStream may produce an error token, which parsers would need to
// handle appropriately.
func NewLazyLexer() Lexer {
	return lex.NewLexer(true)
}

// NewParser returns what is the most flexible and efficient parser in this
// package. At this time, that is the LALR(1) parser. Returns an error if the
// grammar cannot be parsed by an LALR parser.
func NewParser(g grammar.Grammar) (Parser, error) {
	return NewLALR1Parser(g)
}

// NewLALR1Parser returns an LALR(1) parser that can generate parse trees for
// the given grammar. Returns an error if the grammar is not LALR(1).
func NewLALR1Parser(g grammar.Grammar) (Parser, error) {
	return parse.GenerateLALR1Parser(g)
}

// NewSLRParser returns an SLR(1) parser that can generate parse trees for the
// given grammar. Returns an error if the grammar is not SLR(1).
func NewSLRParser(g grammar.Grammar) (Parser, error) {
	return parse.GenerateSimpleLRParser(g)
}

// NewLL1Parser returns an LL(1) parser that can generate parse trees for the
// given grammar. Returns an error if the grammar is not LL(1).
func NewLL1Parser(g grammar.Grammar) (Parser, error) {
	return parse.GenerateLL1Parser(g)
}

// NewCLRParser returns a canonical-LR(0) parser that can generate parse trees
// for the given grammar. Returns an error if the grammar is not CLR(1)
func NewCLRParser(g grammar.Grammar) (Parser, error) {
	return parse.GenerateCanonicalLR1Parser(g)
}
