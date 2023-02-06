// Package buffalo contains parsers and parser-generator constructs used as part
// of research into compiling techniques. It is the tunascript compilers pulled
// out after it turned from "small knowledge gaining side-side project" into
// full-blown compilers and translators research.
//
// It's based off of the buffalo fish and also bison because of course.
//
// This will probably never be as good as bison, so consider using that. This is
// for research and does not seek to replace existing toolchains in any
// practical fashion.
package buffalo

import (
	"github.com/dekarrin/tunaq/internal/buffalo/lex"
	"github.com/dekarrin/tunaq/internal/buffalo/parse"
)

type Parser interface {
	// Parse parses input text and returns the parse tree built from it, or a
	// SyntaxError with the description of the problem.
	Parse(stream lex.TokenStream) (parse.Tree, error)
}
