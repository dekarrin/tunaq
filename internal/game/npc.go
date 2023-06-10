package game

import (
	"fmt"
	"math/rand"

	"github.com/dekarrin/tunaq/tunascript"
)

// This file contains structs and routines related to NPCs.

// NPC is a NonPlayerCharacter in the world. They may move between rooms
// depending on their Movement property and can be talked to by the player. If
// they can move, they only do so when the player does.
type NPC struct {
	// Label is a name for the NPC and canonical way to index it
	// programmatically. It should be upper case and MUST be unique within all
	// NPC labels of the world.
	Label string

	// Name is the short description of the NPC.
	Name string

	// Aliases is all names that can be used to refer to an NPC.
	Aliases []string

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
	Dialog []*DialogStep

	// Convo is the currently active conversation with the NPC. If not set or
	// set to nil, there is no active conversation between the player and the
	// NPC.
	Convo *Conversation

	// for NPCs with a path movement route, routeCur gives the step it is
	// currently on.
	routeCur *int

	// tmplDescription is the precomputed template AST for the description text.
	// It must generally be filled in with the game engine, and will not be
	// present directly when loaded from disk.
	tmplDescription *tunascript.Template
}

// ResetRoute resets the route of the NPC. It should always be called before
// attempting to call NextRouteStep().
func (npc *NPC) ResetRoute() {
	npc.routeCur = new(int)
	*npc.routeCur = -1
}

// NextRouteStep gets the name of the next location this character would like to
// travel to. If it returns an empty string, the NPC's route dictates that they
// should stay.
//
// room is the current room that they are in.
func (npc NPC) NextRouteStep(room *Room) string {
	if npc.routeCur == nil {
		return ""
	}

	switch npc.Movement.Action {
	case RouteStatic:
		return ""
	case RoutePatrol:
		*npc.routeCur++
		*npc.routeCur %= len(npc.Movement.Path)
		return npc.Movement.Path[*npc.routeCur]
	case RouteWander:
		// first, get the list of allowed rooms. if allowedrooms is not set,
		// all rooms are allowed.
		candidateRooms := map[string]bool{}
		for _, egress := range room.Exits {
			label := egress.DestLabel

			if len(npc.Movement.AllowedRooms) > 0 {
				for _, allowed := range npc.Movement.AllowedRooms {
					if label == allowed {
						candidateRooms[label] = true
						break
					}
				}
			} else {
				candidateRooms[label] = true
			}
		}

		for _, forbidden := range npc.Movement.ForbiddenRooms {
			delete(candidateRooms, forbidden)
		}

		if len(candidateRooms) < 1 {
			// should never happen but check anyways and refuse to move if
			// conditions are as such
			return ""
		}

		candidateRoomsSlice := []string{}
		for k := range candidateRooms {
			candidateRoomsSlice = append(candidateRoomsSlice, k)
		}

		selectionIdx := rand.Intn(len(candidateRoomsSlice))
		choice := candidateRoomsSlice[selectionIdx]
		return choice
	default:
		return ""
	}
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
		Dialog:      make([]*DialogStep, len(npc.Dialog)),
		Aliases:     make([]string, len(npc.Aliases)),

		tmplDescription: npc.tmplDescription,
	}

	for i := range npc.Dialog {
		step := npc.Dialog[i].Copy()
		npcCopy.Dialog[i] = &step
	}
	copy(npcCopy.Aliases, npc.Aliases)

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

func (npc NPC) GetAliases() []string {
	return npc.Aliases
}

func (npc NPC) GetDescription() *tunascript.Template {
	return npc.tmplDescription
}
