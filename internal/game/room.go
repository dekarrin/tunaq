// Package game implements game state and advancement.
package game

// File room.go includes symbols for holding data on the rooms and exits between
// them.

import (
	"fmt"
	"strings"
)

// Egress is an egress point from a room. It contains both a description and the
// label it points to.
type Egress struct {
	// DestLabel is the label of the room this egress goes to.
	DestLabel string

	// Description is the long description of the egress point.
	Description string

	// TravelMessage is the message shown when the player uses this egress
	// point.
	TravelMessage string

	// Aliases is the list of aliases that the user can give to travel via this
	// egress. Note that the label is not included in this list by default to
	// prevent spoilerific room names.
	Aliases []string
}

func (egress Egress) String() string {
	return fmt.Sprintf("Egress(%q -> %s)", egress.Aliases, egress.DestLabel)
}

// Copy returns a deeply-copied Egress.
func (egress Egress) Copy() Egress {
	eCopy := Egress{
		DestLabel:     egress.DestLabel,
		Description:   egress.Description,
		TravelMessage: egress.TravelMessage,
		Aliases:       make([]string, len(egress.Aliases)),
	}

	copy(eCopy.Aliases, egress.Aliases)

	return eCopy
}

// Room is a scene in the game. It contains a series of exits that lead to other
// rooms and a description. They also contain a list of the interactables at
// game start (or will in the future).
type Room struct {
	// Label is how the room is referred to in the game. It must be unique from
	// all other Rooms.
	Label string

	// Name is used in short descriptions (prior to LOOK).
	Name string

	// Description is what is returned when LOOK is given with no arguments.
	Description string

	// Exits is a list of room labels and ways to describe them, pointing to
	// other rooms in the
	// game.
	Exits []Egress

	// Items is the items on the ground. This can be changed over time.
	Items []Item

	// NPCs is the non-player characters currently in the world.
	NPCs map[string]*NPC
}

// Copy returns a deeply-copied Room.
func (room Room) Copy() Room {
	rCopy := Room{
		Label:       room.Label,
		Name:        room.Name,
		Description: room.Description,
		Exits:       make([]Egress, len(room.Exits)),
		Items:       make([]Item, len(room.Items)),
		NPCs:        make(map[string]*NPC, len(room.NPCs)),
	}

	for i := range room.Exits {
		rCopy.Exits[i] = room.Exits[i].Copy()
	}

	for i := range room.Items {
		rCopy.Items[i] = room.Items[i].Copy()
	}

	for k := range room.NPCs {
		copiedNPC := room.NPCs[k].Copy()
		rCopy.NPCs[k] = &copiedNPC
	}

	return rCopy
}

func (room Room) String() string {
	var exits []string
	for _, eg := range room.Exits {
		exits = append(exits, eg.String())
	}
	exitsStr := strings.Join(exits, ", ")

	return fmt.Sprintf("Room<%s %q EXITS: %s>", room.Label, room.Name, exitsStr)
}

// GetNPCByAlias returns the NPC from the room that is referred to by the given
// alias. If no NPC has that alias, the returned NPC is nil.
func (room Room) GetNPCByAlias(alias string) *NPC {
	foundLabel := ""

	for label, npc := range room.NPCs {
		for _, al := range npc.Aliases {
			if al == alias {
				foundLabel = label
				break
			}
		}
		if foundLabel != "" {
			break
		}
	}

	var foundNPC *NPC
	if foundLabel != "" {
		foundNPC = room.NPCs[foundLabel]
	}
	return foundNPC
}

// GetEgressByAlias returns the egress from the room that is represented by the
// given alias. If no Egress has that alias, the returned egress is nil.
func (room Room) GetEgressByAlias(alias string) *Egress {
	foundIdx := -1

	for egIdx, eg := range room.Exits {
		for _, al := range eg.Aliases {
			if al == alias {
				foundIdx = egIdx
				break
			}
		}
		if foundIdx != -1 {
			break
		}
	}

	var foundEgress *Egress
	if foundIdx != -1 {
		foundEgress = &room.Exits[foundIdx]
	}
	return foundEgress
}

// GetItemByAlias returns the item from the room that is represented by the
// given alias. If no Item has that alias, the returned item is nil.
func (room Room) GetItemByAlias(alias string) *Item {
	foundIdx := -1

	for idx, it := range room.Items {
		for _, al := range it.Aliases {
			if al == alias {
				foundIdx = idx
				break
			}
		}
		if foundIdx != -1 {
			break
		}
	}

	var foundItem *Item
	if foundIdx != -1 {
		foundItem = &room.Items[foundIdx]
	}
	return foundItem
}

// RemoveItem removes the item of the given label from the room. If there is
// already no item with that label in the room, this has no effect.
func (room *Room) RemoveItem(label string) {
	itemIndex := -1

	// TODO: why aren't we indexing items by their label?
	// makes it hard to have hardcoded rooms i suppose.
	for idx, it := range room.Items {
		if it.Label == label {
			itemIndex = idx
			break
		}
	}

	if itemIndex == -1 {
		// no item by that label is here
		return
	}

	// otherwise, rewrite items to not include that.
	room.Items = append(room.Items[:itemIndex], room.Items[itemIndex+1:]...)
}
