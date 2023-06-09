package game

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/dekarrin/rosed"
	"github.com/dekarrin/tunaq/internal/tqerrors"
	"github.com/dekarrin/tunaq/tunascript"
)

// This file contains functions for handling the debug commands of game.State.

func (gs *State) executeDebugRoom(roomLabel string) (string, error) {
	var output string

	if roomLabel == "" {
		output = gs.CurrentRoom.String() + "\n\n(Type 'DEBUG ROOM label' to teleport to that room)"
	} else {
		if _, ok := gs.World[roomLabel]; !ok {
			return "", tqerrors.Interpreterf("There doesn't seem to be any rooms with label %q in this world", roomLabel)
		}

		gs.CurrentRoom = gs.World[roomLabel]

		output = fmt.Sprintf("Poof! You are now in %q", roomLabel)
	}

	return output, nil
}

func (gs *State) executeDebugNPC(npcLabel string) (string, error) {
	var output string

	if npcLabel == "" {
		output = gs.ListNPCs()
		return output, nil
	} else if strings.HasPrefix(npcLabel, "@") {
		if npcLabel == "@STEP" {
			moved, stayed := gs.StepNPCs()

			pluralNPCs := ""
			if stayed+moved != 1 {
				pluralNPCs = "s"
			}

			output = fmt.Sprintf("Applied movement to %d NPC%s; %d moved, %d stayed", stayed+moved, pluralNPCs, moved, stayed)
		} else {
			return "", tqerrors.Interpreterf("There is no NPC DEBUG action called %q; you can only use the @STEP action with NPCs", npcLabel)
		}
	} else {
		return gs.GetNPCInfo(npcLabel)
	}

	return output, nil
}

func (gs *State) executeDebugExec(s string) (string, error) {
	val, err := gs.scripts.Eval(s)
	return val.String(), err
}

func (gs *State) executeDebugExpand(s string) (string, error) {
	return gs.scripts.Expand(s)
}

func (gs *State) executeDebugFlags() (string, error) {
	output := gs.ListFlags()
	return output, nil
}

func (gs *State) executeDebugOps(s string) (string, error) {
	return tunascript.TranslateOperators(s)
}

// ListFlags returns a text table of the Flags in the game and their current
// values.
func (gs *State) ListFlags() string {
	var output string

	// info on all flags
	data := [][]string{{"Flag", "Value"}}

	// we need to ensure a consistent ordering so need to sort all
	// keys first
	flagLabels := gs.scripts.ListFlags()

	for _, flagLabel := range flagLabels {
		val := gs.scripts.GetFlag(flagLabel)

		infoRow := []string{flagLabel, val}
		data = append(data, infoRow)
	}

	tableOpts := rosed.Options{
		TableHeaders:             true,
		NoTrailingLineSeparators: true,
	}

	output = rosed.Edit("").
		InsertTableOpts(0, data, 80, tableOpts).
		String()

	return output
}

// ListNPCs returns a text table of the NPCs in the game and some general
// information about them.
func (gs *State) ListNPCs() string {
	var output string

	// info on all NPCs and their locations
	data := [][]string{{"NPC", "Movement", "Room"}}

	// we need to ensure a consistent ordering so need to sort all
	// keys first
	orderedNPCLabels := make([]string, len(gs.npcLocations))
	var orderedIdx int
	for npcLabel := range gs.npcLocations {
		orderedNPCLabels[orderedIdx] = npcLabel
		orderedIdx++
	}
	sort.Strings(orderedNPCLabels)

	for _, npcLabel := range orderedNPCLabels {
		roomLabel := gs.npcLocations[npcLabel]
		room := gs.World[roomLabel]
		npc := room.NPCs[npcLabel]

		infoRow := []string{npc.Label, npc.Movement.Action.String(), room.Label}
		data = append(data, infoRow)
	}

	footer := "Type \"DEBUG NPC\" followed by the label of an NPC for more info on that NPC.\n"
	footer += "Type \"DEBUG NPC @STEP\" to move all NPCs forward by one turn."

	tableOpts := rosed.Options{
		TableHeaders: true,
	}

	output = rosed.Edit("\n"+footer).
		InsertTableOpts(0, data, 80, tableOpts).
		String()

	return output
}

// StepNPCs has all NPCs take a movement step, and returns the number of NPCs
// moved to a new location and the number of NPCs who stayed in the same
// location.
func (gs *State) StepNPCs() (moved, stayed int) {

	// check original locations so we can tell how many moved
	originalLocs := make(map[string]string)
	for k, v := range gs.npcLocations {
		originalLocs[k] = v
	}

	gs.MoveNPCs()

	// count how many moved and how many stayed
	for k := range gs.npcLocations {
		if originalLocs[k] != gs.npcLocations[k] {
			moved++
		} else {
			stayed++
		}
	}

	return moved, stayed
}

func (gs *State) GetNPCInfo(label string) (string, error) {
	roomLabel, ok := gs.npcLocations[label]
	if !ok {
		return "", tqerrors.Interpreterf("There doesn't seem to be any NPCs with label %q in this world", label)
	}

	room := gs.World[roomLabel]
	npc := room.NPCs[label]

	npcInfo := [][2]string{
		{"Name", npc.Name},
		{"Pronouns", npc.Pronouns.Short()},
		{"Room", room.Label},
		{"Movement Type", npc.Movement.Action.String()},
		{"Start Room", npc.Start},
	}

	if npc.Movement.Action == RoutePatrol {
		routeInfo := ""
		for i := range npc.Movement.Path {
			if npc.routeCur != nil && (((*npc.routeCur)+1)%len(npc.Movement.Path) == i) {
				routeInfo += "==> "
			} else {
				routeInfo += "--> "
			}
			routeInfo += npc.Movement.Path[i]
			if i+1 < len(npc.Movement.Path) {
				routeInfo += " "
			}
		}
		npcInfo = append(npcInfo, [2]string{"Route", routeInfo})
	} else if npc.Movement.Action == RouteWander {
		allowed := strings.Join(npc.Movement.AllowedRooms, ", ")
		forbidden := strings.Join(npc.Movement.AllowedRooms, ", ")

		if forbidden == "" {
			if allowed == "" {
				forbidden = "(none)"
			} else {
				forbidden = "(any not in Allowed list)"
			}
		}
		if allowed == "" {
			allowed = "(all)"
		}

		npcInfo = append(npcInfo, [2]string{"Allowed Rooms", allowed})
		npcInfo = append(npcInfo, [2]string{"Forbidden Rooms", forbidden})
	}

	diaStr := "(none defined)"
	if len(npc.Dialog) > 0 {
		node := "step"
		if len(npc.Dialog) != 1 {
			node += "s"
		}
		diaStr = fmt.Sprintf("%d %s in dialog tree", len(npc.Dialog), node)
	}
	npcInfo = append(npcInfo, [2]string{"Dialog", diaStr})

	npcInfo = append(npcInfo, [2]string{"Description", npc.Description})

	// build at width + 2 then eliminate the left margin that
	// InsertDefinitionsTable always adds to remove the 2 extra
	// chars
	tableOpts := rosed.Options{ParagraphSeparator: "\n", NoTrailingLineSeparators: true}
	output := rosed.Edit("NPC Info for "+npc.Label+"\n"+
		"\n",
	).
		InsertDefinitionsTableOpts(math.MaxInt, npcInfo, gs.io.Width+2, tableOpts).
		LinesFrom(2).
		Apply(func(idx int, line string) []string {
			line = strings.Replace(line[2:], "  -", "  :", 1)
			return []string{line}
		}).
		String()

	return output, nil
}
