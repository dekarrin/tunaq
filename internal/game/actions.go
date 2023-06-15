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

// selectBestUseMatch selects the best candidate from several matches.
func selectBestUseMatch(matches []useMatch) useMatch {
	// okay, we now have a set of candidate use matches. Let's filter them down
	// and get it down to one
	if len(matches) > 1 {
		// 1. first, if *any* are the main item being used, we default to that.
		newMatches := []useMatch{}
		var mainFound bool
		for _, m := range matches {
			if m.main {
				mainFound = true
			}
		}
		for _, m := range matches {
			if m.main || !mainFound {
				newMatches = append(newMatches, m)
			}
		}

		matches = newMatches
	}

	if len(matches) > 1 {
		// 2. take the most specific one(s) only
		newMatches := []useMatch{}
		highestSpecific := -1
		for _, m := range matches {
			if m.specific > highestSpecific {
				highestSpecific = m.specific
			}
		}

		for _, m := range matches {
			if m.specific == highestSpecific {
				newMatches = append(newMatches, m)
			}
		}

		matches = newMatches
	}

	if len(matches) > 1 {
		// 3. Take the first found.
		matches = []useMatch{matches[0]}
	}

	return matches[0]
}
