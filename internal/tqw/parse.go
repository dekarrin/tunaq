package tqw

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dekarrin/tunaq/internal/game"
)

// these two are getting chucked into a char class so order matters
const labelChars = `]A-Z0-9_!?#%^&*().,<>/+=[|{}:;-`
const aliasChars = `]A-Z0-9_!?#%^&*().,<>/+=[|{}:; -`

var (
	labelRegexp             = regexp.MustCompile(fmt.Sprintf(`[%s]+`, labelChars))
	aliasRegexp             = regexp.MustCompile(fmt.Sprintf(`(?:[%s][%s]*)?[%s]+`, labelChars, aliasChars, labelChars))
	identifierBadCharRegexp = regexp.MustCompile(fmt.Sprintf(`[^%s]`, labelChars))
)

func parseManifest(tqw topLevelManifest) (Manifest, error) {
	manif := Manifest{
		Files: tqw.Files,
	}

	return manif, nil
}

type stringSet map[string]bool

type worldSymbols struct {
	roomLabels    stringSet
	itemLabels    stringSet
	itemAliases   stringSet
	pronounLabels stringSet
	npcLabels     stringSet
	npcAliases    stringSet
}

func parseWorldData(tqw topLevelWorldData) (WorldData, error) {
	var err error

	world := WorldData{
		Rooms: make(map[string]*game.Room),
	}

	// first, get all of our game symbols so we can immediately check validity
	// of every reference as we go through it.
	symbols, err := scanSymbols(tqw)
	if err != nil {
		return world, err
	}

	// with all symbols pre-loaded, we can now immediately check validity of
	// every field, including those that are to be a reference to another game
	// object.

	// validate start
	if _, ok := symbols.roomLabels[strings.ToUpper(tqw.World.Start)]; !ok {
		return world, fmt.Errorf("world: start: no room with label %q exists", tqw.World.Start)
	}
	world.Start = strings.ToUpper(tqw.World.Start)

	// validate rooms
	for _, r := range tqw.Rooms {
		if roomErr := validateRoomDef(r, symbols); roomErr != nil {
			return world, fmt.Errorf("rooms[%q]: %w", r.Label, roomErr)
		}

		room := r.toGameRoom()
		world.Rooms[r.Label] = &room
	}

	// validate pronouns and gather them into a map for later conversion of NPC
	// pronouns references.
	pronouns := map[string]pronounSet{
		"SHE/HER":   pronounSetFromGame(game.PronounsFeminine),
		"HE/HIM":    pronounSetFromGame(game.PronounsMasculine),
		"THEY/THEM": pronounSetFromGame(game.PronounsNonBinary),
		"IT/ITS":    pronounSetFromGame(game.PronounsItIts),
	}
	for _, ps := range tqw.Pronouns {
		if err := validatePronounSetDef(ps, nil); err != nil {
			return world, fmt.Errorf("pronouns[%q]: %w", ps.Label, err)
		}

		pronouns[ps.Label] = ps
	}

	// validate NPCs
	// TODO: should NPCs just be a top-level data item? the first thing that
	// game.State does is UN-add them to rooms and re-index them... by going
	// through all rooms. To be sure it's capped at the number of rooms a human
	// could reasonably make, but thats just not performant.
	for _, npc := range tqw.NPCs {
		if npc.Movement.Action == "" {
			npc.Movement.Action = "STATIC"
		}

		// set any blank dialog types to line
		for idx, ds := range npc.Dialogs {
			if ds.Action == "" {
				ds.Action = "LINE"
				npc.Dialogs[idx] = ds
			}
		}

		// below call will check that NPC does not have BOTH custom_pronoun_set
		// and pronouns defined, and that's why we defer setting the "real"
		// pronouns from npcs.pronouns key until after.
		if err := validateNPCDef(npc, pronouns, world.Rooms, symbols); err != nil {
			return world, fmt.Errorf("npcs[%q]: %w", npc.Label, err)
		}
		// set pronouns to actual
		if npc.Pronouns != "" {
			npc.PronounSet = pronouns[strings.ToUpper(npc.Pronouns)]
		}

		// done parsing, NPC is good to go, add it to the world
		gameNPC := npc.toGameNPC()
		world.Rooms[gameNPC.Start].NPCs[gameNPC.Label] = &gameNPC
	}

	return world, nil
}

// this builds up a pre-list of 'seen' labels and aliases so we can check for
// pointers later. All of them will be checked for conflicts within their own
// class of objects and all of them will be checked for validity as either a
// label or an alias.
//
// Despite not being returned here, egress aliases will be checked for alias
// validity as well as conflict checked against other egress aliases in the
// room, global item aliases, and global NPC aliases.
//
// Despite not being returned here, NPC dialog labels will be checked for label
// validity as well as conflict checked against other dialog labels in the NPC's
// convo tree.
//
// Error is returned if any alias or label fails to follow its naming rules or
// if any of them conflicts with another. Otherwise, global symbols are returned
// so that they can be used to check references to them. The global symbols
// returned will all be converted to upper case already.
func scanSymbols(top topLevelWorldData) (symbols worldSymbols, err error) {
	syms := worldSymbols{
		roomLabels:  make(stringSet),
		itemLabels:  make(stringSet),
		itemAliases: make(stringSet),

		// hard-code the pre-existing pronoun labels
		pronounLabels: stringSet{
			"SHE/HER":   true,
			"HE/HIM":    true,
			"THEY/THEM": true,
			"IT/ITS":    true,
		},

		npcLabels:  make(stringSet),
		npcAliases: make(stringSet),
	}

	// not doing egressAliases because that is not something that other things
	// can conflict with and passing item symbols to a room check should be
	// sufficient to detect it
	for _, r := range top.Rooms {
		rLabelUpper := strings.ToUpper(r.Label)
		if err := checkLabel(rLabelUpper, syms.roomLabels, "a room"); err != nil {
			return syms, fmt.Errorf("room %q: %w", r.Label, err)
		}
		syms.roomLabels[rLabelUpper] = true

		for _, it := range r.Items {
			itLabelUpper := strings.ToUpper(it.Label)
			if err := checkLabel(itLabelUpper, syms.itemLabels, "an item"); err != nil {
				return syms, fmt.Errorf("room %q: item %q: %w", r.Label, it.Label, err)
			}
			syms.itemLabels[itLabelUpper] = true

			for _, alias := range it.Aliases {
				aliasUpper := strings.ToUpper(alias)
				if err := checkAlias(aliasUpper, syms.itemAliases); err != nil {
					return syms, fmt.Errorf("room %q: item %q: alias %q: %w", r.Label, it.Label, alias, err)
				}
				syms.itemLabels[itLabelUpper] = true
			}
		}
	}

	// scan pronouns
	for _, ps := range top.Pronouns {
		psLabelUpper := strings.ToUpper(ps.Label)
		if err := checkLabel(psLabelUpper, syms.pronounLabels, "pronouns"); err != nil {
			return syms, fmt.Errorf("pronouns %q: %w", ps.Label, err)
		}
		syms.pronounLabels[psLabelUpper] = true
	}

	// scan npc labels and aliases
	for _, npc := range top.NPCs {
		npcLabelUpper := strings.ToUpper(npc.Label)
		if err := checkLabel(npcLabelUpper, syms.npcLabels, "an NPC"); err != nil {
			return syms, fmt.Errorf("npc %q: %w", npc.Label, err)
		}
		syms.npcLabels[npcLabelUpper] = true

		for _, alias := range npc.Aliases {
			aliasUpper := strings.ToUpper(alias)
			if err := checkAlias(aliasUpper, syms.npcAliases); err != nil {
				return syms, fmt.Errorf("npc %q: alias %q: %w", npc.Label, alias, err)
			}
			syms.npcAliases[aliasUpper] = true
		}
	}

	// end of getting global symbols
	// now check the non-global ones

	// egress aliases (against each other, npc aliases, and item aliases)
	for _, r := range top.Rooms {
		exitAliasesInRoom := make(stringSet)
		for exitIdx, eg := range r.Exits {
			for _, alias := range eg.Aliases {
				aliasUpper := strings.ToUpper(alias)

				// check against other room aliases
				if err := checkAlias(aliasUpper, exitAliasesInRoom); err != nil {
					return syms, fmt.Errorf("room %q: exit %d: alias %q: %w", r.Label, exitIdx, alias, err)
				}

				// check against item aliases
				if err := checkAlias(aliasUpper, syms.itemLabels); err != nil {
					// first check alias check would have caught invalid label,
					// so if this failed it MUST be due to matching the conflict set
					return syms, fmt.Errorf("room %q: exit %d: alias %q conflicts with item alias", r.Label, exitIdx, alias)
				}

				// check against NPC aliases
				if err := checkAlias(aliasUpper, syms.npcLabels); err != nil {
					// first alias check would have caught invalid label,
					// so if this failed it MUST be due to matching the conflict set
					return syms, fmt.Errorf("room %q: exit %d: alias %q conflicts with NPC alias", r.Label, exitIdx, alias)
				}

				exitAliasesInRoom[aliasUpper] = true
			}
		}
	}

	// NPC dialog step labels (against each other only as they will never be used by normal command parsing)
	// DEFAULT LABEL: if a label isn't specified then it will default to being the string conversion of the index of the
	// step.
	for _, npc := range top.NPCs {
		diaLabelsInTree := make(stringSet)
		for idx, dia := range npc.Dialogs {
			diaLabelUpper := strings.ToUpper(dia.Label)
			if diaLabelUpper == "" {
				diaLabelUpper = fmt.Sprintf("%d", idx)
			}

			if err := checkLabel(diaLabelUpper, diaLabelsInTree, "a step in this NPC's dialog tree"); err != nil {
				return syms, fmt.Errorf("npc %q: dialogs[%q]: %w", npc.Label, idx, err)
			}
			diaLabelsInTree[diaLabelUpper] = true
		}
	}

	return syms, nil
}

func validateNPCDef(npc npc, topLevelPronouns map[string]pronounSet, parsedRooms map[string]*game.Room, syms worldSymbols) error {
	if npc.Label == "" {
		return fmt.Errorf("must have non-blank 'label' field")
	}
	if npc.Name == "" {
		return fmt.Errorf("must have non-blank 'name' field")
	}

	// check start for valid reference
	startUpper := strings.ToUpper(npc.Start)
	if _, ok := syms.roomLabels[startUpper]; !ok {
		return fmt.Errorf("start: no room with label %q exists", npc.Start)
	}

	var empty pronounSet

	if npc.Pronouns != "" {
		if npc.PronounSet != empty {
			return fmt.Errorf("cannot have both 'pronouns' key and custom_pronoun_set defined for the npc")
		}
		if _, ok := topLevelPronouns[strings.ToUpper(npc.Pronouns)]; !ok {
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

	err := validateRouteDef(npc.Movement, parsedRooms, npc.Start, syms)
	if err != nil {
		return fmt.Errorf("movement: %w", err)
	}

	// find all labels that exist in the dialog tree for ref checking (prior
	// checks already ensured every label is unique)
	diaLabels := make(stringSet)
	for i := range npc.Dialogs {
		lbl := strings.ToUpper(npc.Dialogs[i].Label)
		if lbl == "" {
			lbl = fmt.Sprintf("%d", i)
		}
		diaLabels[lbl] = true
	}

	// now that the labels are
	for i := range npc.Dialogs {
		err := validateDialogStepDef(npc.Dialogs[i], diaLabels)
		if err != nil {
			return fmt.Errorf("dialogs[%d]: %w", i, err)
		}
	}

	return nil
}

func validateDialogStepDef(ds dialogStep, allDiaLabels stringSet) error {
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

		// now we check the choices for valid ref
		for idx, ch := range ds.Choices {
			if len(ch) != 2 {
				return fmt.Errorf("choices[%d]: must be a list containing what to say and label of step to jump to", idx)
			}
			if ch[0] == "" {
				return fmt.Errorf("choices[%d]: first item (what to say) cannot be blank", idx)
			}
			if _, ok := allDiaLabels[strings.ToUpper(ch[1])]; !ok {
				return fmt.Errorf("choices[%d]: %q is not the label of any step in this NPC's dialog tree", idx, ch[1])
			}
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

func validateRouteDef(ps route, parsedRooms map[string]*game.Room, npcStart string, syms worldSymbols) error {
	actUpper := strings.ToUpper(ps.Action)
	act, ok := game.RouteActionsByString[actUpper]

	if !ok {
		return fmt.Errorf("action: must be one of 'STATIC', 'PATROL', or 'WANDER', not %q", actUpper)
	}

	pf := game.Pathfinder{World: parsedRooms}
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

		// now check patrol path (note: pathfinder validation will have added benefit of
		// validating that each component of path is a valid room label)

		pathUpper := make([]string, len(ps.Path))
		for i := range ps.Path {
			pathUpper[i] = strings.ToUpper(ps.Path[i])
		}

		// can npc get to initial position?
		err := pf.ValidatePath(append([]string{strings.ToUpper(npcStart)}, pathUpper[0]), false)
		if err != nil {
			return err
		}

		// once at initial, can npc loop through patrol?
		err = pf.ValidatePath(pathUpper, true)
		if err != nil {
			return err
		}
	case game.RouteWander:
		if len(ps.Path) > 0 {
			return fmt.Errorf("'WANDER' route type does not use 'path' property")
		}

		// now check forbidden and allowed if set (note: pathfinder validation
		// will have added benefit of validating that each component of
		// forbidden and allowed is a valid room label)

		// for forbidden, we will never run the pathfinder validation (because
		// if an NPC can't get to a forbidden room it has the same effect as
		// intended, and there's no reason to do additional checking), so we
		// need to explicitly check that each component is at least a real room
		// label glub
		for idx, label := range ps.Forbidden {
			labelUpper := strings.ToUpper(label)
			_, ok := syms.roomLabels[labelUpper]
			if !ok {
				return fmt.Errorf("forbidden[%d]: no room with label %q exists", idx, labelUpper)
			}
		}

		// if allowed is set, each room needs to have at least some path
		// from start (this has side effect of also ensuring each room is a
		// label that exists)
		if len(ps.Allowed) > 0 {
			source := strings.ToUpper(npcStart)

			for idx, label := range ps.Allowed {
				labelUpper := strings.ToUpper(label)
				path := pf.Dijkstra(source, labelUpper)
				if len(path) < 1 {
					return fmt.Errorf("allowed[%d]: %q is not reachable from start", idx, label)
				}
			}
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

// validation does not check for symbol uniqueness or name rules violations, but
// it DOES check to ensure that valid symbols are being pointed to by references
// within the room (such as Dest attribute of an egress).
func validateRoomDef(r room, syms worldSymbols) error {
	if r.Label == "" {
		return fmt.Errorf("must have non-blank 'label' field")
	}
	if r.Name == "" {
		return fmt.Errorf("must have non-blank 'name' field")
	}
	if r.Description == "" {
		return fmt.Errorf("must have non-blank 'description' field")
	}

	// validate egresses
	for idx, eg := range r.Exits {
		egressErr := validateEgressDef(eg, syms)
		if egressErr != nil {
			return fmt.Errorf("exits[%d]: %w", idx, egressErr)
		}
	}

	// items must have their labels and aliases checked against ALL items so not
	// doing the check here
	for idx, eg := range r.Items {
		itemErr := validateItemDef(eg)
		if itemErr != nil {
			return fmt.Errorf("items[%d]: %w", idx, itemErr)
		}
	}

	return nil
}

func validateEgressDef(eg egress, syms worldSymbols) error {
	if eg.Dest == "" {
		return fmt.Errorf("must have non-blank 'dest' field")
	}
	if eg.Description == "" {
		return fmt.Errorf("must have non-blank 'description' field")
	}
	if eg.Message == "" {
		return fmt.Errorf("must have non-blank 'message' field")
	}

	// do not check alias naming rules and uniqueness here, that has already been
	// done during call to scanSymbols, but DO check to ensure that the dest is
	// a valid pointer
	if _, ok := syms.roomLabels[strings.ToUpper(eg.Dest)]; !ok {
		return fmt.Errorf("dest: no room has label %q", strings.ToUpper(eg.Dest))
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

	// do not check alias naming rules and uniqueness here, that has already
	// been done during call to scanSymbols.

	return nil
}

func checkAlias(alias string, conflictSet stringSet) error {
	if _, ok := conflictSet[alias]; ok {
		return fmt.Errorf("alias conflicts with another alias")
	}

	if !aliasRegexp.MatchString(alias) {
		// we know the alias is bad; first check if it's due to a space at start or end so we can give a special message
		if strings.HasPrefix(alias, " ") {
			return fmt.Errorf("aliases cannot start with a space")
		}
		if strings.HasSuffix(alias, " ") {
			return fmt.Errorf("aliases cannot end with a space")
		}

		// if we got this far there's an invalid char somewhere in the string, and its not a leading or trailing space
		badChar := identifierBadCharRegexp.FindString(alias)
		if badChar == "" {
			// something has gone horribly wrong with coding of regular expressions
			panic(fmt.Sprintf("could not identify bad char in alias %q", alias))
		}

		return fmt.Errorf("aliases cannot contain the character %q", badChar)
	}

	return nil
}

func checkLabel(label string, conflictSet stringSet, labeled string) error {
	if _, ok := conflictSet[label]; ok {
		return fmt.Errorf("label %q has already been used for %s", label, labeled)
	}

	if !labelRegexp.MatchString(label) {
		badChar := identifierBadCharRegexp.FindString(label)
		if badChar == "" {
			// something has gone horribly wrong with coding of regular expressions
			panic(fmt.Sprintf("could not identify bad char in alias %q", label))
		}

		return fmt.Errorf("%q has the %q character in it which is not allowed for labels", label, badChar)
	}

	return nil
}
