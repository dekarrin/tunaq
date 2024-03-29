// Package game implements game state and advancement.
package game

// File room.go includes symbols for holding data on the rooms and exits between
// them.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dekarrin/tunaq/tunascript"
)

// Detail is an additional piece of detail in a room that the user is allowed to
// look at and/or target as objects of commands. If targeted by something that
// it cannot handle, player would not be allowed to do that with it. Every
// detail at minimum can be LOOKed at.
type Detail struct {
	// Tags is a list of all tags that will include this Detail. Each tag
	// includes the leading @-sign. All details are also implicitly included by
	// the tag @DETAIL, regardless of whether it appears in this slice.
	Tags []string

	// Label is a unique identifier for the detail. If it is not specified at
	// start, one is automatically assigned.
	Label string

	// Aliases is the aliases that the player can use to target the detail.
	Aliases []string

	// Description is the long description of the detail, shown when the player
	// LOOKs at it.
	Description string

	// If is the tunascript that is evaluated to determine if this detail is
	// interactable and visible to the user. If IfRaw is empty, this will be an
	// expression that always returns true.
	If tunascript.AST

	// IfRaw is the string that contains the TunaScript source code that was
	// parsed into the AST located in If. It will be empty if no code was parsed
	// to do so.
	IfRaw string

	// tmplDescription is the precomputed template AST for the description text.
	// It must generally be filled in with the game engine, and will not be
	// present directly when loaded from disk.
	tmplDescription *tunascript.Template
}

func (d Detail) GetAliases() []string {
	return d.Aliases
}

func (d Detail) GetDescription() *tunascript.Template {
	return d.tmplDescription
}

func (d Detail) GetLabel() string {
	return d.Label
}

func (d Detail) GetTags() []string {
	return d.Tags
}

func (d Detail) String() string {
	return fmt.Sprintf("Detail<%s>", d.Aliases)
}

// Copy returns a deeply-copied Egress.
func (d Detail) Copy() Detail {
	dCopy := Detail{
		Aliases:         make([]string, len(d.Aliases)),
		Tags:            make([]string, len(d.Tags)),
		Label:           d.Label,
		Description:     d.Description,
		IfRaw:           d.IfRaw,
		If:              d.If,
		tmplDescription: d.tmplDescription,
	}

	copy(dCopy.Aliases, d.Aliases)
	copy(dCopy.Tags, d.Tags)

	return dCopy
}

// Egress is an egress point from a room. It contains both a description and the
// label it points to.
type Egress struct {
	// Label is a unique identifier for the detail. If it is not specified at
	// start, one is automatically assigned.
	Label string

	// Tags is a list of all tags that will include this Egress. Each tag
	// includes the leading @-sign. All egresses are also implicitly included by
	// the tag @EXIT, regardless of whether it appears in this slice.
	Tags []string

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

	// If is the tunascript that is evaluated to determine if this egress is
	// interactable and visible to the user. If IfRaw is empty, this will be an
	// expression that always returns true.
	If tunascript.AST

	// IfRaw is the string that contains the TunaScript source code that was
	// parsed into the AST located in If. It will be empty if no code was parsed
	// to do so.
	IfRaw string

	// tmplDescription is the precomputed template AST for the description text.
	// It must generally be filled in with the game engine, and will not be
	// present directly when loaded from disk.
	tmplDescription *tunascript.Template

	// tmplTravelMessage is the precomputed template AST for the travel message
	// text. It must generally be filled in with the game engine, and will not
	// be present directly when loaded from disk.
	tmplTravelMessage *tunascript.Template
}

func (egress Egress) String() string {
	return fmt.Sprintf("Egress(%q -> %s)", egress.Aliases, egress.DestLabel)
}

// Copy returns a deeply-copied Egress.
func (egress Egress) Copy() Egress {
	eCopy := Egress{
		Label:             egress.Label,
		DestLabel:         egress.DestLabel,
		Description:       egress.Description,
		TravelMessage:     egress.TravelMessage,
		Aliases:           make([]string, len(egress.Aliases)),
		Tags:              make([]string, len(egress.Tags)),
		If:                egress.If,
		IfRaw:             egress.IfRaw,
		tmplDescription:   egress.tmplDescription,
		tmplTravelMessage: egress.tmplTravelMessage,
	}

	copy(eCopy.Aliases, egress.Aliases)
	copy(eCopy.Tags, egress.Tags)

	return eCopy
}

func (egress Egress) GetAliases() []string {
	return egress.Aliases
}

func (egress Egress) GetDescription() *tunascript.Template {
	return egress.tmplDescription
}

func (egress Egress) GetLabel() string {
	return egress.Label
}

func (egress Egress) GetTags() []string {
	return egress.Tags
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
	// other rooms in the game.
	Exits []*Egress

	// Items is the items on the ground. This can be changed over time.
	Items []*Item

	// NPCs is the non-player characters currently in the world.
	NPCs map[string]*NPC

	// Details is the details that the player can look at in the room.
	Details []*Detail

	// tmplDescription is the precomputed template AST for the description text.
	// It must generally be filled in with the game engine, and will not be
	// present directly when loaded from disk.
	tmplDescription *tunascript.Template
}

// Copy returns a deeply-copied Room.
func (room Room) Copy() Room {
	rCopy := Room{
		Label:       room.Label,
		Name:        room.Name,
		Description: room.Description,
		Exits:       make([]*Egress, len(room.Exits)),
		Items:       make([]*Item, len(room.Items)),
		NPCs:        make(map[string]*NPC, len(room.NPCs)),
		Details:     make([]*Detail, len(room.Details)),

		tmplDescription: room.tmplDescription,
	}

	for i := range room.Exits {
		eggCopy := room.Exits[i].Copy()
		rCopy.Exits[i] = &eggCopy
	}

	for i := range room.Items {
		itemCopy := room.Items[i].Copy()
		rCopy.Items[i] = &itemCopy
	}

	for k := range room.NPCs {
		copiedNPC := room.NPCs[k].Copy()
		rCopy.NPCs[k] = &copiedNPC
	}

	for i := range room.Details {
		detCopy := room.Details[i].Copy()
		rCopy.Details[i] = &detCopy
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

// GetTargetable returns the first Targetable game object (Egress, Item, NPC,
// or Detail) from the room that is referred to by the given alias. If no
// Targetable has that alias, the returned Targetable will be nil. If no
// Targetable visible to user has that alias, the returned egress is nil. If a
// Targetable with that alias exists, visibility is determined by calling its If
// function with the given interpreter. To allow returning of a Targetable
// regardless of its visibility, simply pass in nil for the interpreter. User
// label represents the thing trying to get/use/activate a Targetable, and may
// not come into play for all of them.
func (room Room) GetTargetable(alias string, userLabel string, tsEng *tunascript.Interpreter) (t Targetable) {
	if det := room.GetDetailByAlias(alias, userLabel, tsEng); det != nil {
		return det
	}
	if eg := room.GetEgressByAlias(alias, userLabel, tsEng); eg != nil {
		return eg
	}
	if it := room.GetItemByAlias(alias, userLabel, tsEng); it != nil {
		return it
	}
	if npc := room.GetNPCByAlias(alias, userLabel, tsEng); npc != nil {
		return npc
	}

	return nil
}

// GetDetailByAlias returns the Detail from the room that is referred to by the
// given alias. If no detail visible to asker has that alias, the returned
// *Detail is nil. If a detail with that alias exists, visibility is determined
// by whether calling its If with the given interpreter returns true. To allow
// returning of a detail regardless of its visibility, simply pass in "" for the
// asker or nil for the interpreter.
func (room Room) GetDetailByAlias(alias string, asker string, tsEng *tunascript.Interpreter) *Detail {
	foundIdx := -1

	for dIdx, d := range room.Details {
		for _, al := range d.Aliases {
			if al == alias {
				foundIdx = dIdx
				break
			}
		}
		if foundIdx != -1 {
			break
		}
	}

	var foundDetail *Detail
	if foundIdx != -1 {
		foundDetail = room.Details[foundIdx]
	}

	// run the If-check
	if foundDetail != nil && asker != "" && tsEng != nil {
		tsEng.AddFlag(FlagAsker, asker)
		if !tsEng.Exec(foundDetail.If).Bool() {
			foundDetail = nil
		}
		tsEng.RemoveFlag(FlagAsker)
	}

	return foundDetail
}

// GetNPCByAlias returns the NPC from the room that is referred to by the given
// alias. If no NPC visible to asker has that alias, the returned NPC is nil. If
// an NPC with that alias exists, visibility is determined by whether calling
// its If with the given interpreter returns true. To allow returning of an NPC
// regardless of its visibility, simply pass in "" for the asker or nil for the
// interpreter.
func (room Room) GetNPCByAlias(alias string, asker string, tsEng *tunascript.Interpreter) *NPC {
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

	// run the If-check
	if foundNPC != nil && asker != "" && tsEng != nil {
		tsEng.AddFlag(FlagAsker, asker)
		if !tsEng.Exec(foundNPC.If).Bool() {
			foundNPC = nil
		}
		tsEng.RemoveFlag(FlagAsker)
	}

	return foundNPC
}

// GetEgressByAlias returns the active egress from the room that is represented
// by the given alias. If no Egress visible to exiter has that alias, the
// returned egress is nil. If an egress with that alias exists, visibility is
// determined by whether calling its If with the given interpreter returns true.
// To allow returning of an egress regardless of its visibility, simply pass in
// "" for the exiter or nil for the interpreter.
func (room Room) GetEgressByAlias(alias string, exiter string, tsEng *tunascript.Interpreter) *Egress {
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
		foundEgress = room.Exits[foundIdx]
	}

	// run the If-check
	if foundEgress != nil && exiter != "" && tsEng != nil {
		tsEng.AddFlag(FlagAsker, exiter)
		if !tsEng.Exec(foundEgress.If).Bool() {
			foundEgress = nil
		}
		tsEng.RemoveFlag(FlagAsker)
	}

	return foundEgress
}

// GetItemByAlias returns the item from the room that is represented by the
// given alias. If no Item visible to asker has that alias, the returned item
// is nil. If an item with that alias exists, visibility is determined by
// whether calling its If with the given interpreter returns true. To allow
// returning of an Item regardless of its visibility, simply pass in "" for the
// asker or nil for the interpreter.
func (room Room) GetItemByAlias(alias string, asker string, tsEng *tunascript.Interpreter) *Item {
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
		foundItem = room.Items[foundIdx]
	}

	// run the If-check
	if foundItem != nil && asker != "" && tsEng != nil {
		tsEng.AddFlag(FlagAsker, asker)
		if !tsEng.Exec(foundItem.If).Bool() {
			foundItem = nil
		}
		tsEng.RemoveFlag(FlagAsker)
	}
	return foundItem
}

// NPCsAvailable returns all exits that the entity with the given label can
// see, as per the Egress's If value. The returned slice will be ordered by
// label.
func (room Room) NPCsAvailable(asker string, tsEng *tunascript.Interpreter) []*NPC {
	var avail []*NPC

	var allLabels []string

	for k := range room.NPCs {
		allLabels = append(allLabels, k)
	}

	sort.Strings(allLabels)

	if asker == "" || tsEng == nil {
		for _, lbl := range allLabels {
			avail = append(avail, room.NPCs[lbl])
		}
	} else {
		tsEng.AddFlag(FlagAsker, asker)
		for _, lbl := range allLabels {
			npc := room.NPCs[lbl]
			if tsEng.Exec(npc.If).Bool() {
				avail = append(avail, npc)
			}
		}
		tsEng.RemoveFlag(FlagAsker)
	}

	return avail
}

// DetailsAvailable returns all details that the entity with the given label can
// see, as per the Detail's If value.
func (room Room) DetailsAvailable(exiter string, tsEng *tunascript.Interpreter) []*Detail {
	var avail []*Detail

	if exiter == "" || tsEng == nil {
		return room.Details
	}

	tsEng.AddFlag(FlagAsker, exiter)
	for i := range room.Details {
		if tsEng.Exec(room.Details[i].If).Bool() {
			avail = append(avail, room.Details[i])
		}
	}
	tsEng.RemoveFlag(FlagAsker)

	return avail
}

// ExitsAvailable returns all exits that the entity with the given label can
// see, as per the Egress's If value.
func (room Room) ExitsAvailable(exiter string, tsEng *tunascript.Interpreter) []*Egress {
	var avail []*Egress

	if exiter == "" || tsEng == nil {
		return room.Exits
	}

	tsEng.AddFlag(FlagAsker, exiter)
	for i := range room.Exits {
		if tsEng.Exec(room.Exits[i].If).Bool() {
			avail = append(avail, room.Exits[i])
		}
	}
	tsEng.RemoveFlag(FlagAsker)

	return avail
}

// ItemsAvailable returns all items that the entity with the given label can
// see, as per the Item's If value.
func (room Room) ItemsAvailable(asker string, tsEng *tunascript.Interpreter) []*Item {
	var avail []*Item

	if asker == "" || tsEng == nil {
		return room.Items
	}

	tsEng.AddFlag(FlagAsker, asker)
	for i := range room.Items {
		if tsEng.Exec(room.Items[i].If).Bool() {
			avail = append(avail, room.Items[i])
		}
	}
	tsEng.RemoveFlag(FlagAsker)

	return avail
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
