package lex

type ActionType int

const (
	ActionNone ActionType = iota
	ActionScan
	ActionState
	ActionScanAndState
)

type Action struct {
	Type    ActionType
	ClassID string
	State   string
}

func SwapState(toState string) Action {
	return Action{
		Type:  ActionState,
		State: toState,
	}
}

func LexAs(classID string) Action {
	return Action{
		Type:    ActionScan,
		ClassID: classID,
	}
}

func LexAndSwapState(classID string, newState string) Action {
	return Action{
		Type:    ActionScanAndState,
		ClassID: classID,
		State:   newState,
	}
}

func Discard() Action {
	return Action{}
}
