package game

import (
	"fmt"
	"strings"

	"github.com/dekarrin/rosed"
	"github.com/dekarrin/tunaq/internal/command"
	"github.com/dekarrin/tunaq/internal/tqerrors"
	"github.com/dekarrin/tunaq/internal/util"
)

var commandHelp = [][2]string{
	{"HELP", "show this help"},
	{"DROP/PUT", "put down an object in the room"},
	{"DEBUG NPC", "print info on all NPCs, or a single NPC with label LABEL if 'DEBUG NPC LABEL' is typed, or steps all NPCs if 'DEBUG NPC @STEP' is typed."},
	{"DEBUG ROOM", "print info on the current room, or teleport to room with label LABEL if 'DEBUG ROOM LABEL' is typed."},
	{"EXITS", "show the names of all exits from the room"},
	{"GO/MOVE", "go to another room via one of the exits"},
	{"INVENTORY/INVEN", "show your current inventory"},
	{"LOOK", "show the description of the room"},
	{"QUIT/BYE", "end the game"},
	{"TAKE/GET", "pick up an object in the room"},
	{"TALK/SPEAK", "talk to someone/something in the room [WIP]"},
	{"USE", "use an object in your inventory [WIP]"},
}

var textFormatOptions = rosed.Options{
	PreserveParagraphs: true,
}

// State is the game's entire state.
type State struct {
	// World is all rooms that exist and their current state.
	World map[string]*Room

	// CurrentRoom is the room that the player is in.
	CurrentRoom *Room

	// Inventory is the objects that the player currently has.
	Inventory Inventory

	// npcLocations is a map of an NPC's label to the label of the room that the
	// NPC is currently in.
	npcLocations map[string]string

	// width is how wide to make output
	io IODevice
}

type IODevice struct {
	// The width of each line of output.
	Width int

	// a function to send output. If s is empty, an empty line is sent.
	Output func(s string, a ...interface{}) error

	// a function to use to get string input. If prompt is blank, no prompt is
	// sent before the input is read.
	Input func(prompt string) (string, error)

	// a function to use to get int input. If prompt is blank, no prompt is
	// sent before the input is read. If invalid input is received, keeps
	// prompting until a valid one is entered.
	InputInt func(prompt string) (int, error)
}

// New creates a new State and loads the list of rooms into it. It performs
// basic sanity checks to ensure that a valid world is being passed in and
// normalizes them as needed.
//
// startingRoom is the label of the room to start with.
// ioDev is the input/output device to use when the user needs to be prompted
// for more info, or for showing to the user.
// io.Width is how wide the output should be. State will try to make all\
// output fit within this width. If not set or < 2, it will be automatically
// assumed to be 80.
func New(world map[string]*Room, startingRoom string, ioDev IODevice) (State, error) {
	if ioDev.Width < 2 {
		ioDev.Width = 80
	}
	if ioDev.Input == nil {
		return State{}, fmt.Errorf("io device must define an Input function")
	}
	if ioDev.InputInt == nil {
		return State{}, fmt.Errorf("io device must define an InputInt function")
	}
	if ioDev.Output == nil {
		return State{}, fmt.Errorf("io device must define an Output function")
	}

	gs := State{
		World:        world,
		Inventory:    make(Inventory),
		npcLocations: make(map[string]string),
		io:           ioDev,
	}

	// now set the current room
	var startExists bool
	gs.CurrentRoom, startExists = gs.World[startingRoom]
	if !startExists {
		return gs, fmt.Errorf("starting room with label %q does not exist in passed-in rooms", startingRoom)
	}

	// read current npc locations and prep them for movement
	for _, r := range gs.World {
		for _, npc := range r.NPCs {
			npc.ResetRoute()
			gs.npcLocations[npc.Label] = r.Label
		}
	}

	return gs, nil
}

// MoveNPCs applies all movements on NPCs that are in the world.
func (gs *State) MoveNPCs() {
	newLocs := map[string]string{}

	for npcLabel, roomLabel := range gs.npcLocations {
		room := gs.World[roomLabel]
		npc := room.NPCs[npcLabel]

		next := npc.NextRouteStep(room)

		if next != "" {
			nextRoom := gs.World[next]
			nextRoom.NPCs[npc.Label] = npc
			delete(room.NPCs, npc.Label)
			newLocs[npc.Label] = nextRoom.Label
		} else {
			newLocs[npc.Label] = room.Label
		}
	}

	gs.npcLocations = newLocs
}

// Look gets the look description as a single long string. It returns non-nil
// error if there are issues retrieving it. If alias is empty, the room is
// looked at. The returned string is not formatted except that any seperate
// listings (such as items or NPCs in a room) will be separated by "\n\n".
func (gs *State) Look(alias string) (string, error) {
	if alias != "" {
		return "", tqerrors.Interpreterf("I can't LOOK at particular things yet")
	}

	output := gs.CurrentRoom.Description
	if len(gs.CurrentRoom.Items) > 0 {
		var itemNames []string

		for _, it := range gs.CurrentRoom.Items {
			itemNames = append(itemNames, it.Name)
		}

		output += "\n\n"
		output += "On the ground, you can see "

		output += util.MakeTextList(itemNames) + "."
	}

	if len(gs.CurrentRoom.NPCs) > 0 {
		var npcNames []string

		for _, npc := range gs.CurrentRoom.NPCs {
			npcNames = append(npcNames, npc.Name)
		}

		// TODO: prop so npcs can be invisible to looks for static npcs that are
		// mostly included in description.
		if len(npcNames) > 0 {
			output += "\n\nOh! "

			output += util.MakeTextList(npcNames)

			if len(npcNames) == 1 {
				output += " is "
			} else {
				output += " are "
			}

			output += "here."
		}
	}

	return output, nil
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
func (gs *State) Advance(cmd command.Command) error {
	var output string
	var err error

	switch cmd.Verb {
	case "QUIT":
		return tqerrors.Interpreterf("I can't QUIT; I'm not being executed by a quitable engine")
	case "GO":
		output, err = gs.ExecuteCommandGo(cmd)
	case "EXITS":
		output, err = gs.ExecuteCommandExits(cmd)
	case "TAKE":
		output, err = gs.ExecuteCommandTake(cmd)
	case "DROP":
		output, err = gs.ExecuteCommandDrop(cmd)
	case "LOOK":
		output, err = gs.ExecuteCommandLook(cmd)
	case "INVENTORY":
		output, err = gs.ExecuteCommandInventory(cmd)
	case "TALK":
		output, err = gs.ExecuteCommandTalk(cmd)
	case "DEBUG":
		output, err = gs.ExecuteCommandDebug(cmd)
	case "HELP":
		output, err = gs.ExecuteCommandHelp(cmd)
	default:
		return tqerrors.Interpreterf("I don't know how to %q", cmd.Verb)
	}

	if err != nil {
		return err
	}

	// IO to give output:
	return gs.io.Output(output + "\n\n")
}

// ExecuteCommandGo executes the GO command with the arguments in the provided
// Command and returns the output.
func (gs *State) ExecuteCommandGo(cmd command.Command) (string, error) {
	egress := gs.CurrentRoom.GetEgressByAlias(cmd.Recipient)
	if egress == nil {
		return "", tqerrors.Interpreterf("%q isn't a place you can go from here", cmd.Recipient)
	}

	gs.CurrentRoom = gs.World[egress.DestLabel]

	gs.MoveNPCs()

	lookText, err := gs.Look("")
	if err != nil {
		return "", err
	}

	output := rosed.Edit(egress.TravelMessage).WithOptions(textFormatOptions).
		Wrap(gs.io.Width).
		Insert(rosed.End, "\n\n").
		CharsFrom(rosed.End).
		Insert(rosed.End, lookText).
		Wrap(gs.io.Width).
		String()

	return output, nil
}

// ExecuteCommandExits executes the EXITS command with the arguments in the
// provided Command and returns the output.
func (gs *State) ExecuteCommandExits(cmd command.Command) (string, error) {
	exitTable := ""

	for _, eg := range gs.CurrentRoom.Exits {
		exitTable += strings.Join(eg.Aliases, "/")
		exitTable += " -> "
		exitTable += eg.Description
		exitTable += "\n"
	}

	return exitTable, nil
}

// ExecuteCommandTake executes the TAKE command with the arguments in the
// provided Command and returns the output.
func (gs *State) ExecuteCommandTake(cmd command.Command) (string, error) {
	item := gs.CurrentRoom.GetItemByAlias(cmd.Recipient)
	if item == nil {
		return "", tqerrors.Interpreterf("I don't see any %q here", cmd.Recipient)
	}

	// first remove the item from the room
	gs.CurrentRoom.RemoveItem(item.Label)

	// then add it to inventory.
	gs.Inventory[item.Label] = *item

	output := fmt.Sprintf("You pick up the %s and add it to your inventory.", item.Name)
	return output, nil
}

// ExecuteCommandDrop executes the DROP command with the arguments in the
// provided Command and returns the output.
func (gs *State) ExecuteCommandDrop(cmd command.Command) (string, error) {
	item := gs.Inventory.GetItemByAlias(cmd.Recipient)
	if item == nil {
		return "", tqerrors.Interpreterf("You don't have a %q", cmd.Recipient)
	}

	// first remove item from inven
	delete(gs.Inventory, item.Label)

	// add to room
	gs.CurrentRoom.Items = append(gs.CurrentRoom.Items, *item)

	output := fmt.Sprintf("You drop the %s onto the ground", item.Name)
	return output, nil
}

// ExecuteCommandDrop executes the LOOK command with the arguments in the
// provided Command and returns the output.
func (gs *State) ExecuteCommandLook(cmd command.Command) (string, error) {
	output, err := gs.Look(cmd.Recipient)
	if err != nil {
		return "", err
	}

	output = rosed.Edit(output).WrapOpts(gs.io.Width, textFormatOptions).String()
	return output, nil
}

// ExecuteCommandInventory executes the INVENTORY command with the arguments in
// the provided Command and returns the output.
func (gs *State) ExecuteCommandInventory(cmd command.Command) (string, error) {
	var output string

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

	output = rosed.Edit(output).WrapOpts(gs.io.Width, textFormatOptions).String()
	return output, nil
}

// ExecuteCommandTalk executes the TALK command with the arguments in
// the provided Command and returns the output. This will enter into an input
// loop that will not exit until the conversation is PAUSED or an END step is
// reached in it.
func (gs *State) ExecuteCommandTalk(cmd command.Command) (string, error) {
	npc := gs.CurrentRoom.GetNPCByAlias(cmd.Recipient)
	if npc == nil {
		return "", tqerrors.Interpreterf("I don't see a %q you can talk to here.", cmd.Recipient)
	}

	if npc.Convo == nil {
		npc.Convo = &Conversation{Dialog: npc.Dialog}
	}

	err := gs.RunConversation(npc)
	if err != nil {
		return "", err
	}

	output := fmt.Sprintf("You stop talking to %s.", strings.ToLower(npc.Pronouns.Objective))
	return output, nil
}

// ExecuteCommandDebug executes the DEBUG command with the arguments in the
// provided Command and returns the output. The DEBUG command has varied
// arguments and may do different things based on what else is provided.
func (gs *State) ExecuteCommandDebug(cmd command.Command) (string, error) {
	if cmd.Recipient == "ROOM" {
		return gs.executeDebugRoom(cmd.Instrument)
	} else if cmd.Recipient == "NPC" {
		return gs.executeDebugNPC(cmd.Instrument)
	} else {
		return "", tqerrors.Interpreterf("I don't know how to debug %q", cmd.Recipient)
	}
}

// ExecuteCommandHelp executes the HELP command with the arguments in the
// provided Command and returns the output.
func (gs *State) ExecuteCommandHelp(cmd command.Command) (string, error) {
	var output string

	ed := rosed.
		Edit("").
		WithOptions(rosed.Options{ParagraphSeparator: "\n"}).
		InsertDefinitionsTable(0, commandHelp, 80)
	output = ed.
		Insert(0, "Here are the commands you can use (WIP commands do not yet work fully):\n").
		String()

	return output, nil
}
