// Package game implements game state and advancement.
package game

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/dekarrin/tunaq/internal/tqerrors"
)

// CommandReader is a type that can be used for getting command input.
// TODO: abstract this into own package (cmd, perhaps `cmd.Reader`) and make it
// return parsed commands, not 'lines'.
type CommandReader interface {
	// ReadCommand reads a single user command. It will block until one is
	// ready. If there is an error or output is at end (EOF), the returned
	// string will be empty, otherwise it will always be non-empty.
	//
	// When error is io.EOF, string will always be empty. If EOF was encountered
	// on a call but some input was received, the input will be returned and
	// error will be nil, and the next call to ReadCommand will return "",
	// io.EOF.
	ReadCommand() (string, error)

	// Close performs any operations required to clean the resources created by
	// the CommandReader. It should be called at least once when the
	// CommandReader is no longer needed.
	Close() error
}

// Inventory is a store of items.
type Inventory map[string]Item

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
		foundItem = &item
	}
	return foundItem
}

// Item is an object that can be picked up. It contains a unique label, a
// description, and aliases that it can be referred to by. All aliases SHOULD be
// unique in case an item is dropped with another, but as long as at least ONE
// alias is present, we can handle the ambiguous case by asking player to
// restate.
type Item struct {

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
	}

	copy(iCopy.Aliases, item.Aliases)

	return iCopy
}

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

// GetCommand is the fundamental unit of obtaining input from the user in an
// interactive fashion. It prompts the user for an input and attempts to parse
// it as a valid command, returning that command if it is successful. If it is
// not, error output is printed to the ostream and the user is prompted until
// they enter a valid command.
//
// Note that this function does not check if the command is executable, only
// that a Command can be parsed from the user input.
//
// TODO: abstract this and the entire command parsing structure to new package,
// cmd.
func GetCommand(cmdStream CommandReader, ostream *bufio.Writer) (Command, error) {
	var cmd Command
	gotValidCommand := false

	if _, err := ostream.WriteString("Enter command\n"); err != nil {
		return cmd, fmt.Errorf("could not write output: %w", err)
	}
	if err := ostream.Flush(); err != nil {
		return cmd, fmt.Errorf("could not flush output: %w", err)
	}

	for !gotValidCommand {
		// IO to get input:
		input, err := cmdStream.ReadCommand()
		if err != nil {
			return cmd, fmt.Errorf("could not get input: %w", err)
		}

		// now attempt to parse the input
		cmd, err = ParseCommand(input)
		if err != nil {
			consoleMessage := tqerrors.GameMessage(err)
			errMsg := fmt.Sprintf("%v\nTry HELP for valid commands\n", consoleMessage)
			// IO to report error and prompt user to try again
			if _, err := ostream.WriteString(errMsg); err != nil {
				return cmd, fmt.Errorf("could not write output: %w", err)
			}
			if err := ostream.Flush(); err != nil {
				return cmd, fmt.Errorf("could not flush output: %w", err)
			}
		} else if cmd.Verb != "" {
			gotValidCommand = true
		}
	}

	return cmd, nil
}
