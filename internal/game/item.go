package game

import (
	"fmt"
	"strings"

	"github.com/dekarrin/tunaq/tunascript"
)

// File item.go holds symbols related to items and inventory

// Inventory is a store of items.
type Inventory map[string]*Item

// GetItemByAlias returns the item from the Inventory that is represented by the
// given alias. If no Item in the inventory has that alias, the returned item is
// nil.
func (inv Inventory) GetItemByAlias(alias string) *Item {
	foundLabel := ""

	for label, it := range inv {
		for _, al := range it.Aliases {
			if al == alias {
				foundLabel = label
				break
			}
		}
		if foundLabel != "" {
			break
		}
	}

	var foundItem *Item
	if foundLabel != "" {
		item := inv[foundLabel]
		foundItem = item
	}
	return foundItem
}

// Item is an object that can be picked up. It contains a unique label, a
// description, and aliases that it can be referred to by. All aliases SHOULD be
// unique in case an item is dropped with another, but as long as at least ONE
// alias is present, we can handle the ambiguous case by asking player to
// restate.
type Item struct {
	// Tags is a list of all tags that will include this Item. Each tag
	// includes the leading @-sign. All items are also implicitly included by
	// the tag @ITEM, regardless of whether it appears in this slice.
	Tags []string

	// Label is a name for the item and canonical way to index it
	// programmatically. It should be upper case and MUST be unique within all
	// labels of the world.
	Label string

	// Name is the short name of the item.
	Name string

	// Description is what is shown when the player LOOKs at the item.
	Description string

	// Aliases are all of the strings that can be used to refer to the item. It
	// must have at least one string that is unique amongst the labels in the
	// world it is in. It does not include Label by default, this must be
	// explicitly given.
	Aliases []string

	// If is the tunascript that is evaluated to determine if this item is
	// interactable and visible to the user. If IfRaw is empty, this will be an
	// expression that always returns true.
	If tunascript.AST

	// IfRaw is the string that contains the TunaScript source code that was
	// parsed into the AST located in If. It will be empty if no code was parsed
	// to do so.
	IfRaw string

	// OnUse is a set of criteria to match when using this item. It includes the
	// other things it must be used with, as well as any other conditions that
	// must be true.
	OnUse []UseAction

	// tmplDescription is the precomputed template AST for the description text.
	// It must generally be filled in with the game engine, and will not be
	// present directly when loaded from disk.
	tmplDescription *tunascript.Template
}

func (item Item) String() string {
	return fmt.Sprintf("Item(%q, (%s))", item.Label, strings.Join(item.Aliases, ", "))
}

// Copy returns a deeply-copied Item.
func (item Item) Copy() Item {
	iCopy := Item{
		Label:       item.Label,
		Name:        item.Name,
		Description: item.Description,
		Aliases:     make([]string, len(item.Aliases)),
		Tags:        make([]string, len(item.Tags)),
		If:          item.If,
		IfRaw:       item.IfRaw,
		OnUse:       make([]UseAction, len(item.OnUse)),

		tmplDescription: item.tmplDescription,
	}

	copy(iCopy.Aliases, item.Aliases)
	copy(iCopy.Tags, item.Tags)
	copy(iCopy.OnUse, item.OnUse)

	return iCopy
}

func (item Item) GetAliases() []string {
	return item.Aliases
}

func (item Item) GetDescription() *tunascript.Template {
	return item.tmplDescription
}

func (item Item) GetLabel() string {
	return item.Label
}

func (item Item) GetTags() []string {
	return item.Tags
}
