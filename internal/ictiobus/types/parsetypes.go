package types

type ParserType string

const (
	ParserLL1   ParserType = "LL(1)"
	ParserSLR1  ParserType = "SLR(1)"
	ParserCLR1  ParserType = "CLR(1)"
	ParserLALR1 ParserType = "LALR(1)"
)

func (pt ParserType) String() string {
	return string(pt)
}
