package tqw

import (
	"fmt"
	"strings"

	"github.com/dekarrin/tunaq/internal/game"
)

func parseManifest(tqw topLevelManifest) (Manifest, error) {
	manif := Manifest{
		Files: tqw.Files,
	}

	return manif, nil
}

func parseWorldData(tqw topLevelWorldData) (WorldData, error) {
	var err error

	var world WorldData
	world.Start = tqw.World.Start
	world.Rooms = make(map[string]*game.Room)

	for idx, r := range tqw.Rooms {
		if roomErr := validateRoomDef(r); roomErr != nil {
			return world, fmt.Errorf("parsing: rooms[%d (%q)]: %w", idx, r.Label, roomErr)
		}

		if _, ok := world.Rooms[r.Label]; ok {
			return world, fmt.Errorf("parsing: rooms[%d (%q)]: room label %q has already been used", idx, r.Label, r.Label)
		}

		room := r.toGameRoom()
		world.Rooms[r.Label] = &room
	}

	pronouns := map[string]pronounSet{
		"SHE/HER":   pronounSetFromGame(game.PronounsFeminine),
		"HE/HIM":    pronounSetFromGame(game.PronounsMasculine),
		"THEY/THEM": pronounSetFromGame(game.PronounsNonBinary),
		"IT/ITS":    pronounSetFromGame(game.PronounsItIts),
	}

	// check loaded pronouns
	for idx, ps := range tqw.Pronouns {
		if err := validatePronounSetDef(ps, nil); err != nil {
			return world, fmt.Errorf("parsing: pronouns[%d (%s)]: %w", idx, ps.Label, err)
		}

		if _, ok := pronouns[ps.Label]; ok {
			return world, fmt.Errorf("parsing: pronouns[%d (%s)]: duplicate pronoun name %q", idx, ps.Label, ps.Label)
		}

		pronouns[ps.Label] = ps
	}

	npcs := make([]game.NPC, 0)
	// parse individual npcs
	for idx, npc := range tqw.NPCs {
		if npc.Movement.Action == "" {
			npc.Movement.Action = "STATIC"
		}

		// set any blank dialog types to line
		for idx, ds := range npc.Dialog {
			if ds.Action == "" {
				ds.Action = "LINE"
				npc.Dialog[idx] = ds
			}
		}

		if err := validateNPCDef(npc, pronouns); err != nil {
			return world, fmt.Errorf("parsing: npcs[%d (%q)]: %w", idx, npc.Label, err)
		}

		// set pronouns to actual
		if npc.Pronouns != "" {
			empty := pronounSet{}
			if npc.PronounSet != empty {
				return world, fmt.Errorf("parsing: npcs[%d (%q)]: can't define custom pronoun set because pronouns is set to %q", idx, npc.Label, npc.Pronouns)
			}
			npc.PronounSet = pronouns[strings.ToUpper(npc.Pronouns)]
		}

		npcs = append(npcs, npc.toGameNPC())
	}

	// now that they are all loaded and individually checked for validity,
	// ensure that all room egresses are valid existing labels
	for roomIdx, r := range tqw.Rooms {
		for egressIdx, eg := range r.Exits {
			if _, ok := world.Rooms[eg.Dest]; !ok {
				errMsg := "validating: rooms[%d (%q)]: exits[%d]: no room with label %q exists"
				return world, fmt.Errorf(errMsg, roomIdx, r.Label, egressIdx, eg.Dest)
			}
		}
	}

	// check that the start actually points to a real location
	if _, ok := world.Rooms[tqw.World.Start]; !ok {
		return world, fmt.Errorf("validating: world: start: no room with label %q exists", tqw.World.Start)
	}

	// ensure that all npc routes are valid, that convo trees make sense, then place NPCs in their start rooms
	// also ensure that all labels are unique among NPCs.
	seenNPCLabels := map[string]int{}
	pf := game.Pathfinder{World: world.Rooms}
	for idx, npc := range npcs {
		// check labels
		if seenInIndex, ok := seenNPCLabels[npc.Label]; ok {
			return world, fmt.Errorf("validating: npcs[%d (%q)]: duplicate label %q; already used by npc %d", idx, npc.Label, npc.Label, seenInIndex)
		}
		seenNPCLabels[npc.Label] = idx

		// check start label
		_, ok := world.Rooms[npc.Start]
		if !ok {
			return world, fmt.Errorf("validating: npcs[%d (%q)]: start: no room with label %q exists", idx, npc.Label, npc.Start)
		}

		// then check route based on movement type
		switch npc.Movement.Action {
		case game.RoutePatrol:
			// can npc get to initial position?
			err = pf.ValidatePath(append([]string{npc.Start}, npc.Movement.Path[0]), false)
			if err != nil {
				return world, fmt.Errorf("validating: npcs[%d (%q)]: %w", idx, npc.Label, err)
			}

			// once at initial, can npc loop through patrol?
			err = pf.ValidatePath(npc.Movement.Path, true)
			if err != nil {
				return world, fmt.Errorf("validating: npcs[%d (%q)]: %w", idx, npc.Label, err)
			}
		case game.RouteWander:
			for roomIdx, roomLabel := range npc.Movement.AllowedRooms {
				_, ok := world.Rooms[roomLabel]
				if !ok {
					errMsg := "validating: npcs[%d (%q)]: movement: allowed[%d]: no room with label %q exists"
					return world, fmt.Errorf(errMsg, idx, npc.Label, roomIdx, roomLabel)
				}
			}

			for roomIdx, roomLabel := range npc.Movement.ForbiddenRooms {
				_, ok := world.Rooms[roomLabel]
				if !ok {
					errMsg := "validating: npcs[%d (%q)]: movement: forbidden[%d]: no room with label %q exists"
					return world, fmt.Errorf(errMsg, idx, npc.Label, roomIdx, roomLabel)
				}
			}

			// if allowed is set, each room needs to have at least some path
			// from start.
			if len(npc.Movement.AllowedRooms) > 0 {
				source := npc.Start

				for aRoomIdx, aRoom := range npc.Movement.AllowedRooms {
					path := pf.Dijkstra(source, aRoom)
					if len(path) < 1 {
						errMsg := "validating: npcs[%d (%q)]: movement: allowed[%d]: %q is not reachable from start"
						return world, fmt.Errorf(errMsg, idx, npc.Label, aRoomIdx, aRoom)
					}
				}

				if len(npc.Movement.ForbiddenRooms) > 0 {
					// if forbidden is set AND allowed is set, forbidden must
					// refer to rooms reachable from at least one allowed room
					// or start.
					source := npc.Start

					for fRoomIdx, fRoom := range npc.Movement.ForbiddenRooms {
						path := pf.Dijkstra(source, fRoom)
						if len(path) < 1 {
							errMsg := "validating: npcs[%d (%q)]: movement: forbidden[%d]: %q is not reachable from start"
							return world, fmt.Errorf(errMsg, idx, npc.Label, fRoomIdx, fRoom)
						}
					}
				}
			}
		case game.RouteStatic:
			// nothing more to validate, they don't move
		default:
			return world, fmt.Errorf("validating: npcs[%d (%q)]: movement: action: unknown action type '%v'", idx, npc.Label, npc.Movement.Action)
		}

		// check convo tree for duplicate labels
		seenConvoLabels := map[string]int{}
		for diaIdx, diaStep := range npc.Dialog {
			var label string
			if diaStep.Label != "" {
				label = diaStep.Label
			} else {
				label = fmt.Sprintf("%d", diaIdx)
			}

			if seenInIndex, ok := seenConvoLabels[label]; ok {
				return world, fmt.Errorf("validating: npcs[%d (%q)]: dialog[%d]: duplicate label %q; already used by dialog[%d]", idx, npc.Label, diaIdx, label, seenInIndex)
			}
			seenConvoLabels[label] = diaIdx
		}

		// check convo tree for choice label validity
		for diaIdx, diaStep := range npc.Dialog {
			if diaStep.Action == game.DialogChoice {
				choiceNum := -1
				for _, dest := range diaStep.Choices {
					choiceNum++
					if _, ok := seenConvoLabels[dest]; !ok {
						msg := "validating: npcs[%d (%q)]: dialog[%d]: choices[%d]: %q is not a label or index that exists in this NPC's dialog set"
						return world, fmt.Errorf(msg, idx, npc.Label, diaIdx, choiceNum, dest)
					}
				}
			}
		}

		// should be good to go, add the NPC to the world
		npcRef := npc
		world.Rooms[npc.Start].NPCs[npc.Label] = &npcRef
	}

	// TODO: check that no item overwrites another

	return world, nil
}

func validateNPCDef(npc npc, topLevelPronouns map[string]pronounSet) error {
	if npc.Label == "" {
		return fmt.Errorf("must have non-blank 'label' field")
	}
	if npc.Name == "" {
		return fmt.Errorf("must have non-blank 'name' field")
	}

	var empty pronounSet

	if npc.Pronouns != "" {
		if npc.PronounSet != empty {
			return fmt.Errorf("cannot have both 'pronouns' key and custom_pronoun_set defined for the npc")
		}
		if _, ok := topLevelPronouns[npc.Pronouns]; !ok {
			return fmt.Errorf("no pronoun set called %q is defined", npc.Pronouns)
		}
	} else if npc.PronounSet == empty {
		return fmt.Errorf("must have non-blank 'pronouns' key or define custom_pronoun_set for the npc")
	} else {
		err := validatePronounSetDef(npc.PronounSet, topLevelPronouns)
		if err != nil {
			return fmt.Errorf("custom_pronoun_set: %w", err)
		}
	}

	err := validateRouteDef(npc.Movement)
	if err != nil {
		return fmt.Errorf("movement: %w", err)
	}

	for i := range npc.Dialog {
		err := validateDialogStepDef(npc.Dialog[i])
		if err != nil {
			return fmt.Errorf("dialog[%d]: %w", i, err)
		}
	}

	return nil
}

func validateDialogStepDef(ds dialogStep) error {
	diaUpper := strings.ToUpper(ds.Action)
	dia, ok := game.DialogActionsByString[diaUpper]

	if !ok {
		return fmt.Errorf("action: must be one of 'LINE', 'CHOICE', or 'END', not %q", diaUpper)
	}

	switch dia {
	case game.DialogLine:
		if len(ds.Choices) > 0 {
			return fmt.Errorf("'LINE' dialog step type does not use 'choices' key")
		}
		if ds.Content == "" {
			return fmt.Errorf("'LINE' dialog step type requires a string as value of 'content' property")
		}
	case game.DialogChoice:
		if len(ds.Choices) < 2 {
			return fmt.Errorf("'CHOICE' dialog step type must have a list with at least 2 choices as value of 'choices' property")
		}
		if ds.Response != "" {
			return fmt.Errorf("'CHOICE' dialog step type does not use 'response' property")
		}
		if ds.Content == "" {
			return fmt.Errorf("'CHOICE' dialog step type requires a string as value of 'content' property")
		}
	case game.DialogEnd:
		if ds.Response != "" {
			return fmt.Errorf("'END' dialog step type does not use 'response' property")
		}
		if len(ds.Choices) > 0 {
			return fmt.Errorf("'END' dialog step type does not use 'choices' property")
		}
		if ds.Content != "" {
			return fmt.Errorf("'END' dialog step does not use 'content' property")
		}
	default:
		// should never happen but you never know
		return fmt.Errorf("unknown dialog step type: %q", dia)
	}

	return nil
}

func validateRouteDef(ps route) error {
	actUpper := strings.ToUpper(ps.Action)
	act, ok := game.RouteActionsByString[actUpper]

	if !ok {
		return fmt.Errorf("action: must be one of 'STATIC', 'PATROL', or 'WANDER', not %q", actUpper)
	}

	switch act {
	case game.RoutePatrol:
		if len(ps.Path) < 2 {
			return fmt.Errorf("'PATROL' route type must have a list with at least 2 rooms as value of 'path' property")
		}
		if len(ps.Allowed) > 0 {
			return fmt.Errorf("'PATROL' route type does not use 'allowed' property")
		}
		if len(ps.Forbidden) > 0 {
			return fmt.Errorf("'PATROL' route type does not use 'forbidden' property")
		}
	case game.RouteWander:
		if len(ps.Path) > 0 {
			return fmt.Errorf("'WANDER' route type does not use 'path' property")
		}
	case game.RouteStatic:
		if len(ps.Path) > 0 {
			return fmt.Errorf("'STATIC' route type does not use 'path' property")
		}
		if len(ps.Allowed) > 0 {
			return fmt.Errorf("'STATIC' route type does not use 'allowed' property")
		}
		if len(ps.Forbidden) > 0 {
			return fmt.Errorf("'STATIC' route type does not use 'forbidden' property")
		}
	default:
		// should never happen but you never know
		return fmt.Errorf("unknown route type: %q", act)
	}
	return nil
}

// if topLevel is nil, then the top level is being validated.
func validatePronounSetDef(ps pronounSet, topLevel map[string]pronounSet) error {
	if topLevel == nil {
		// then it is a top-level def, and as such MUST have a label
		if ps.Label == "" {
			return fmt.Errorf("top-level pronoun definition must have a label")
		}
	} else if ps.Label != "" {
		return fmt.Errorf("custom pronoun set cannot have a 'label' key")
	}
	return nil
}

func validateRoomDef(r room) error {
	if r.Label == "" {
		return fmt.Errorf("must have non-blank 'label' field")
	}
	if r.Name == "" {
		return fmt.Errorf("must have non-blank 'name' field")
	}
	if r.Description == "" {
		return fmt.Errorf("must have non-blank 'description' field")
	}

	// sanity check that egress aliases are not duplicated
	seenAliases := map[string]bool{}
	for idx, eg := range r.Exits {
		egressErr := validateEgressDef(eg)
		if egressErr != nil {
			return fmt.Errorf("exits[%d]: %w", idx, egressErr)
		}

		for alIdx, alias := range eg.Aliases {
			if _, ok := seenAliases[alias]; ok {
				errMsg := "exits[%d]: aliases[%d]: duplicate egress alias %q in room"
				return fmt.Errorf(errMsg, idx, alIdx, alias)
			}
		}
	}

	for idx, eg := range r.Items {
		itemErr := validateItemDef(eg)
		if itemErr != nil {
			return fmt.Errorf("items[%d]: %w", idx, itemErr)
		}
	}

	return nil
}

func validateEgressDef(eg egress) error {
	if eg.Dest == "" {
		return fmt.Errorf("must have non-blank 'dest' field")
	}
	if eg.Description == "" {
		return fmt.Errorf("must have non-blank 'description' field")
	}
	if eg.Message == "" {
		return fmt.Errorf("must have non-blank 'message' field")
	}

	for idx, al := range eg.Aliases {
		if al == "" {
			return fmt.Errorf("aliases[%d]: must not be blank", idx)
		}
	}

	return nil
}

func validateItemDef(item item) error {
	if item.Label == "" {
		return fmt.Errorf("must have non-blank 'label' field")
	}
	if item.Name == "" {
		return fmt.Errorf("must have non-blank 'name' field")
	}
	if item.Description == "" {
		return fmt.Errorf("must have non-blank 'description' field")
	}

	for idx, al := range item.Aliases {
		if al == "" {
			return fmt.Errorf("aliases[%d]: must not be blank", idx)
		}
	}

	return nil
}
