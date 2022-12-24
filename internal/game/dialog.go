package game

import "fmt"

type DialogAction int

const (
	DialogEnd DialogAction = iota
	DialogLine
	DialogChoice
)

func (dst DialogAction) String() string {
	switch dst {
	case DialogEnd:
		return "END"
	case DialogLine:
		return "LINE"
	case DialogChoice:
		return "CHOICE"
	default:
		return fmt.Sprintf("DialogAction(%d)", int(dst))
	}
}

// DialogStep is a single step of a Dialog tree. It instructions a Dialog as to
// what should happen in it and how to do it. A step either specifies an end,
// a line, or a choice.
type DialogStep struct {
	Action DialogAction

	// The line of dialog. Not used if action is END.
	Content string

	// How the player responds to the line. Only used if action is LINE, else
	// use choices.
	Response string

	// Choices and the label of dialog step they map to. If one isn't given, it
	// is assumed to end the conversation.
	Choices map[string]string
}

// Conversation includes the dialog tree and the current position within it. It
// can be created simply by manually creating a Conversation and assigning a
// sequence of steps to Dialog.
type Conversation struct {
	Dialog []DialogStep
	cur    int
}

// NextStep gets the next DialogStep in the conversation. If it returns a
// DialogStep with an Action of DialogEnd, the dialog is over and there is
// nothing further to do. If it returns a DialogStep with an Action of
// DialogLine, NextStep can be safely called to go to the next DialogStep after
// that one is processed. If it returns a DialogStep with an Action of
// DialogChoice, JumpTo should be used to jump to the given alias before calling
// NextStep again.
func (convo *Conversation) NextStep() DialogStep {
	if convo.cur >= len(convo.Dialog) {
		return DialogStep{Action: DialogEnd}
	}

	current := convo.Dialog[convo.cur]
	convo.cur++
	return current
}

// JumpTo makes the Conversation JumpTo the step with the given label, so that
// the next call to NextStep returns that DialogStep.
func (convo *Conversation) JumpTo(label string) {

}
