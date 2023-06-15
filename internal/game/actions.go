package game

import "github.com/dekarrin/tunaq/tunascript"

// UseAction is the definition of use-item or use-detail events.
type UseAction struct {
	// With gives the labels (or tags) of things that the source item/detail has
	// to be used with in order for this event to execute. If it's empty, it
	// must be used by itself (with nothing else).
	With []string

	// If gives tunascript that must resolve to true for the event to fire. If
	// no tunascript was parsed, this will be something that always returns
	// true.
	If tunascript.AST

	// IfRaw gives the exact source tunascript that was parsed to create If. If
	// no code was parsed, this will be the empty string.
	IfRaw string

	// Do contains the tunascript that will be executed when the other
	// conditions are met.
	Do tunascript.AST

	// DoRaw gives the exact source tunascript(s) that were parsed to create Do.
	DoRaw []string
}

// Copy returns a deeply-copied UseAction.
func (ue UseAction) Copy() UseAction {
	aCopy := UseAction{
		With:  ue.With,
		If:    ue.If,
		IfRaw: ue.IfRaw,
		Do:    ue.Do,
		DoRaw: ue.DoRaw,
	}

	return aCopy
}
