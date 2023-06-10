package game

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/dekarrin/rosed"
	"github.com/dekarrin/tunaq/internal/command"
	"github.com/dekarrin/tunaq/internal/tqerrors"
	"github.com/dekarrin/tunaq/internal/util"
	"github.com/dekarrin/tunaq/tunascript"
)

var commandHelp = [][2]string{
	{"HELP", "show this help"},
	{"DROP/PUT", "put down an object in the room"},
	{"DEBUG NPC", "print info on all NPCs, or a single NPC with label LABEL if 'DEBUG NPC LABEL' is typed, or steps all NPCs if 'DEBUG NPC @STEP' is typed."},
	{"DEBUG ROOM", "print info on the current room, or teleport to room with label LABEL if 'DEBUG ROOM LABEL' is typed."},
	{"DEBUG EXEC [code]", "print what the tunascript code evaluates to"},
	{"DEBUG EXPAND [text]", "print the given text with tunascript $IFs and flags expanded"},
	{"DEBUG OPS [text]", "REMINDER TO REMOVE THIS COMMAND WHEN DONE TESTING"},
	{"DEBUG FLAGS", "print all flags and their values"},
	{"EXITS", "show the names of all exits from the room"},
	{"GO/MOVE", "go to another room via one of the exits"},
	{"INVENTORY/INVEN", "show your current inventory"},
	{"LOOK [something]", "show the description of something, or the room with LOOK by itself"},
	{"QUIT/BYE", "end the game"},
	{"TAKE/GET", "pick up an object in the room"},
	{"TALK/SPEAK", "talk to someone/something in the room"},
	{"USE", "use an object in your inventory [WIP]"},
}

var textFormatOptions = rosed.Options{
	PreserveParagraphs: true,
	IndentStr:          "  ",
}

// State is the game's entire state. It should not be used directly; call New to
// create and initialize one.
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

	// itemLocations is a map of items to the label of the room that the item is
	// currently in. will map to "@INVEN" if it is in the inventory.
	itemLocations map[string]string

	// width is how wide to make output
	io IODevice

	scripts tunascript.Interpreter
}

// TODO: this should rly be an interface, not a struct.
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

type worldInterface struct {
	fnInInven func(string) bool
	fnMove    func(string, string) bool
	fnOutput  func(string) bool
}

func (wi worldInterface) InInventory(item string) bool {
	return wi.fnInInven(item)
}

func (wi worldInterface) Move(target, dest string) bool {
	return wi.fnMove(target, dest)
}

func (wi worldInterface) Output(out string) bool {
	return wi.fnOutput(out)
}

// New creates a new State and loads the list of rooms into it. It performs
// basic sanity checks to ensure that a valid world is being passed in and
// normalizes them as needed.
//
// startingRoom is the label of the room to start with.
// ioDev is the input/output device to use when the user needs to be prompted
// for more info, or for showing to the user.
// io.Width is how wide the output should be. State will try to make all
// output fit within this width. If not set or < 2, it will be automatically
// assumed to be 80.
func New(world map[string]*Room, startingRoom string, flags map[string]string, ioDev IODevice) (*State, error) {
	if ioDev.Width < 2 {
		ioDev.Width = 80
	}
	if ioDev.Input == nil {
		return nil, fmt.Errorf("io device must define an Input function")
	}
	if ioDev.InputInt == nil {
		return nil, fmt.Errorf("io device must define an InputInt function")
	}
	if ioDev.Output == nil {
		return nil, fmt.Errorf("io device must define an Output function")
	}

	gs := &State{
		World:         world,
		Inventory:     make(Inventory),
		npcLocations:  make(map[string]string),
		itemLocations: make(map[string]string),
		io:            ioDev,
	}

	scriptInterface := worldInterface{
		fnInInven: func(s string) bool {
			_, ok := gs.Inventory[strings.ToUpper(s)]
			return ok
		},
		fnMove: func(target, dest string) bool {
			target = strings.ToUpper(target)
			dest = strings.ToUpper(dest)

			if _, ok := gs.World[dest]; !ok {
				// TODO: don't fail silently
				return false
			}
			if target == "@PLAYER" {
				if gs.CurrentRoom.Label == dest {
					return false
				}
				gs.CurrentRoom = gs.World[dest]
				return true
			} else {
				// item?
				if roomLabel, ok := gs.itemLocations[target]; ok {
					if roomLabel == "@INVEN" {
						// it DOES move from backpack
						it := gs.Inventory[target]
						gs.World[dest].Items = append(gs.World[dest].Items, it)
						delete(gs.Inventory, it.Label)
						gs.itemLocations[target] = dest
					}
					if roomLabel == dest {
						return false
					}
					// get the item
					var item Item
					for _, it := range gs.World[roomLabel].Items {
						if it.Label == target {
							item = it
							break
						}
					}

					gs.World[roomLabel].RemoveItem(target)
					gs.World[dest].Items = append(gs.World[dest].Items, item)
					gs.itemLocations[target] = dest
					return true
				}

				// npc?
				roomLabel, ok := gs.npcLocations[target]
				if !ok {
					return false
				}
				if roomLabel == dest {
					return false
				}

				npc := gs.World[roomLabel].NPCs[target]
				delete(gs.World[roomLabel].NPCs, npc.Label)
				gs.World[dest].NPCs[npc.Label] = npc
				gs.npcLocations[target] = dest
				return true
			}
		},
		fnOutput: func(s string) bool {
			err := ioDev.Output(s)
			return err == nil
		},
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
		for _, item := range r.Items {
			gs.itemLocations[item.Label] = r.Label
		}

	}

	// start scripting engine
	gs.scripts = tunascript.Interpreter{
		File:   "(text)",
		Target: scriptInterface,
	}

	for fl := range flags {
		err := gs.scripts.AddFlag(fl, flags[fl])
		if err != nil {
			return gs, err
		}
	}

	// parse all expandable templates for later execution
	err := gs.preParseAllTunascriptTemplates()
	if err != nil {
		return gs, err
	}

	return gs, nil
}

func (gs *State) preParseAllTunascriptTemplates() error {
	roomKeys := util.OrderedKeys(gs.World)

	for _, rKey := range roomKeys {
		r := gs.World[rKey]

		// compute room desc
		preComp, err := gs.preParseTemplate(r.Description)
		if err != nil {
			// show bad text separately
			return fmt.Errorf("room %q: description: %w", r.Label, err)
		}
		r.tmplDescription = preComp

		// compute room exit descs and messages
		for i := range r.Exits {
			eg := r.Exits[i]

			egDescComp, err := gs.preParseTemplate(eg.Description)
			if err != nil {
				return fmt.Errorf("room %q: exit %d: description: %w", r.Label, i, err)
			}
			egMsgComp, err := gs.preParseTemplate(eg.TravelMessage)
			if err != nil {
				return fmt.Errorf("room %q: exit %d: message: %w", r.Label, i, err)
			}

			eg.tmplDescription = egDescComp
			eg.tmplTravelMessage = egMsgComp

			r.Exits[i] = eg
		}

		// compute room detail descs
		for i := range r.Details {
			det := r.Details[i]

			detComp, err := gs.preParseTemplate(det.Description)
			if err != nil {
				return fmt.Errorf("room %q: detail %d: description: %w", r.Label, i, err)
			}
			det.tmplDescription = detComp

			r.Details[i] = det
		}

		// compute item descs
		for i := range r.Items {
			it := r.Items[i]

			itemComp, err := gs.preParseTemplate(it.Description)
			if err != nil {
				return fmt.Errorf("item %q: description: %w", it.Label, err)
			}
			it.tmplDescription = itemComp

			r.Items[i] = it
		}

		// compute NPC descs
		npcKeys := util.OrderedKeys(r.NPCs)
		for _, npcLabel := range npcKeys {
			npc := r.NPCs[npcLabel]

			npcComp, err := gs.preParseTemplate(npc.Description)
			if err != nil {
				return fmt.Errorf("npc %q: description: %w", npc.Label, err)
			}
			npc.tmplDescription = npcComp

			// no need to re-assign to map bc npc is a ptr-to so mutations are
			// reflected in map

			// now set each dialog step's precomputed texts
			for i := range npc.Dialog {
				dia := npc.Dialog[i]

				diaResponseComp, err := gs.preParseTemplate(dia.Response)
				if err != nil {
					return fmt.Errorf("npc %q: line %d: response: %w", npc.Label, i, err)
				}
				diaContentComp, err := gs.preParseTemplate(dia.Content)
				if err != nil {
					return fmt.Errorf("npc %q: line %d: content: %w", npc.Label, i, err)
				}
				diaChoiceComps := make([]*tunascript.Template, len(dia.Choices))
				for j := range dia.Choices {
					chComp, err := gs.preParseTemplate(dia.Choices[j][0])
					if err != nil {
						return fmt.Errorf("npc %q: line %d: choice %d: %w", npc.Label, i, j, err)
					}
					diaChoiceComps[j] = chComp
				}

				dia.tmplResponse = diaResponseComp
				dia.tmplContent = diaContentComp
				dia.tmplChoices = diaChoiceComps
			}
		}
	}

	return nil
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

// Expand executes the given template text and turns it into the resulting text.
// Any tunascript queries required to evaluate template flow-control statements
// are executed at this time.
//
// The parameter what is the thing that is being described. It is only used in
// error output. If what is blank and there is an error, a generic string will
// be used.
//
// If there is an error with the expanded text, the returned string will contain
// the error followed by the unexpanded text.
func (gs *State) Expand(s *tunascript.Template, what string) string {
	if what == "" {
		what = "TEXT"
	}

	log.Printf("EXPANDING WITH: %v", gs.scripts)
	expanded := gs.scripts.ExecTemplate(*s)

	return expanded
}

// Look gets the look description as a single long string. It returns non-nil
// error if there are issues retrieving it. If alias is empty, the room is
// looked at. The returned string is not formatted except that any seperate
// listings (such as items or NPCs in a room) will be separated by "\n\n". The
// returned string will be expanded from its tunascript template.
func (gs *State) Look(alias string) (string, error) {
	var desc string
	if alias != "" {
		lookTarget := gs.CurrentRoom.GetTargetable(alias)
		if lookTarget == nil {
			return "", tqerrors.Interpreterf("I don't see any %q here", alias)
		}

		desc = gs.Expand(lookTarget.GetDescription(), fmt.Sprintf("DESCRIPTION FOR %q", alias))
	} else {
		desc = gs.Expand(gs.CurrentRoom.tmplDescription, fmt.Sprintf("DESCRIPTION OF ROOM %q", gs.CurrentRoom.Label))

		if len(gs.CurrentRoom.Items) > 0 {
			var itemNames []string

			for _, it := range gs.CurrentRoom.Items {
				itemNames = append(itemNames, it.Name)
			}

			desc += "\n\n"
			desc += "On the ground, you can see "

			desc += util.MakeTextList(itemNames, true) + "."
		}

		if len(gs.CurrentRoom.NPCs) > 0 {
			var npcNames []string

			for _, npc := range gs.CurrentRoom.NPCs {
				npcNames = append(npcNames, npc.Name)
			}

			// TODO: prop so npcs can be invisible to looks for static npcs that are
			// mostly included in description.
			if len(npcNames) > 0 {
				desc += "\n\nOh! "

				desc += util.MakeTextList(npcNames, false)

				if len(npcNames) == 1 {
					desc += " is "
				} else {
					desc += " are "
				}

				desc += "here."
			}
		}
	}

	return desc, nil
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
	return gs.io.Output("\n" + output + "\n\n")
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

	expanded := gs.Expand(egress.tmplTravelMessage, fmt.Sprintf("Exit travel message for %q", cmd.Recipient))

	output := rosed.Edit(expanded).WithOptions(textFormatOptions).
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
	ed := rosed.Edit("You search for ways out of the room, ").WithOptions(textFormatOptions)
	if len(gs.CurrentRoom.Exits) < 1 {
		ed = ed.Insert(rosed.End, "but you can't seem to find any exits right now")
	} else {

		ed = ed.
			Insert(rosed.End, "and find:\n").
			CharsFrom(rosed.End)

		for _, eg := range gs.CurrentRoom.Exits {
			expanded := gs.Expand(eg.tmplDescription, fmt.Sprintf("DESCRIPTION FOR %q", eg.Aliases[0]))
			ed = ed.Insert(rosed.End, "XX* "+eg.Aliases[0]+": "+expanded+"\n")
		}

		// from prior CharsEnd, this should only apply to the list of exits.
		ed = ed.
			WithOptions(ed.Options.WithParagraphSeparator("\n")).
			Wrap(gs.io.Width).
			ApplyParagraphs(func(_ int, para, _, _ string) []string {
				// set first two chars to spaces
				newPara := rosed.Edit(para).Overtype(0, "  ").String()
				return []string{newPara}
			}).
			Commit().
			Insert(rosed.End, "\n(You might be able to call them other things, too)")
	}

	return ed.String(), nil
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

	gs.itemLocations[item.Label] = "@INVEN"

	output := fmt.Sprintf("You pick up the %s and add it to your inventory", item.Name)
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
	gs.itemLocations[item.Label] = gs.CurrentRoom.Label

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

	ed := rosed.Edit("").WithOptions(textFormatOptions)

	if cmd.Recipient == "" {
		ed = ed.Insert(rosed.End, "You check your surroundings.\n\n")
	} else {
		// is this an NPC? don't use 'the' with them
		tgt := gs.CurrentRoom.GetTargetable(cmd.Recipient)

		theText := "the"
		if IsNPC(tgt) {
			theText = ""
		}

		ed = ed.Insert(rosed.End, fmt.Sprintf("You examine %s %s.\n\n", theText, cmd.Recipient))
	}

	output = ed.
		Insert(rosed.End, output).
		Wrap(gs.io.Width).
		String()

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
		output += util.MakeTextList(itemNames, true) + "."
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
	} else if cmd.Recipient == "EXEC" {
		return gs.executeDebugExec(cmd.Instrument)
	} else if cmd.Recipient == "EXPAND" {
		return gs.executeDebugExpand(cmd.Instrument)
	} else if cmd.Recipient == "FLAGS" {
		return gs.executeDebugFlags()
	} else if cmd.Recipient == "OPS" {
		return gs.executeDebugOps(cmd.Instrument)
	} else {
		return "", tqerrors.Interpreterf("I don't know how to debug %q", cmd.Recipient)
	}
}

// ExecuteCommandHelp executes the HELP command with the arguments in the
// provided Command and returns the output.
func (gs *State) ExecuteCommandHelp(cmd command.Command) (string, error) {
	output := rosed.Edit("").WithOptions(
		textFormatOptions.
			WithParagraphSeparator("\n").
			WithNoTrailingLineSeparators(true)).
		Insert(rosed.End, "Here are the commands you can use (WIP commands do not yet work fully):\n").
		InsertDefinitionsTable(rosed.End, commandHelp, gs.io.Width).String()

	return output, nil
}

func (gs *State) preParseTemplate(toExpand string) (*tunascript.Template, error) {
	preComp, err := gs.scripts.ParseTemplate(toExpand)
	if err != nil {
		var displayText string

		anyNonWSChar := regexp.MustCompile(`\S`)
		// if there's no non-whitespace char in the text, then... well that's
		// bizarre because it *should* work on empty input.
		if anyNonWSChar.MatchString(toExpand) {
			displayText = "TEMPLATE CONTENT:\n"
			displayText += strings.Repeat("=", gs.io.Width) + "\n"
			displayText += rosed.
				Edit(strings.TrimSpace(toExpand)).
				WithOptions(textFormatOptions).
				Wrap(gs.io.Width).
				String()
			displayText += "\n" + strings.Repeat("=", gs.io.Width) + "\n"
		} else {
			displayText = "(NO CONTENT IN TEMPLATE)\n"
		}

		var addendum string
		// TODO: this should be done by an errors.Is check, not this nonsense.
		// might require updating ictiobus though to make syntax errors
		// concerned with EOT special (which probs should be done, glub)
		if strings.Contains(err.Error(), "unexpected end of input") {
			addendum = "\n\nMERMAID'S ADVICE:\nDid you forget to write $[[ENDIF]] somewhere in the template?"
		} else if strings.Contains(err.Error(), "unexpected \"(\"") {
			addendum = "\n\nMERMAID'S ADVICE:\nDid you forget a \"$\" before the name of a function?"
		}

		return nil, fmt.Errorf("template code has an error\n%s\nSYNTAX ERROR:\n%w%s", displayText, err, addendum)
	}

	return &preComp, nil
}
