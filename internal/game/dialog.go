package game

import "fmt"

type DialogStepType int

const (
	DialogEnd DialogStepType = iota
	DialogLine
	DialogChoice
)

func (dst DialogStepType) String() string {
	switch dst {
	case DialogEnd:
		return "END"
	case DialogLine:
		return "LINE"
	case DialogChoice:
		return "CHOICE"
	default:
		return fmt.Sprintf("DialogStepType(%d)", int(dst))
	}
}

// DialogStep is a single step of a Dialog tree. It instructions a Dialog as to
// what should happen in it and how to do it. A step either specifies an end,
// a line, or a choice.
type DialogStep struct {
}
