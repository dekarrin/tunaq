package game

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dekarrin/rosed"
	"github.com/dekarrin/tunaq/internal/command"
	"github.com/dekarrin/tunaq/internal/tqerrors"
	"github.com/dekarrin/tunaq/internal/util"
	"github.com/dekarrin/tunaq/tunascript"
)

const (
	TagPlayer = "@PLAYER"
	TagSelf   = "@SELF"
	FlagAsker = "TQ_ASKER"
)

var commandHelp = [][2]string{
	{"HELP", "show this help"},
	{"DROP/PUT", "put down an object in the room"},
	{"DEBUG NPC", "print info on all NPCs, or a single NPC with label LABEL if 'DEBUG NPC LABEL' is typed, or steps all NPCs if 'DEBUG NPC @STEP' is typed."},
	{"DEBUG ROOM", "print info on the current room, or teleport to room with label LABEL if 'DEBUG ROOM LABEL' is typed."},
	{"DEBUG EXEC [code]", "print what the tunascript code evaluates to"},
	{"DEBUG EXPAND [text]", "print the given text with tunascript $IFs and flags expanded"},
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

	// TagSets is a set of tags and the Targetables that they refer to.
	TagSets map[string][]Targetable

	// npcLocations is a map of an NPC's label to the label of the room that the
	// NPC is currently in.
	npcLocations map[string]string

	// itemLocations is a map of items to the label of the room that the item is
	// currently in. will map to "@INVEN" if it is in the inventory.
	itemLocations map[string]string

	// exitLocations is a map of exit labels to the label of the room that they
	// are in. This will never change but is maintained for tracking purposes.
	exitLocations map[string]string

	// detailLocations is a map of detail labels to the label of the room that
	// they are in. This will never change but is maintained for tracking
	// purposes.
	detailLocations map[string]string

	// tsBufferOutput will send tunascript to tsBuf instead of to the io device
	// if set to true. methods of *State can call this before executing
	// tunascript to control exactly when it is output.
	tsBufferOutput bool

	// output from TunaScript $OUTPUT() functions is sent here instead of to the
	// io device if tsBufferOutput is set to true.
	tsBuf *strings.Builder

	// width is how wide to make output
	io IODevice

	scripts tunascript.Interpreter
}

type IODevice interface {
	// The width of each line of output. Queryable property only; this has no
	// effect on the internals of the other methods.
	Width() int

	// SetWidth sets the width of each line of output. This only changes what
	// Width() returns, and does not have any effect on the internals of the
	// other methods.
	SetWidth(w int)

	// a function to send output. If s is empty, an empty line is sent.
	Output(s string, a ...interface{}) error

	// a function to use to get string input. If prompt is blank, no prompt is
	// sent before the input is read.
	Input(prompt string) (string, error)

	// a function to use to get int input. If prompt is blank, no prompt is
	// sent before the input is read. If invalid input is received, keeps
	// prompting until a valid one is entered.
	InputInt(prompt string) (int, error)
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
	if ioDev == nil {
		return nil, fmt.Errorf("io device must not be nil")
	}
	if ioDev.Width() < 2 {
		ioDev.SetWidth(80)
	}

	gs := &State{
		World:           world,
		Inventory:       make(Inventory),
		TagSets:         make(map[string][]Targetable),
		npcLocations:    make(map[string]string),
		itemLocations:   make(map[string]string),
		exitLocations:   make(map[string]string),
		detailLocations: make(map[string]string),
		tsBuf:           &strings.Builder{},
		io:              ioDev,
	}

	// first, go through and track all taggables
	var taggedNPCs, taggedExits, taggedDetails, taggedItems []Targetable

	for roomLabel := range gs.World {
		r := gs.World[roomLabel]
		for i := range r.Items {
			item := r.Items[i]
			taggedItems = append(taggedItems, item)
			for _, tag := range item.Tags {
				tagged := gs.TagSets[tag]
				tagged = append(tagged, item)
				gs.TagSets[tag] = tagged
			}
			item.Tags = append(item.Tags, "@ITEM")
			r.Items[i] = item
		}
		for i := range r.NPCs {
			npc := r.NPCs[i]
			taggedNPCs = append(taggedNPCs, npc)
			for _, tag := range npc.Tags {
				tagged := gs.TagSets[tag]
				tagged = append(tagged, npc)
				gs.TagSets[tag] = tagged
			}
			npc.Tags = append(npc.Tags, "@NPC")
			r.NPCs[i] = npc
		}
		for i := range r.Details {
			det := r.Details[i]
			taggedDetails = append(taggedDetails, det)
			for _, tag := range det.Tags {
				tagged := gs.TagSets[tag]
				tagged = append(tagged, det)
				gs.TagSets[tag] = tagged
			}
			det.Tags = append(det.Tags, "@DETAIL")
			r.Details[i] = det
		}
		for i := range r.Exits {
			egress := r.Exits[i]
			taggedExits = append(taggedExits, egress)
			for _, tag := range egress.Tags {
				tagged := gs.TagSets[tag]
				tagged = append(tagged, egress)
				gs.TagSets[tag] = tagged
			}
			egress.Tags = append(egress.Tags, "@EXIT")
			r.Exits[i] = egress
		}
		gs.World[roomLabel] = r
	}
	gs.TagSets["@NPC"] = taggedNPCs
	gs.TagSets["@EXIT"] = taggedExits
	gs.TagSets["@ITEM"] = taggedItems
	gs.TagSets["@DETAIL"] = taggedDetails

	// now set the current room
	var startExists bool
	gs.CurrentRoom, startExists = gs.World[startingRoom]
	if !startExists {
		return gs, fmt.Errorf("starting room with label %q does not exist in passed-in rooms", startingRoom)
	}

	// read current targetable entity locations. for NPCs, prep them for movement
	for _, r := range gs.World {
		for _, npc := range r.NPCs {
			npc.ResetRoute()
			gs.npcLocations[npc.Label] = r.Label
		}
		for _, item := range r.Items {
			gs.itemLocations[item.Label] = r.Label
		}
		for _, det := range r.Details {
			gs.detailLocations[det.Label] = r.Label
		}
		for _, eg := range r.Exits {
			gs.exitLocations[eg.Label] = r.Label
		}
	}

	// start scripting engine
	gs.scripts = tunascript.Interpreter{
		File:   "(text)",
		Target: scriptBackend{game: gs},
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

// MoveNPCs applies all movements on NPCs that are in the world whose If's
// currently evaluate to true.
func (gs *State) MoveNPCs() {
	newLocs := map[string]string{}

	for npcLabel, roomLabel := range gs.npcLocations {
		room := gs.World[roomLabel]
		npc := room.NPCs[npcLabel]

		gs.scripts.AddFlag(FlagAsker, "@SELF")
		isActive := gs.scripts.Exec(npc.If).Bool()
		gs.scripts.RemoveFlag(FlagAsker)

		if !isActive {
			continue
		}

		next := npc.NextRouteStep(room, &gs.scripts)

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
func (gs *State) Expand(s *tunascript.Template) string {
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
		lookTarget := gs.CurrentRoom.GetTargetable(alias, TagPlayer, &gs.scripts)
		if lookTarget == nil {
			return "", tqerrors.Interpreterf("I don't see any %q here", alias)
		}

		desc = gs.Expand(lookTarget.GetDescription())
	} else {
		desc = gs.Expand(gs.CurrentRoom.tmplDescription)

		availItems := gs.CurrentRoom.ItemsAvailable(TagPlayer, &gs.scripts)
		if len(availItems) > 0 {
			var itemNames []string

			for _, it := range availItems {
				itemNames = append(itemNames, it.Name)
			}

			desc += "\n\n"
			desc += "On the ground, you can see "

			desc += util.MakeTextList(itemNames, true) + "."
		}

		availNPCs := gs.CurrentRoom.NPCsAvailable(TagPlayer, &gs.scripts)
		if len(availNPCs) > 0 {
			var npcNames []string

			for _, npc := range availNPCs {
				npcNames = append(npcNames, npc.Name)
			}

			// TODO: prop so npcs can be invisible to looks for static npcs that are
			// mostly included in description.
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
	case "USE":
		output, err = gs.ExecuteCommandUse(cmd)
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

// ExecuteCommandUse executes the USE command with the arguments in the provided
// Command and returns the output.
func (gs *State) ExecuteCommandUse(cmd command.Command) (string, error) {
	// allToBeUsed
	var useAliases []string
	useAliases = append(useAliases, cmd.Recipient)

	// TODO: in future when player command parser supports with multiple, bring
	// in multiple stuff here. For now, just add the one and only other thing if
	// it is present
	if cmd.Instrument != "" {
		useAliases = append(useAliases, cmd.Instrument)
	}

	// verify that each item requested to use is even a thing in this room (or
	// in backpack)
	useTargets := make([]Targetable, len(useAliases))
	for i := range useAliases {
		var tgt Targetable
		tgtItem := gs.Inventory.GetItemByAlias(useAliases[i])
		if tgtItem == nil {
			// if not in invent, it could be in the room
			tgt = gs.CurrentRoom.GetTargetable(useAliases[i], TagPlayer, &gs.scripts)
			if tgt == nil {
				return "", tqerrors.Interpreterf("I don't see any %q here or in your inventory", useAliases[i])
			}
		} else {
			tgt = tgtItem
		}

		useTargets[i] = tgt
	}

	// all of the things requested to be used together exist in this room (or
	// inven). Now we must do tricky matching to see if each has a use action
	// that applies to the other(s)
	useMatches := gs.findAllUseMatches(useTargets, useAliases)

	// if we didn't find any matches, give an error to the user
	if len(useMatches) < 1 {
		if len(useTargets) > 1 {
			return "", tqerrors.Interpreterf("You're pretty sure those don't work together")
		}
		return "", tqerrors.Interpreterf("You aren't sure how to use the %s by itself", useAliases[0])
	}

	um := selectBestUseMatch(useMatches)
	// okay, we now have, FINALLY, a single UseAction that we can call

	// first, evaluate the If. We don't exec if it's false
	if !gs.scripts.Exec(um.act.If).Bool() {
		// give the same generic error as if there is no way to use them
		if len(useMatches) < 1 {
			if len(useTargets) > 1 {
				others := util.MakeTextList(useAliases[1:], false)
				return "", tqerrors.Interpreterf("You try to use the %s together with the %s... but nothing happens", useAliases[0], others)
			}
		}
		return "", tqerrors.Interpreterf("You try to use the %s... but nothing happens", useAliases[0])
	}

	ed := rosed.Edit("").WithOptions(textFormatOptions)

	var withOthersMsg string
	if len(useTargets) > 1 {
		others := util.MakeTextList(useAliases[1:], false)
		withOthersMsg += " together with the " + others
	}
	ed = ed.Insert(rosed.End, fmt.Sprintf("You use the %s%s...\n\n", useAliases[0], withOthersMsg))

	// enable buffering so any output doesn't just go directly to gs before we
	// get a chance to write any other output
	gs.tsBufferOutput = true
	defer func() {
		gs.tsBuf.Reset()
		gs.tsBufferOutput = false
	}()

	// execute the use script
	gs.scripts.Exec(um.act.Do)

	// was an $OUTPUT() func executed?
	tsOutput := gs.tsBuf.String()

	if tsOutput != "" {
		// add the $OUTPUT() text to the result
		ed = ed.Insert(rosed.End, tsOutput)
	} else {
		// otherwise, we need to show the user that something has occured
		ed = ed.Insert(rosed.End, "Something happened!")
	}

	output := ed.
		Wrap(gs.io.Width()).
		String()

	return output, nil
}

// ExecuteCommandGo executes the GO command with the arguments in the provided
// Command and returns the output.
func (gs *State) ExecuteCommandGo(cmd command.Command) (string, error) {
	egress := gs.CurrentRoom.GetEgressByAlias(cmd.Recipient, TagPlayer, &gs.scripts)
	if egress == nil {
		return "", tqerrors.Interpreterf("%q isn't a place you can go from here", cmd.Recipient)
	}

	gs.CurrentRoom = gs.World[egress.DestLabel]

	gs.MoveNPCs()

	lookText, err := gs.Look("")
	if err != nil {
		return "", err
	}

	expanded := gs.Expand(egress.tmplTravelMessage)

	output := rosed.Edit(expanded).WithOptions(textFormatOptions).
		Wrap(gs.io.Width()).
		Insert(rosed.End, "\n\n").
		CharsFrom(rosed.End).
		Insert(rosed.End, lookText).
		Wrap(gs.io.Width()).
		String()

	return output, nil
}

// ExecuteCommandExits executes the EXITS command with the arguments in the
// provided Command and returns the output.
func (gs *State) ExecuteCommandExits(cmd command.Command) (string, error) {
	ed := rosed.Edit("You search for ways out of the room, ").WithOptions(textFormatOptions)

	foundExits := gs.CurrentRoom.ExitsAvailable(TagPlayer, &gs.scripts)
	if len(foundExits) < 1 {
		ed = ed.Insert(rosed.End, "but you can't seem to find any exits right now")
	} else {

		ed = ed.
			Insert(rosed.End, "and find:\n").
			CharsFrom(rosed.End)

		for _, eg := range foundExits {
			expanded := gs.Expand(eg.tmplDescription)
			ed = ed.Insert(rosed.End, "XX* "+eg.Aliases[0]+": "+expanded+"\n")
		}

		// from prior CharsEnd, this should only apply to the list of exits.
		ed = ed.
			WithOptions(ed.Options.WithParagraphSeparator("\n")).
			Wrap(gs.io.Width()).
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
	item := gs.CurrentRoom.GetItemByAlias(cmd.Recipient, TagPlayer, &gs.scripts)
	if item == nil {
		return "", tqerrors.Interpreterf("I don't see any %q here", cmd.Recipient)
	}

	// first remove the item from the room
	gs.CurrentRoom.RemoveItem(item.Label)

	// then add it to inventory.
	gs.Inventory[item.Label] = item

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
	gs.CurrentRoom.Items = append(gs.CurrentRoom.Items, item)
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
		tgt := gs.CurrentRoom.GetTargetable(cmd.Recipient, TagPlayer, &gs.scripts)

		theText := "the"
		if IsNPC(tgt) {
			theText = ""
		}

		ed = ed.Insert(rosed.End, fmt.Sprintf("You examine %s %s.\n\n", theText, cmd.Recipient))
	}

	output = ed.
		Insert(rosed.End, output).
		Wrap(gs.io.Width()).
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

	output = rosed.Edit(output).WrapOpts(gs.io.Width(), textFormatOptions).String()
	return output, nil
}

// ExecuteCommandTalk executes the TALK command with the arguments in
// the provided Command and returns the output. This will enter into an input
// loop that will not exit until the conversation is PAUSED or an END step is
// reached in it.
func (gs *State) ExecuteCommandTalk(cmd command.Command) (string, error) {
	npc := gs.CurrentRoom.GetNPCByAlias(cmd.Recipient, TagPlayer, &gs.scripts)
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
		InsertDefinitionsTable(rosed.End, commandHelp, gs.io.Width()).String()

	return output, nil
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

func (gs *State) preParseTemplate(toExpand string) (*tunascript.Template, error) {
	preComp, err := gs.scripts.ParseTemplate(toExpand)
	if err != nil {
		var displayText string

		anyNonWSChar := regexp.MustCompile(`\S`)
		// if there's no non-whitespace char in the text, then... well that's
		// bizarre because it *should* work on empty input.
		if anyNonWSChar.MatchString(toExpand) {
			displayText = "TEMPLATE CONTENT:\n"
			displayText += strings.Repeat("=", gs.io.Width()) + "\n"
			displayText += rosed.
				Edit(strings.TrimSpace(toExpand)).
				WithOptions(textFormatOptions).
				Wrap(gs.io.Width()).
				String()
			displayText += "\n" + strings.Repeat("=", gs.io.Width()) + "\n"
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

func (gs *State) HasTag(label, tag string) bool {
	tagged := gs.TagSets[tag]

	for _, tgt := range tagged {
		if IsDetail(tgt) {
			det := tgt.(*Detail)
			if det.Label == label {
				return true
			}
		} else if IsEgress(tgt) {
			eg := tgt.(*Egress)
			if eg.Label == label {
				return true
			}
		} else if IsItem(tgt) {
			item := tgt.(*Item)
			if item.Label == label {
				return true
			}
		} else if IsNPC(tgt) {
			npc := tgt.(*NPC)
			if npc.Label == label {
				return true
			}
		}
	}
	return false
}

type useMatch struct {
	main     bool
	specific int
	act      UseAction
}

func (gs *State) findAllUseMatches(targets []Targetable, targetAliases []string) []useMatch {
	// gather exact labels because we need them for matching
	targetLabels := make([]string, len(targets))
	for i := range targets {
		targetLabels[i] = targets[i].GetLabel()
	}

	matches := []useMatch{}

	otherLabels := make([]string, len(targets)-1)
	for i := range targets {
		mainTarget := targets[i]
		copy(otherLabels, targetLabels[0:i])
		copy(otherLabels[i:], targetLabels[i+1:])

		if IsItem(mainTarget) {
			item := mainTarget.(*Item)

			for j := range item.OnUse {
				withLatags := item.OnUse[j].With
				if gs.concreteMatchesLatagList(otherLabels, withLatags) {
					specific := len(otherLabels)
					for _, latag := range withLatags {
						if strings.HasPrefix(latag, "@") {
							specific--
						}
					}

					otherAliases := make([]string, len(targets)-1)
					copy(otherAliases, targetAliases[0:i])
					copy(otherAliases[i:], targetAliases[i+1:])

					m := useMatch{i == 0, specific, item.OnUse[j]}
					matches = append(matches, m)
				}
			}
		}

		// otherwise, nothing else currently has actions defined on them, so
		// move on to the next permutation
	}

	return matches
}

// latags is any list of strings where each element could be a label or a
// tag. Concrete must not contain tags or it will always be false. The lengths
// of the two lists must match or it will always be false. Concrete must contain
// only labels.
func (gs *State) concreteMatchesLatagList(concrete []string, latags []string) bool {
	// CONSIDER: otherWiths = ["DACHSUND", "ROTTWEILER"]
	// ACTUAL WITHS: ["@DOGS"], ["DACHSHUND"]
	// Check DACHSUND =? "@DOGS": TRUE, candidate
	//	Check ROTWEILDER =? DACHSUND: FALSE, break
	// CHECK ROTTWIELER =? "@DOGS": TRUE, candidate
	//  Check DACHSUND =? DACHSUND: TRUE, done

	if len(concrete) != len(latags) {
		return false
	}

	if len(concrete) == 0 {
		return true
	}

	for i, latag := range latags {
		conc := concrete[i]

		if strings.HasPrefix(latag, "@") {
			// is the concrete the label of somefin with that tag?
			if !gs.HasTag(conc, latag) {
				continue
			}
		} else if conc != latag {
			continue
		}

		restConc := make([]string, len(concrete)-1)
		copy(restConc, concrete[0:i])
		copy(restConc[i:], concrete[i+1:])

		restLatag := make([]string, len(latags)-1)
		copy(restLatag, latags[0:i])
		copy(restLatag[i:], latags[i+1:])

		if gs.concreteMatchesLatagList(restConc, restLatag) {
			return true
		}
	}

	return false
}

func (g State) MarshalBinary() (data []byte, err error) {
	return []byte{0x00, 0x01}, nil
}

func (g *State) UnmarshalBinary(data []byte) error {
	*g = State{}
	return nil
}
