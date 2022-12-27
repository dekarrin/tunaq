package game

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/dekarrin/rosed"
	"github.com/dekarrin/tunaq/internal/tqerrors"
	"github.com/dekarrin/tunaq/internal/util"
)

var commandHelp = [][2]string{
	{"HELP", "show this help"},
	{"DROP/PUT", "put down an object in the room"},
	{"DEBUG ROOM", "print info on the current room"},
	{"EXITS", "show the names of all exits from the room"},
	{"GO/MOVE", "go to another room via one of the exits"},
	{"INVENTORY/INVEN", "show your current inventory"},
	{"LOOK", "show the description of the room"},
	{"QUIT/BYE", "end the game"},
	{"TAKE/GET", "pick up an object in the room"},
	{"TALK/SPEAK", "talk to someone/something in the room [WIP]"},
	{"USE", "use an object in your inventory [WIP]"},
}

// State is the game's entire state.
type State struct {
	// World is all rooms that exist and their current state.
	World map[string]*Room

	// CurrentRoom is the room that the player is in.
	CurrentRoom *Room

	// Inventory is the objects that the player currently has.
	Inventory Inventory
}

// New creates a new State and loads the list of rooms into it. It performs
// basic sanity checks to ensure that a valid world is being passed in and
// normalizes them as needed.
//
// startingRoom is the label of the room to start with.
func New(world map[string]*Room, startingRoom string) (State, error) {
	gs := State{
		World:     world,
		Inventory: make(Inventory),
	}

	// now set the current room
	var startExists bool
	gs.CurrentRoom, startExists = gs.World[startingRoom]
	if !startExists {
		return gs, fmt.Errorf("starting room with label %q does not exist in passed-in rooms", startingRoom)
	}

	return gs, nil
}

// MoveNPCs applies all movements on NPCs that are in the world.
func (gs *State) MoveNPCs() {
	alreadyMovedNPCs := map[string]bool{}
	for _, room := range gs.World {
		for _, npc := range room.NPCs {
			if _, hasMoved := alreadyMovedNPCs[npc.Label]; hasMoved {
				continue
			}

			next := npc.NextRouteStep(room)

			if next != "" {
				nextRoom := gs.World[next]
				nextRoom.NPCs[npc.Label] = npc
				delete(room.NPCs, npc.Label)
			}

			alreadyMovedNPCs[npc.Label] = true
		}
	}
}

// Advance advances the game state based on the given command. If there is a
// problem executing the command, it is given in the error output and the game
// state is not advanced. If it is, the result of the command is written to the
// provided output stream.
//
// Invalid commands will be returned as non-nil errors as opposed to writing
// directly to the IO stream; the caller can decide whether to do this themself.
//
// Note that for this, QUIT is not considered a valid command is it would be on
// a controlling engine to end the game state based on that.
//
// TODO: differentiate syntax errors from io errors
func (gs *State) Advance(cmd Command, ostream *bufio.Writer) error {
	var output string

	switch cmd.Verb {
	case "QUIT":
		return tqerrors.Interpreterf("I can't QUIT; I'm not being executed by a quitable engine")
	case "GO":
		egress := gs.CurrentRoom.GetEgressByAlias(cmd.Recipient)
		if egress == nil {
			return tqerrors.Interpreterf("%q isn't a place you can go from here", cmd.Recipient)
		}

		gs.CurrentRoom = gs.World[egress.DestLabel]

		gs.MoveNPCs()

		output = egress.TravelMessage
	case "EXITS":
		exitTable := ""

		for _, eg := range gs.CurrentRoom.Exits {
			exitTable += strings.Join(eg.Aliases, "/")
			exitTable += " -> "
			exitTable += eg.Description
			exitTable += "\n"
		}

		output = exitTable
	case "TAKE":
		item := gs.CurrentRoom.GetItemByAlias(cmd.Recipient)
		if item == nil {
			return tqerrors.Interpreterf("I don't see any %q here", cmd.Recipient)
		}

		// first remove the item from the room
		gs.CurrentRoom.RemoveItem(item.Label)

		// then add it to inventory.
		gs.Inventory[item.Label] = *item

		output = fmt.Sprintf("You pick up the %s and add it to your inventory.", item.Name)
	case "DROP":
		item := gs.Inventory.GetItemByAlias(cmd.Recipient)
		if item == nil {
			return tqerrors.Interpreterf("You don't have a %q", cmd.Recipient)
		}

		// first remove item from inven
		delete(gs.Inventory, item.Label)

		// add to room
		gs.CurrentRoom.Items = append(gs.CurrentRoom.Items, *item)

		output = fmt.Sprintf("You drop the %s onto the ground", item.Name)
	case "LOOK":
		if cmd.Recipient != "" {
			return tqerrors.Interpreterf("I can't LOOK at particular things yet")
		}

		output = gs.CurrentRoom.Description
		if len(gs.CurrentRoom.Items) > 0 {
			var itemNames []string

			for _, it := range gs.CurrentRoom.Items {
				itemNames = append(itemNames, it.Name)
			}

			output += "\n\n"
			output += "On the ground, you can see "

			output += util.MakeTextList(itemNames) + "."
		}
	case "INVENTORY":
		if len(gs.Inventory) < 1 {
			output = "You aren't carrying anything"
		} else {
			var itemNames []string
			for _, it := range gs.Inventory {
				itemNames = append(itemNames, it.Name)
			}

			output = "You currently have the following items:\n"
			output += util.MakeTextList(itemNames) + "."
		}
	case "DEBUG":
		if cmd.Recipient == "ROOM" {
			output = gs.CurrentRoom.String()
		} else {
			return tqerrors.Interpreterf("I don't know how to debug %q", cmd.Recipient)
		}
	case "HELP":
		ed := rosed.
			Edit("").
			WithOptions(rosed.Options{ParagraphSeparator: "\n"}).
			InsertDefinitionsTable(0, commandHelp, 80)
		output = ed.
			Insert(0, "Here are the commands you can use (WIP commands do not yet work fully):\n").
			String()
	default:
		return tqerrors.Interpreterf("I don't know how to %q", cmd.Verb)
	}

	// IO to give output:
	if _, err := ostream.WriteString(output + "\n\n"); err != nil {
		return fmt.Errorf("could not write output: %w", err)
	}
	if err := ostream.Flush(); err != nil {
		return fmt.Errorf("could not flush output: %w", err)
	}

	return nil
}
