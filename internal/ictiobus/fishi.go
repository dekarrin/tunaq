package ictiobus

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dekarrin/tunaq/internal/ictiobus/grammar"
	"github.com/dekarrin/tunaq/internal/ictiobus/lex"
	"github.com/dekarrin/tunaq/internal/ictiobus/types"
	"github.com/gomarkdown/markdown"
	mkast "github.com/gomarkdown/markdown/ast"
	mkparser "github.com/gomarkdown/markdown/parser"
)

type fishiScanner bool

func (fs fishiScanner) RenderNode(w io.Writer, node mkast.Node, entering bool) mkast.WalkStatus {
	if !entering {
		return mkast.GoToNext
	}

	codeBlock, ok := node.(*mkast.CodeBlock)
	if !ok || codeBlock == nil {
		return mkast.GoToNext
	}

	if strings.ToLower(strings.TrimSpace(string(codeBlock.Info))) == "fishi" {
		w.Write(codeBlock.Literal)
	}
	return mkast.GoToNext
}

func (fs fishiScanner) RenderHeader(w io.Writer, ast mkast.Node) {}
func (fs fishiScanner) RenderFooter(w io.Writer, ast mkast.Node) {}

func ReadFishiMdFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	err = ProcessFishiMd(data)
	if err != nil {
		return err
	}

	return nil
}

func ProcessFishiMd(mdText []byte) error {

	// debug steps: output source after preprocess
	// output token stream
	// output grammar constructed
	// output parser table and type

	fishiSource := GetFishiFromMarkdown(mdText)
	fishiSource = Preprocess(fishiSource)
	fishi := bytes.NewBuffer(fishiSource)

	lx := CreateBootstrapLexer()
	stream, err := lx.Lex(fishi)
	if err != nil {
		return err
	}
	fmt.Println("------------------------------------------------------------------")

	g := CreateBootstrapGrammarFromLexerStream(stream)
	if err := g.Validate(); err != nil {
		return err
	}

	// now, can we make a parser from this?
	var parser Parser

	parser, ambigWarns, err := NewCLRParser(g, true)
	if err != nil {
		return err
	}
	parser.RegisterTraceListener(func(s string) {
		fmt.Printf(">> %s\n", strings.ReplaceAll(s, "\n", "\n   "))
	})

	for i := range ambigWarns {
		fmt.Printf("warn: ambiguous grammar: %s\n", ambigWarns[i])
	}

	fmt.Printf("successfully built %s parser:\n%v\n", parser.Type().String(), parser)

	// now, try to make a parse tree for your own grammar
	fishiSource = []byte(`%%grammar
{RULE}

%%grammar

{RULE}
	`)
	fishiSource = Preprocess(fishiSource)
	fishi = bytes.NewBuffer(fishiSource)
	stream, err = lx.Lex(fishi)
	if err != nil {
		return err
	}
	pt, err := parser.Parse(stream)
	if err != nil {
		return err
	}
	fmt.Printf("successfully parsed own spec:\n")
	fmt.Printf("%s\n", pt.String())

	return nil
}

func CreateBootstrapGrammarFromLexerStream(lx types.TokenStream) grammar.Grammar {
	bootCfg := grammar.Grammar{}

	bootCfg.AddTerm(tcHeaderTokens.ID(), tcHeaderTokens)
	bootCfg.AddTerm(tcHeaderGrammar.ID(), tcHeaderGrammar)
	bootCfg.AddTerm(tcHeaderActions.ID(), tcHeaderActions)
	bootCfg.AddTerm(tcDirAction.ID(), tcDirAction)
	//bootCfg.AddTerm(tcDirDefault.ID(), tcDirDefault)
	bootCfg.AddTerm(tcDirHook.ID(), tcDirHook)
	bootCfg.AddTerm(tcDirHuman.ID(), tcDirHuman)
	bootCfg.AddTerm(tcDirIndex.ID(), tcDirIndex)
	bootCfg.AddTerm(tcDirProd.ID(), tcDirProd)
	bootCfg.AddTerm(tcDirShift.ID(), tcDirShift)
	//bootCfg.AddTerm(tcDirStart.ID(), tcDirStart)
	bootCfg.AddTerm(tcDirState.ID(), tcDirState)
	bootCfg.AddTerm(tcDirSymbol.ID(), tcDirSymbol)
	bootCfg.AddTerm(tcDirToken.ID(), tcDirToken)
	bootCfg.AddTerm(tcDirWith.ID(), tcDirWith)
	bootCfg.AddTerm(tcFreeformText.ID(), tcFreeformText)
	bootCfg.AddTerm(tcNewline.ID(), tcNewline)
	bootCfg.AddTerm(tcTerminal.ID(), tcTerminal)
	bootCfg.AddTerm(tcNonterminal.ID(), tcNonterminal)
	bootCfg.AddTerm(tcEq.ID(), tcEq)
	bootCfg.AddTerm(tcAlt.ID(), tcAlt)
	bootCfg.AddTerm(tcAttrRef.ID(), tcAttrRef)
	bootCfg.AddTerm(tcInt.ID(), tcInt)
	bootCfg.AddTerm(tcId.ID(), tcId)
	bootCfg.AddTerm(tcEscseq.ID(), tcEscseq)
	bootCfg.AddTerm(tcEpsilon.ID(), tcEpsilon)

	bootCfg.AddRule("FISHISPEC", []string{"BLOCKS"})

	bootCfg.AddRule("BLOCKS", []string{"BLOCKS", "BLOCK"})
	bootCfg.AddRule("BLOCKS", []string{"BLOCK"})

	bootCfg.AddRule("BLOCK", []string{"GRAMMAR-BLOCK"})

	bootCfg.AddRule("GRAMMAR-BLOCK", []string{tcHeaderGrammar.ID(), "GRAMMAR-CONTENT"})
	bootCfg.AddRule("GRAMMAR-BLOCK", []string{tcHeaderGrammar.ID(), "NEWLINES", "GRAMMAR-CONTENT"})

	bootCfg.AddRule("GRAMMAR-CONTENT", []string{"GRAMMAR-CONTENT", "GRAMMAR-STATE-BLOCK"})
	bootCfg.AddRule("GRAMMAR-CONTENT", []string{"GRAMMAR-STATE-BLOCK"})

	//	bootCfg.AddRule("GRAMMAR-STATE-BLOCK", []string{tcDirState.ID(), "GRAMMAR-RULES"})
	bootCfg.AddRule("GRAMMAR-STATE-BLOCK", []string{"GRAMMAR-RULES"})

	bootCfg.AddRule("GRAMMAR-RULES", []string{"GRAMMAR-RULES", "NEWLINES", "GRAMMAR-RULE"})
	bootCfg.AddRule("GRAMMAR-RULES", []string{"GRAMMAR-RULE"})

	bootCfg.AddRule("GRAMMAR-RULE", []string{tcNonterminal.ID()})

	/*

		bootCfg.AddRule("GRAMMAR-BLOCK", []string{tcHeaderGrammar.ID(), "GRAMMAR-CONTENT"})
		bootCfg.AddRule("GRAMMAR-BLOCK", []string{tcHeaderGrammar.ID(), "NEWLINES", "GRAMMAR-CONTENT"})

		bootCfg.AddRule("GRAMMAR-CONTENT", []string{"GRAMMAR-STATE-BLOCK"})
		bootCfg.AddRule("GRAMMAR-CONTENT", []string{"GRAMMAR-CONTENT", "GRAMMAR-STATE-BLOCK"})

		bootCfg.AddRule("GRAMMAR-STATE-BLOCK", []string{"GRAMMAR-RULES"})

		bootCfg.AddRule("GRAMMAR-RULES", []string{"GRAMMAR-RULES", "NEWLINES", "GRAMMAR-RULE"})
		bootCfg.AddRule("GRAMMAR-RULES", []string{"GRAMMAR-RULE"})

		bootCfg.AddRule("GRAMMAR-RULE", []string{tcNonterminal.ID(), tcEq.ID(), "ALTERNATIONS"})

		bootCfg.AddRule("ALTERNATIONS", []string{"ALTERNATIONS", tcAlt.ID(), "PRODUCTION"})
		bootCfg.AddRule("ALTERNATIONS", []string{"PRODUCTION"})

		bootCfg.AddRule("PRODUCTION", []string{"SYMBOLS"})
		bootCfg.AddRule("PRODUCTION", []string{tcEpsilon.ID()})

		bootCfg.AddRule("SYMBOLS", []string{"SYMBOLS", "SYMBOL"})
		bootCfg.AddRule("SYMBOLS", []string{"SYMBOL"})

		bootCfg.AddRule("SYMBOL", []string{tcEq.ID()})
		bootCfg.AddRule("SYMBOL", []string{tcNonterminal.ID()})
	*/
	bootCfg.AddRule("NEWLINES", []string{"NEWLINES", tcNewline.ID()})
	bootCfg.AddRule("NEWLINES", []string{tcNewline.ID()})

	bootCfg.Start = "FISHISPEC"
	bootCfg.RemoveUnusedTerminals()

	/*

			var inGrammarBlock bool
			var atEntryStart bool
			var forRule string
			var waitForEq bool
			var gettingProds bool
			var curProd []string
			var prodStarted bool

			convNonTerm := func(s string) string {
				// chop off leading/trailing {}'s
				s = s[1 : len(s)-1]

				return strings.ToUpper(s)
			}

			convTerm := func(s string) string {
				return strings.ToLower(s)
			}

			firstRule := ""

			for lx.HasNext() {
				tok := lx.Next()
				if !inGrammarBlock {
					if tok.Class().ID() == tcHeaderGrammar.ID() {
						inGrammarBlock = true
						atEntryStart = true
						waitForEq = false
						gettingProds = false
						prodStarted = false
						curProd = nil
						continue
					}
				}

				if !inGrammarBlock {
					continue
				}

				switch tok.Class().ID() {
				case tcHeaderActions.ID():
					fallthrough
				case tcHeaderTokens.ID():
					if prodStarted {
						if len(curProd) == 0 {
							curProd = append(curProd, "")
						}
						bootCfg.AddRule(forRule, curProd)
						curProd = nil
						prodStarted = false
					}
					inGrammarBlock = false
					continue
				case tcNewline.ID():
					atEntryStart = true
					waitForEq = false
					gettingProds = false
					continue
				case tcNonterminal.ID():
					if atEntryStart {
						if prodStarted {
							if len(curProd) == 0 {
								curProd = append(curProd, "")
							}
							bootCfg.AddRule(forRule, curProd)
							curProd = nil
							prodStarted = false
						}
						forRule = convNonTerm(tok.Lexeme())
						if firstRule == "" {
							firstRule = forRule
						}
						atEntryStart = false
						waitForEq = true
						gettingProds = false
						curProd = nil
						continue
					} else if gettingProds {
						curProd = append(curProd, convNonTerm(tok.Lexeme()))
					} else {
						fmt.Println(icterrors.NewSyntaxErrorFromToken("didn't expect a non-terminal", tok).FullMessage())
						panic("fail")
					}
				case tcTerminal.ID():
					if gettingProds {
						term := convTerm(tok.Lexeme())
						curProd = append(curProd, term)
					} else {
						fmt.Println(icterrors.NewSyntaxErrorFromToken("didn't expect a terminal", tok).FullMessage())
						panic("fail")
					}
				case tcAlt.ID():
					if atEntryStart {
						atEntryStart = false
						gettingProds = true

						if len(curProd) == 0 {
							curProd = append(curProd, "")
						}
						bootCfg.AddRule(forRule, curProd)
						curProd = nil
					} else if gettingProds {

						if len(curProd) == 0 {
							curProd = append(curProd, "")
						}
						bootCfg.AddRule(forRule, curProd)
						curProd = nil
					} else {
						fmt.Println(icterrors.NewSyntaxErrorFromToken("didn't expect an alt", tok).FullMessage())
						panic("fail")
					}
				case tcEq.ID():
					if waitForEq {
						prodStarted = true
						waitForEq = false
						gettingProds = true
						continue
					} else {
						fmt.Println(icterrors.NewSyntaxErrorFromToken("didn't expect an eq", tok).FullMessage())
						panic("fail")
					}
				default:
					fmt.Println(icterrors.NewSyntaxErrorFromToken("bootstrap cannot handle token", tok).FullMessage())
					panic("fail")
				}
			}

			if prodStarted {

				if len(curProd) == 0 {
					curProd = append(curProd, "")
				}
				bootCfg.AddRule(forRule, curProd)
				curProd = nil
				prodStarted = false
			}


		bootCfg.Start = firstRule
	*/
	return bootCfg
}

func GetFishiFromMarkdown(mdText []byte) []byte {
	doc := markdown.Parse(mdText, mkparser.New())
	var scanner fishiScanner
	fishi := markdown.Render(doc, scanner)
	return fishi
}

// Preprocess does a preprocess step on the source, which as of now includes
// stripping comments and normalizing end of lines to \n.
func Preprocess(source []byte) []byte {
	toBuf := make([]byte, len(source))
	copy(toBuf, source)
	scanner := bufio.NewScanner(bytes.NewBuffer(toBuf))
	var preprocessed strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, "\r\n") || strings.HasPrefix(line, "\n\r") {
			line = line[0 : len(line)-2]
		} else {
			line = strings.TrimSuffix(line, "\n")
		}
		line, _, _ = strings.Cut(line, "#")
		preprocessed.WriteString(line)
		preprocessed.WriteRune('\n')
	}

	return []byte(preprocessed.String())
}

var (

	// %default is not in this version, not needed to self-describe
	//tcDirDefault    = lex.NewTokenClass("default_dir", "'default' directive")

	// %start is not in this version, not needed to self-describe
	//tcDirStart      = lex.NewTokenClass("start_dir", "'start' directive")

	tcHeaderTokens  = lex.NewTokenClass("tokens_header", "'tokens' header")
	tcHeaderGrammar = lex.NewTokenClass("grammar_header", "'grammar' header")
	tcHeaderActions = lex.NewTokenClass("actions_header", "'actions' header")
	tcDirAction     = lex.NewTokenClass("action_dir", "'action' directive")
	tcDirHook       = lex.NewTokenClass("hook_dir", "'hook' directive")
	tcDirHuman      = lex.NewTokenClass("human_dir", "'human' directive")
	tcDirIndex      = lex.NewTokenClass("index_dir", "'index' directive")
	tcDirProd       = lex.NewTokenClass("prod_dir", "'prod' directive")
	tcDirShift      = lex.NewTokenClass("shift_dir", "'stateshift' directive")
	tcDirState      = lex.NewTokenClass("state_dir", "'state' directive")
	tcDirSymbol     = lex.NewTokenClass("symbol_dir", "'symbol' directive")
	tcDirToken      = lex.NewTokenClass("token_dir", "'token' directive")
	tcDirWith       = lex.NewTokenClass("with_dir", "'with' directive")
	tcFreeformText  = lex.NewTokenClass("freeform_text", "freeform text value")
	tcNewline       = lex.NewTokenClass("newline", "'\\n'")
	tcTerminal      = lex.NewTokenClass("terminal", "terminal symbol")
	tcNonterminal   = lex.NewTokenClass("nonterminal", "non-terminal symbol")
	tcEq            = lex.NewTokenClass("eq", "'='")
	tcAlt           = lex.NewTokenClass("alt", "'|'")
	tcAttrRef       = lex.NewTokenClass("attr_ref", "attribute reference")
	tcInt           = lex.NewTokenClass("int", "integer value")
	tcId            = lex.NewTokenClass("id", "identifier")
	tcEscseq        = lex.NewTokenClass("escseq", "escape sequence")
	tcEpsilon       = lex.NewTokenClass("epsilon", "epsilon production")
)

func CreateBootstrapLexer() Lexer {
	bootLx := NewLexer()

	// default state, shared by all
	bootLx.RegisterClass(tcEscseq, "")
	bootLx.RegisterClass(tcHeaderTokens, "")
	bootLx.RegisterClass(tcHeaderGrammar, "")
	bootLx.RegisterClass(tcHeaderActions, "")
	//bootLx.RegisterClass(tcDirStart, "")
	bootLx.RegisterClass(tcDirState, "")

	// default patterns and defs
	bootLx.AddPattern(`%!.`, lex.LexAs(tcEscseq.ID()), "")
	bootLx.AddPattern(`%%[Tt][Oo][Kk][Ee][Nn][Ss]`, lex.LexAndSwapState(tcHeaderTokens.ID(), "tokens"), "")
	bootLx.AddPattern(`%%[Gg][Rr][Aa][Mm][Mm][Aa][Rr]`, lex.LexAndSwapState(tcHeaderGrammar.ID(), "grammar"), "")
	bootLx.AddPattern(`%%[Aa][Cc][Tt][Ii][Oo][Nn][Ss]`, lex.LexAndSwapState(tcHeaderActions.ID(), "actions"), "")
	//bootLx.AddPattern(`%[Ss][Tt][Aa][Rr][Tt]`, lex.LexAs(tcDirStart.ID()), "")
	bootLx.AddPattern(`%[Ss][Tt][Aa][Tt][Ee]`, lex.LexAs(tcDirState.ID()), "")

	// tokens classes
	bootLx.RegisterClass(tcFreeformText, "tokens")
	bootLx.RegisterClass(tcDirShift, "tokens")
	bootLx.RegisterClass(tcDirHuman, "tokens")
	bootLx.RegisterClass(tcDirToken, "tokens")
	//bootLx.RegisterClass(tcDirDefault, "tokens")
	bootLx.RegisterClass(tcNewline, "tokens")

	// tokens patterns
	bootLx.AddPattern(`%[Sa][Tt][Aa][Tt][Ee][Ss][Hh][Ii][Ff][Tt]`, lex.LexAs(tcDirShift.ID()), "tokens")
	bootLx.AddPattern(`%[Hh][Uu][Mm][Aa][Nn]`, lex.LexAs(tcDirHuman.ID()), "tokens")
	bootLx.AddPattern(`%[Tt][Oo][Kk][Ee][Nn]`, lex.LexAs(tcDirToken.ID()), "tokens")
	//bootLx.AddPattern(`%[Dd][Ee][Ff][Aa][Uu][Ll][Tt]`, lex.LexAs(tcDirDefault.ID()), "tokens")
	bootLx.AddPattern(`\n`, lex.LexAs(tcNewline.ID()), "tokens")
	bootLx.AddPattern(`[^%\n]+`, lex.LexAs(tcFreeformText.ID()), "tokens")

	// grammar classes
	bootLx.RegisterClass(tcNewline, "grammar")
	bootLx.RegisterClass(tcEq, "grammar")
	bootLx.RegisterClass(tcAlt, "grammar")
	bootLx.RegisterClass(tcNonterminal, "grammar")
	bootLx.RegisterClass(tcTerminal, "grammar")
	bootLx.RegisterClass(tcEpsilon, "grammar")

	// gramamr patterns
	bootLx.AddPattern(`%[Ee][Pp][Ss]`, lex.LexAs(tcEpsilon.ID()), "grammar")
	bootLx.AddPattern(`\n`, lex.LexAs(tcNewline.ID()), "grammar")
	bootLx.AddPattern(`[^\S\n]+`, lex.Discard(), "grammar")
	bootLx.AddPattern(`=`, lex.LexAs(tcEq.ID()), "grammar")
	bootLx.AddPattern(`\|`, lex.LexAs(tcAlt.ID()), "grammar")
	bootLx.AddPattern(`{}`, lex.LexAs(tcEpsilon.ID()), "grammar")
	bootLx.AddPattern(`{[A-Za-z][^}]*}`, lex.LexAs(tcNonterminal.ID()), "grammar")
	bootLx.AddPattern(`\S+`, lex.LexAs(tcTerminal.ID()), "grammar")

	// actions classes
	bootLx.RegisterClass(tcAttrRef, "actions")
	bootLx.RegisterClass(tcInt, "actions")
	bootLx.RegisterClass(tcNonterminal, "actions")
	bootLx.RegisterClass(tcDirSymbol, "actions")
	bootLx.RegisterClass(tcDirProd, "actions")
	bootLx.RegisterClass(tcDirWith, "actions")
	bootLx.RegisterClass(tcDirHook, "actions")
	bootLx.RegisterClass(tcDirAction, "actions")
	bootLx.RegisterClass(tcDirIndex, "actions")
	bootLx.RegisterClass(tcId, "actions")
	bootLx.RegisterClass(tcTerminal, "actions")

	// actions patterns
	bootLx.AddPattern(`\s+`, lex.Discard(), "actions")
	bootLx.AddPattern(`(?:{[A-Za-z][^}]*}|\S+)(?:\$\d+)?\.[\$A-Za-z][$A-Za-z0-9_-]*`, lex.LexAs(tcAttrRef.ID()), "actions")
	bootLx.AddPattern(`[0-9]+`, lex.LexAs(tcInt.ID()), "actions")
	bootLx.AddPattern(`{[A-Za-z][^}]*}`, lex.LexAs(tcNonterminal.ID()), "actions")
	bootLx.AddPattern(`%[Ss][Yy][Mm][Bb][Oo][Ll]`, lex.LexAs(tcDirSymbol.ID()), "actions")
	bootLx.AddPattern(`%[Pp][Rr][Oo][Dd]`, lex.LexAs(tcDirProd.ID()), "actions")
	bootLx.AddPattern(`%[Ww][Ii][Tt][Hh]`, lex.LexAs(tcDirWith.ID()), "actions")
	bootLx.AddPattern(`%[Hh][Oo][Oo][Kk]`, lex.LexAs(tcDirHook.ID()), "actions")
	bootLx.AddPattern(`%[Aa][Cc][Tt][Ii][Oo][Nn]`, lex.LexAs(tcDirAction.ID()), "actions")
	bootLx.AddPattern(`%[Ii][Nn][Dd][Ee][Xx]`, lex.LexAs(tcDirIndex.ID()), "actions")
	bootLx.AddPattern(`[A-Za-z][A-Za-z0-9_-]*`, lex.LexAs(tcId.ID()), "actions")
	bootLx.AddPattern(`\S+`, lex.LexAs(tcTerminal.ID()), "actions")

	return bootLx
}
