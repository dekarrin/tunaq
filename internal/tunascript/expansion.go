package tunascript

import (
	"fmt"
	"strings"

	"github.com/dekarrin/tunaq/internal/util"
)

// indexOfMatchingParen takes the given string, which must start with
// a parenthesis char "(", and returns the index of the ")" that matches it. Any
// text in between is analyzed for other parenthesis and if they are there, they
// must be matched as well.
//
// Index will be -1 if it does not have a match.
// Error is non-nil if there is unlex-able tunascript syntax between the parens,
// of if s cannot be operated on.
func indexOfMatchingParen(sRunes []rune) (int, AST, error) {
	if sRunes[0] != '(' {
		var errStr string
		if len(sRunes) > 50 {
			errStr = string(sRunes[:50]) + "..."
		} else {
			errStr = string(sRunes)
		}
		return 0, AST{}, SyntaxError{
			message: fmt.Sprintf("no %s at start of analysis string %q", literalStrGroupOpen, errStr),
		}
	}

	if len(sRunes) < 2 {
		return 0, AST{}, SyntaxError{
			message: "unexpected end of expression (unmatched '" + literalStrGroupOpen + "')",
		}
	}

	tokenStr, consumed, err := lexRunes(sRunes, true)
	if err != nil {
		return 0, AST{}, err
	}

	// check that we got minimum tokens
	if tokenStr.Len() < 3 {
		// not enough tokens; at minimum we require lparen, rparen, and EOT.
		return 0, AST{}, SyntaxError{
			message: "unexpected end of expression (unmatched '" + literalStrGroupOpen + "')",
		}
	}
	// check that we ended on a right paren (will be second-to-last bc last is EOT)
	if tokenStr.tokens[len(tokenStr.tokens)-2].class.id != tsGroupClose.id {
		// in this case, lexing got to the end of the string but did not finish
		// on a right paren. This is a syntax error.
		return 0, AST{}, SyntaxError{
			message: "unexpected end of expression (unmatched '" + literalStrGroupOpen + "')",
		}
	}

	// modify returned list of tokens to not include the start and end parens
	// before parsing
	eotLexeme := tokenStr.tokens[len(tokenStr.tokens)-1]          // preserve EOT
	tokenStr.tokens = tokenStr.tokens[1 : len(tokenStr.tokens)-2] // chop off ends
	tokenStr.tokens = append(tokenStr.tokens, eotLexeme)          // add EOT back in

	// now parse it to get back the actual AST
	ast, err := Parse(tokenStr)
	if err != nil {
		return 0, ast, err
	}

	return consumed, ast, nil
}

// SyntaxCheckTree executes every branch of the tree without giving output to
// ensure that every branch can be parsed. If this function does not return an
// an error, the same tree passed to ExpandTree should never return an error.
//
// If checkFlags, This also ensures every flag reference in the tree refers
// to an existing flag.
func (inter Interpreter) SyntaxCheckTree(ast *ExpansionAST, checkFlags bool) error {
	if ast == nil {
		return fmt.Errorf("nil ast")
	}

	for i := range ast.nodes {
		n := ast.nodes[i]

		if n.flag != nil {
			// if flag is not
			if checkFlags {
				flUpper := strings.ToUpper(n.flag.name[1:])
				_, ok := inter.flags[flUpper]
				if !ok {
					return fmt.Errorf("%q is not a defined flag", flUpper)
				}
			}
		} else if n.text != nil {
			// do no checking, we dont care about looking at raw text
		} else if n.branch != nil {
			cond := n.branch.ifNode.cond
			contentExpansionAST := n.branch.ifNode.content

			conditionalValue, err := inter.invoke(*cond, true)
			if err != nil {
				return fmt.Errorf("syntax error: %v", err)
			}
			if len(conditionalValue) != 1 {
				return fmt.Errorf("incorrect number of arguments to $IF; must be exactly 1")
			}

			// regardless of value of conditional, do every branch
			_, err = inter.ExpandTree(contentExpansionAST)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (inter Interpreter) ExpandTree(ast *ExpansionAST) (string, error) {
	if ast == nil {
		return "", fmt.Errorf("nil ast")
	}

	expandFlag := func(fullFlagToken string) string {
		flagName := fullFlagToken[1:]
		if flagName == "" {
			// bare identifier start, go add it to expanded
			return literalStrIdentifierStart
		} else {
			// it is a full var name
			flagName = strings.ToUpper(flagName)
			flag, ok := inter.flags[flagName]
			if !ok {
				return ""
			}
			return flag.Value.String()
		}
	}

	usb := util.UndoableStringBuilder{}

	for i := range ast.nodes {
		n := ast.nodes[i]

		if n.flag != nil {
			flagVal := expandFlag(n.flag.name)
			usb.WriteString(flagVal)
		} else if n.text != nil {
			usb.WriteString(n.text.t)
		} else if n.branch != nil {
			cond := n.branch.ifNode.cond
			contentExpansionAST := n.branch.ifNode.content

			conditionalValue, err := inter.invoke(*cond, true)
			if err != nil {
				return "", fmt.Errorf("syntax error: %v", err)
			}
			if len(conditionalValue) != 1 {
				return "", fmt.Errorf("incorrect number of arguments to $IF; must be exactly 1")
			}

			if conditionalValue[0].Bool() {
				expandedContent, err := inter.ExpandTree(contentExpansionAST)
				if err != nil {
					return "", err
				}

				expandedContent = strings.TrimSpace(expandedContent)
				usb.WriteString(expandedContent)
			} else {
				// get rid of extra space here if it's not still too slow

				var prevTextHasSpace, nextTextHasSpace bool

				// is there trailing space char in prior text block?
				if i > 0 {
					prevNode := ast.nodes[i-1]
					if prevNode.text != nil {
						textNode := prevNode.text
						if textNode.minusSpaceSuffix != nil {
							prevTextHasSpace = true
						}
					}
				}

				// is there a leading space char in next text block?
				if i+1 < len(ast.nodes) {
					nextNode := ast.nodes[i+1]
					if nextNode.text != nil {
						textNode := nextNode.text
						if textNode.minusSpacePrefix != nil {
							nextTextHasSpace = true
						}
					}
				}

				// if both have a space char, eliminate the prior one
				if prevTextHasSpace && nextTextHasSpace {
					usb.Undo()

					prevText := ast.nodes[i-1].text.minusSpaceSuffix

					usb.WriteString(*prevText)
				}
			}
		}
	}

	str := usb.String()
	return str, nil
}

// ParseExpansion applies expansion analysis to the given text.
//
//   - any flag reference with the $ will be expanded to its full value.
//   - any $IF() ... $ENDIF() block will be evaluated and included in the output
//     text only if the tunaquest expression inside the $IF evaluates to true.
//   - function calls are not allowed outside of the tunascript expression in an
//     $IF. If they are there, they will be interpreted as a variable expansion,
//     and if there is no value matching that one, it will be expanded to an
//     empty string. E.g. "$ADD()" in the body text would evaluate to value of
//     flag called "ADD" (probably ""), followed by literal parenthesis.
//   - bare dollar signs are evaluated as literal. This will only happen if they
//     are not immediately followed by identifier chars.
//   - literal $ signs can be included with a backslash. Thus the escape
//     backslash will work.
//   - literal backslashes can be included by escaping them.
func (inter Interpreter) ParseExpansion(s string) (*ExpansionAST, error) {
	sRunes := []rune{}
	sBytes := []int{}
	for b, ch := range s {
		sRunes = append(sRunes, ch)
		sBytes = append(sBytes, b)
	}

	ast, _, err := inter.parseExpansion(sRunes, sBytes, true)
	return ast, err
}

func (inter Interpreter) parseExpansion(sRunes []rune, sBytes []int, topLevel bool) (*ExpansionAST, int, error) {
	tree := &ExpansionAST{
		nodes: make([]expTreeNode, 0),
	}

	const (
		modeText = iota
		modeIdent
	)

	var ident strings.Builder

	var escaping bool

	curText := strings.Builder{}
	mode := modeText

	buildTextNode := func(s string) *expTextNode {
		var noPre, noSuf *string
		if strings.HasPrefix(s, " ") {
			noPre = new(string)
			*noPre = strings.TrimPrefix(s, " ")
		}
		if strings.HasSuffix(s, " ") {
			noSuf = new(string)
			*noSuf = strings.TrimSuffix(s, " ")
		}
		return &expTextNode{
			t:                s,
			minusSpacePrefix: noPre,
			minusSpaceSuffix: noSuf,
		}

	}

	for i := 0; i < len(sRunes); i++ {
		ch := sRunes[i]
		switch mode {
		case modeText:
			if !escaping && ch == '\\' {
				escaping = true
			} else if !escaping && ch == '$' {
				if curText.Len() > 0 {
					lastText := curText.String()
					tree.nodes = append(tree.nodes, expTreeNode{
						text: buildTextNode(lastText),
					})
					curText.Reset()
				}

				ident.WriteRune('$')
				mode = modeIdent
			} else {
				curText.WriteRune(ch)
			}
		case modeIdent:
			if ('A' <= ch && ch <= 'Z') || ('a' <= ch && ch <= 'z') || ('0' <= ch && ch <= '9') || ch == '_' {
				ident.WriteRune(ch)
			} else if ch == '(' {
				fnName := ident.String()

				if fnName == "$IF" {
					// we've encountered an IF block, recurse.
					parenMatch, tsExpr, err := indexOfMatchingParen(sRunes[i:])
					if err != nil {
						return tree, 0, fmt.Errorf("at char %d: %w", i, err)
					}
					exprLen := parenMatch - 1

					if exprLen < 1 {
						return tree, 0, fmt.Errorf("at char %d: args cannot be empty", i)
					}

					branch := expBranchNode{
						ifNode: expCondNode{
							cond: &tsExpr,
						},
					}

					i += parenMatch

					if i+1 >= len(sRunes) {
						return nil, 0, fmt.Errorf("unexpected end of text (unmatched $IF)")
					}

					ast, consumed, err := inter.parseExpansion(sRunes[i+1:], sBytes[i+1:], false)
					if err != nil {
						return nil, 0, err
					}

					branch.ifNode.content = ast

					tree.nodes = append(tree.nodes, expTreeNode{
						branch: &branch,
					})

					i += consumed

					ident.Reset()

					i++ // to skip the closing paren that the recursed call detected and returned on
					mode = modeText
				} else if fnName == "$ENDIF" {
					parenMatch, tsExpr, err := indexOfMatchingParen(sRunes[i:])
					if err != nil {
						return nil, 0, fmt.Errorf("at char %d: %w", i, err)
					}
					exprLen := parenMatch - 1

					if exprLen != 0 {
						return nil, 0, fmt.Errorf("at char %d: $ENDIF() takes zero arguments, received %d", i, len(tsExpr.nodes))
					}
					i += parenMatch

					if topLevel {
						return nil, 0, fmt.Errorf("unexpected end of text (unmatched $ENDIF)")
					}

					return tree, i, nil
				} else {
					return nil, 0, fmt.Errorf("at char %d: %s() is not a text function; only $IF() or $ENDIF() are allowed", i, ident.String())
				}
			} else {
				flagName := ident.String()

				tree.nodes = append(tree.nodes, expTreeNode{
					flag: &expFlagNode{
						name: flagName,
					},
				})

				mode = modeText
				i-- // reparse 'normally'
			}
		default:
			// should never happen
			return nil, 0, fmt.Errorf("unknown parser mode: %v", mode)
		}
	}

	if !topLevel {
		return nil, 0, fmt.Errorf("unexpected end of text (unmatched $IF)")
	}

	if curText.Len() > 0 {
		lastText := curText.String()
		tree.nodes = append(tree.nodes, expTreeNode{
			text: buildTextNode(lastText),
		})
		curText.Reset()
	}

	if ident.Len() > 0 {
		flagName := ident.String()

		tree.nodes = append(tree.nodes, expTreeNode{
			flag: &expFlagNode{
				name: flagName,
			},
		})
	}

	return tree, len(sRunes), nil
}

// Expand applies expansion on the given text. Expansion will expand the
// following constructs:
//
//   - any flag reference with the $ will be expanded to its full value.
//   - any $IF() ... $ENDIF() block will be evaluated and included in the output
//     text only if the tunaquest expression inside the $IF evaluates to true.
//   - function calls are not allowed outside of the tunascript expression in an
//     $IF. If they are there, they will be interpreted as a variable expansion,
//     and if there is no value matching that one, it will be expanded to an
//     empty string. E.g. "$ADD()" in the body text would evaluate to value of
//     flag called "ADD" (probably ""), followed by literal parenthesis.
//   - bare dollar signs are evaluated as literal. This will only happen if they
//     are not immediately followed by identifier chars.
//   - literal $ signs can be included with a backslash. Thus the escape
//     backslash will work.
//   - literal backslashes can be included by escaping them.
func (inter Interpreter) Expand(s string) (string, error) {
	expAST, err := inter.ParseExpansion(s)
	if err != nil {
		return "", err
	}

	expanded, err := inter.ExpandTree(expAST)
	if err != nil {
		return "", err
	}

	return expanded, nil
}
