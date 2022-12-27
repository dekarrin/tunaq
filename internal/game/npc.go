package game

import (
	"fmt"
)

// This file contains structs and routines related to NPCs.

// PronounSet is a set of pronouns for an NPC that can be used by code that
// generates references to NPCs without being specific to a particular one. You
// can create your own Pronouns struct, or just stick to the previously-defined
// ones. These should be all upper-case.
type PronounSet struct {
	// Nominative is the pronoun used in the nominative case. 'SHE', 'HE', or
	// 'THEY' for example; the pronoun that would be used to replace 'NPC' in
	// the following sentence: "NPC WENT TO THE STORE."
	Nominative string

	// Objective is the pronoun used in the objective case. 'HER', 'HIM', or
	// 'THEM' for example; the pronoun that would be used to replace 'NPC' in
	// the following sentence: "YOU TALK TO NPC."
	Objective string

	// Possessive the pronoun used in the possesive case. 'HERS', 'HIS', or
	// 'THEIRS' for example; the pronoun that would be used to replace "NPC'S"
	// in the following sentence: "THAT ITEM IS NPC'S."
	Possessive string

	// Determiner is the pronoun used in the possesive case when using a pronoun
	// as an adjective of some noun to show ownership. 'HER', 'HIS', or 'THEIR'
	// for example; the pronoun that would be used to replace "NPC's" in the
	// following sentence: "THAT IS NPC'S ITEM."
	Determiner string

	// Reflexive is the pronoun used to show the pronoun-user refering back to
	// themself. 'HERSELF', 'HIMSELF', or 'THEMSELF' for example; the pronoun
	// that would be used to replace "NPCSELF" is the following sentence: "NPC
	// THINKS ABOUT NPCSELF A LOT".
	Reflexive string
}

var (

	// PronounsFeminine is the predefined set of feminine pronouns, commonly
	// referred to as "she/her" pronouns.
	PronounsFeminine = PronounSet{"SHE", "HER", "HERS", "HER", "HERSELF"}

	// PronounsMasculine is the predefined set of masculine pronouns, commonly
	// referred to as "he/him" pronouns.
	PronounsMasculine = PronounSet{"HE", "HIM", "HIS", "HIS", "HIMSELF"}

	// PronounsNonBinary is the predefined set of non-binary pronouns, commonly
	// referred to as "they/them" pronouns.
	PronounsNonBinary = PronounSet{"THEY", "THEM", "THEIRS", "THEIR", "THEMSELF"}

	// PronounsItIts is the predefined set of pronouns commonly referred to as
	// "it/its".
	PronounsItIts = PronounSet{"IT", "IT", "ITS", "ITS", "ITSELF"}
)

// RouteAction is the type of action that a route has an NPC take.
type RouteAction int

const (
	RouteStatic RouteAction = iota
	RoutePatrol
	RouteWander
)

func (ra RouteAction) String() string {
	switch ra {
	case RouteStatic:
		return "STATIC"
	case RoutePatrol:
		return "PATROL"
	case RouteWander:
		return "WANDER"
	default:
		return fmt.Sprintf("RouteAction(%d)", int(ra))
	}
}

// RouteActionsByString is a map indexing string values to their corresponding
// RouteAction.
var RouteActionsByString map[string]RouteAction = map[string]RouteAction{
	RouteStatic.String(): RouteStatic,
	RoutePatrol.String(): RoutePatrol,
	RouteWander.String(): RouteWander,
}

// Route is a type of movement for an NPC to take
type Route struct {
	// Action is the type of action the route has the NPC move. RouteStatic is
	// not moving, RoutePatrol is follow the steps in 'Patrol', RouteWander is
	// to wander about but stay within AllowedRooms (if defined) or out of
	// ForbiddenRooms (if defined).
	Action RouteAction

	// Path is the steps that the route takes, by their room labels. It is
	// only used if Action is set to RoutePatrol
	Path []string

	// AllowedRooms is the list of rooms by their label that wandering movement
	// will stay within. It is only used if Action is set to RouteWander. If
	// neither this nor ForbiddenRooms has entries, the NPC is permitted to
	// wander anywhere. If both are set and contain the same entry,
	// ForbiddenRooms takes precedent and the room will be forbidden.
	AllowedRooms []string

	// ForbiddenRooms is the list of rooms by their label that wandering
	// movement will stay out of. It is only used if Action is set to
	// RouteWander. If neither this nor AllowedRooms has entries, the NPC is
	// permitted to wander anywhere. If both are set and contain the same entry,
	// ForbiddenRooms takes precedent and the room will be forbidden.
	ForbiddenRooms []string
}

// Copy returns a deeply-copied Route.
func (r Route) Copy() Route {
	rCopy := Route{
		Action:         r.Action,
		Path:           make([]string, len(r.Path)),
		AllowedRooms:   make([]string, len(r.AllowedRooms)),
		ForbiddenRooms: make([]string, len(r.ForbiddenRooms)),
	}

	copy(rCopy.Path, r.Path)
	copy(rCopy.AllowedRooms, r.AllowedRooms)
	copy(rCopy.ForbiddenRooms, r.ForbiddenRooms)

	return rCopy
}

func (r Route) Route() string {
	str := fmt.Sprintf("Route<%q", r.Action)

	switch r.Action {
	case RouteStatic:
		return str + ">"
	case RoutePatrol:
		str += fmt.Sprintf(" path=[")
		for idx, p := range r.Path {
			str += fmt.Sprintf("%q", p)
			if idx+1 < len(r.Path) {
				str += ", "
			}
		}
		str += "]>"
		return str
	case RouteWander:
		str += fmt.Sprintf(" allowed=[")
		for idx, ar := range r.AllowedRooms {
			str += fmt.Sprintf("%q", ar)
			if idx+1 < len(r.AllowedRooms) {
				str += ", "
			}
		}
		str += "], forbidden=["
		for idx, fr := range r.ForbiddenRooms {
			str += fmt.Sprintf("%q", fr)
			if idx+1 < len(r.ForbiddenRooms) {
				str += ", "
			}
		}
		str += "]>"
		return str
	default:
		return str + " (UNKNOWN TYPE)>"
	}
}

// NPC is a NonPlayerCharacter in the world. They may move between rooms
// depending on their Movement property and can be talked to by the player. If
// they can move, they only do so when the player does.
type NPC struct {
	// Label is a name for the NPC and canonical way to index it
	// programmatically. It should be upper case and MUST be unique within all
	// labels of the world.
	Label string

	// Name is the short description of the NPC.
	Name string

	// Pronouns is used to programatically refer to the NPC in auto-generated
	// phrases.
	Pronouns PronounSet

	// Description is the long description given when a player LOOKs at the NPC.
	Description string

	// Start is the label of the room that the NPC is in at the start of the
	// game.
	Start string

	// Movement is the type of Movement that the NPC engages in.
	Movement Route

	// Dialog is the Dialog tree that the NPC engages in with the player.
	Dialog []DialogStep

	// Convo is the currently active conversation with the NPC. If not set or
	// set to nil, there is no active conversation between the player and the
	// NPC.
	Convo *Conversation
}

// Copy returns a deeply-copied NPC.
func (npc NPC) Copy() NPC {
	npcCopy := NPC{
		Label:       npc.Label,
		Name:        npc.Name,
		Description: npc.Description,
		Pronouns:    npc.Pronouns,
		Start:       npc.Start,
		Movement:    npc.Movement.Copy(),
		Dialog:      make([]DialogStep, len(npc.Dialog)),
	}

	for i := range npc.Dialog {
		npcCopy.Dialog[i] = npc.Dialog[i].Copy()
	}

	if npc.Convo != nil {
		npcCopy.Convo = &Conversation{
			cur:    npc.Convo.cur,
			Dialog: npcCopy.Dialog,
		}

		if npc.Convo.aliases != nil {
			npcCopy.Convo.aliases = make(map[string]int)
			for k, v := range npc.Convo.aliases {
				npcCopy.Convo.aliases[k] = v
			}
		}
	}

	return npcCopy
}

func (npc NPC) String() string {
	return fmt.Sprintf("NPC<%q>", npc.Name)
}
