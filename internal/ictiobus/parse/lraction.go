package parse

import (
	"fmt"

	"github.com/dekarrin/tunaq/internal/ictiobus/grammar"
)

type LRActionType int

const (
	LRShift LRActionType = iota
	LRReduce
	LRAccept
	LRError
)

type LRAction struct {
	Type LRActionType

	// Production is used when Type is LRReduce. It is the production which
	// should be reduced; the β of A -> β.
	Production grammar.Production

	// Symbol is used when Type is LRReduce. It is the symbol to reduce the
	// production to; the A of A -> β.
	Symbol string

	// State is the state to shift to. It is used only when Type is LRShift.
	State string
}

func (act LRAction) String() string {
	switch act.Type {
	case LRAccept:
		return "ACTION<accept>"
	case LRError:
		return "ACTION<error>"
	case LRReduce:
		return fmt.Sprintf("ACTION<reduce %s -> %s>", act.Symbol, act.Production.String())
	case LRShift:
		return fmt.Sprintf("ACTION<shift %s>", act.State)
	default:
		return "ACTION<unknown>"
	}
}

func (act LRAction) Equal(o any) bool {
	other, ok := o.(LRAction)
	if !ok {
		otherPtr := o.(*LRAction)
		if !ok {
			return false
		}
		if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if act.Type != other.Type {
		return false
	} else if !act.Production.Equal(other.Production) {
		return false
	} else if act.State != other.State {
		return false
	} else if act.Symbol != other.Symbol {
		return false
	}

	return true
}
