package syntax

// TODO: this file is here as part of migration to an ictiobus generated
// frontend. Replace it with references to ByID() in the output frontend.

const (
	// if newline is put in any of these it will break the lexer detection of
	// position so don't do that

	// also the tests are all hardcoded and that is by design as it implies that
	// if you change something here, you must also update any tests that rely on
	// it.

	literalStrStringQuote     = "@"
	literalStrSeparator       = ","
	literalStrGroupOpen       = "("
	literalStrGroupClose      = ")"
	literalStrIdentifierStart = "$"
	literalStrOpSet           = "="
	literalStrOpIs            = "=="
	literalStrOpIsNot         = "!="
	literalStrOpLessThan      = "<"
	literalStrOpGreaterThan   = ">"
	literalStrOpLessThanIs    = "<="
	literalStrOpGreaterThanIs = ">="
	literalStrOpAnd           = "&&"
	literalStrOpOr            = "||"
	literalStrOpNot           = "!"
	literalStrOpPlus          = "+"
	literalStrOpMinus         = "-"
	literalStrOpMultiply      = "*"
	literalStrOpDivide        = "/"
	literalStrOpIncset        = "+="
	literalStrOpDecset        = "-="
	literalStrOpInc           = "++"
	literalStrOpDec           = "--"
)
