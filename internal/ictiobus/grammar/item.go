package grammar

import (
	"fmt"
	"strings"

	"github.com/dekarrin/tunaq/internal/util"
)

type LR0Item struct {
	NonTerminal string
	Left        []string
	Right       []string
}

func (lr0 LR0Item) Equal(o any) bool {
	other, ok := o.(LR0Item)
	if !ok {
		otherPtr, ok := o.(*LR0Item)
		if !ok {
			return false
		}
		if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if lr0.NonTerminal != other.NonTerminal {
		return false
	} else if len(lr0.Left) != len(other.Left) {
		return false
	} else if len(lr0.Right) != len(other.Right) {
		return false
	}

	// now check the left and right
	for i := range lr0.Left {
		if lr0.Left[i] != other.Left[i] {
			return false
		}
	}
	for i := range lr0.Right {
		if lr0.Right[i] != other.Right[i] {
			return false
		}
	}

	return true
}

type LR1Item struct {
	LR0Item
	Lookahead string
}

func EqualCoreSets(s1, s2 util.VSet[string, LR1Item]) bool {
	return CoreSet(s1).Equal(CoreSet(s2))
}

func CoreSet(s util.VSet[string, LR1Item]) util.SVSet[LR0Item] {
	cores := util.NewSVSet[LR0Item]()
	for _, elem := range s.Elements() {
		lr1 := s.Get(elem)
		cores.Set(lr1.LR0Item.String(), lr1.LR0Item)
	}

	return cores
}

func (lr1 LR1Item) Equal(o any) bool {
	other, ok := o.(LR1Item)
	if !ok {
		otherPtr, ok := o.(*LR1Item)
		if !ok {
			return false
		}
		if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if !lr1.LR0Item.Equal(other.LR0Item) {
		return false
	} else if lr1.Lookahead != other.Lookahead {
		return false
	}

	return true
}

func (lr1 LR1Item) Copy() LR1Item {
	lrCopy := LR1Item{}
	lrCopy.NonTerminal = lr1.NonTerminal
	lrCopy.Left = make([]string, len(lr1.Left))
	copy(lrCopy.Left, lr1.Left)
	lrCopy.Right = make([]string, len(lr1.Right))
	copy(lrCopy.Right, lr1.Right)
	lrCopy.Lookahead = lr1.Lookahead

	return lrCopy
}

func MustParseLR0Item(s string) LR0Item {
	i, err := ParseLR0Item(s)
	if err != nil {
		panic(err.Error())
	}
	return i
}

func MustParseLR1Item(s string) LR1Item {
	i, err := ParseLR1Item(s)
	if err != nil {
		panic(err.Error())
	}
	return i
}

func ParseLR0Item(s string) (LR0Item, error) {
	sides := strings.Split(s, "->")
	if len(sides) != 2 {
		return LR0Item{}, fmt.Errorf("not an item of form 'NONTERM -> ALPHA.BETA': %q", s)
	}
	nonTerminal := strings.TrimSpace(sides[0])

	if nonTerminal == "" {
		return LR0Item{}, fmt.Errorf("empty nonterminal name not allowed for item")
	}

	parsedItem := LR0Item{
		NonTerminal: nonTerminal,
	}

	productionsString := strings.TrimSpace(sides[1])
	prodStrings := strings.Split(productionsString, ".")
	if len(prodStrings) != 2 {
		return LR0Item{}, fmt.Errorf("item must have exactly one dot")
	}

	alphaStr := strings.TrimSpace(prodStrings[0])
	betaStr := strings.TrimSpace(prodStrings[1])

	alphaSymbols := strings.Split(alphaStr, " ")
	betaSymbols := strings.Split(betaStr, " ")

	var parsedAlpha, parsedBeta []string

	for _, aSym := range alphaSymbols {
		aSym = strings.TrimSpace(aSym)

		if aSym == "" {
			continue
		}

		if strings.ToLower(aSym) == "ε" {
			// epsilon production
			aSym = ""
		}

		parsedAlpha = append(parsedAlpha, aSym)
	}

	for _, bSym := range betaSymbols {
		bSym = strings.TrimSpace(bSym)

		if bSym == "" {
			continue
		}

		if strings.ToLower(bSym) == "ε" {
			// epsilon production
			bSym = ""
		}

		parsedBeta = append(parsedBeta, bSym)
	}

	parsedItem.Left = parsedAlpha
	parsedItem.Right = parsedBeta

	return parsedItem, nil
}

func ParseLR1Item(s string) (LR1Item, error) {
	sides := strings.Split(s, ",")
	if len(sides) != 2 {
		return LR1Item{}, fmt.Errorf("not an item of form 'NONTERM -> ALPHA.BETA, a': %q", s)
	}

	item := LR1Item{}
	var err error
	item.LR0Item, err = ParseLR0Item(sides[0])
	if err != nil {
		return item, err
	}

	item.Lookahead = strings.TrimSpace(sides[1])

	return item, nil
}

func (item LR0Item) String() string {
	nonTermPhrase := ""
	if item.NonTerminal != "" {
		nonTermPhrase = fmt.Sprintf("%s -> ", item.NonTerminal)
	}

	left := strings.Join(item.Left, " ")
	right := strings.Join(item.Right, " ")

	if len(left) > 0 {
		left = left + " "
	}
	if len(right) > 0 {
		right = " " + right
	}

	return fmt.Sprintf("%s%s.%s", nonTermPhrase, left, right)
}

func (item LR1Item) String() string {
	return fmt.Sprintf("%s, %s", item.LR0Item.String(), item.Lookahead)
}
