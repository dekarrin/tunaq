package parse

import (
	"fmt"

	"github.com/dekarrin/tunaq/internal/ictiobus/grammar"
)

func isShiftReduceConlict(act1, act2 LRAction) (isSR bool, shiftAct LRAction) {
	if act1.Type == LRReduce && act2.Type == LRShift {
		return true, act2
	}
	if act2.Type == LRReduce && act1.Type == LRShift {
		return true, act1
	}

	return false, act1
}

func makeLRConflictError(act1, act2 LRAction, onInput string) error {
	if act1.Type == LRReduce && act2.Type == LRShift || act1.Type == LRShift && act2.Type == LRReduce {
		// shift-reduce conflict

		reduceRule := ""
		if act1.Type == LRReduce {
			reduceRule = act1.Symbol + " -> " + act1.Production.String()
		} else {
			reduceRule = act2.Symbol + " -> " + act2.Production.String()
		}
		return fmt.Errorf("shift/reduce conflict detected on terminal %q (shift or reduce %s)", onInput, reduceRule)
	} else if act1.Type == LRReduce && act2.Type == LRReduce {
		// reduce-reduce conflict

		reduce1 := act1.Symbol + " -> " + act1.Production.String()
		reduce2 := act2.Symbol + " -> " + act2.Production.String()
		return fmt.Errorf("reduce/reduce conflict detected on terminal %q (reduce %s or reduce %s)", onInput, reduce1, reduce2)
	} else if act1.Type == LRAccept || act2.Type == LRAccept {
		nonAcceptAct := act2

		if act2.Type == LRAccept {
			nonAcceptAct = act1
		}

		// accept-? conflict
		if nonAcceptAct.Type == LRShift {
			return fmt.Errorf("accept/shift conflict detected on terminal %q", onInput)
		} else if nonAcceptAct.Type == LRReduce {
			reduce := nonAcceptAct.Symbol + " -> " + nonAcceptAct.Production.String()
			return fmt.Errorf("accept/reduce conflict detected on terminal %q (accept or reduce %s)", onInput, reduce)
		}
	} else if act1.Type == LRShift && act2.Type == LRShift {
		return fmt.Errorf("(!) shift/shift conflict on terminal %q", onInput)
	}
	return fmt.Errorf("LR action conflict on terminal %q (%s or %s)", onInput, act1.String(), act2.String())
}

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
