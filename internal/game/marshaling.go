package game

import (
	"encoding/json"
	"fmt"
	"strings"
)

type jsonNPC struct {
	Label       string           `json:"label"`
	Name        string           `json:"name"`
	Pronouns    jsonPronounSet   `json:"pronouns"`
	Description string           `json:"description"`
	Start       string           `json:"start"`
	Movement    jsonRoute        `json:"movement"`
	Dialog      []jsonDialogStep `json:"dialog"`
}

func (jn jsonNPC) toNPC() NPC {
	npc := NPC{
		Label:       jn.Label,
		Name:        jn.Name,
		Pronouns:    jn.Pronouns.toPronounSet(),
		Description: jn.Description,
		Start:       jn.Start,
		Movement:    jn.Movement.toRoute(),
		Dialog:      make([]DialogStep, len(jn.Dialog)),
	}

	for i := range jn.Dialog {
		npc.Dialog[i] = jn.Dialog[i].toDialogStep()
	}

	return npc
}

type jsonRoute struct {
	Action         string   `json:"action"`
	Path           []string `json:"path"`
	ForbiddenRooms []string `json:"forbiddenRooms"`
	AllowedRooms   []string `json:"allowedRooms"`
}

func (jr jsonRoute) toRoute() Route {
	act, ok := RouteActionsByString[strings.ToUpper(jr.Action)]
	if !ok {
		act = RouteStatic
	}

	r := Route{
		Action:         act,
		Path:           make([]string, len(jr.Path)),
		ForbiddenRooms: make([]string, len(jr.ForbiddenRooms)),
		AllowedRooms:   make([]string, len(jr.AllowedRooms)),
	}

	copy(r.Path, jr.Path)
	copy(r.ForbiddenRooms, jr.ForbiddenRooms)
	copy(r.AllowedRooms, jr.AllowedRooms)

	return r
}

type jsonDialogStep struct {
	Action   string     `json:"action"`
	Label    string     `json:"label"`
	Content  string     `json:"content"`
	Response string     `json:"response"`
	Choices  [][]string `json:"choices"`
}

func (jds jsonDialogStep) toDialogStep() DialogStep {
	act, ok := DialogActionsByString[strings.ToUpper(jds.Action)]
	if !ok {
		act = DialogLine
	}

	ds := DialogStep{
		Action:   act,
		Label:    jds.Label,
		Content:  jds.Content,
		Response: jds.Response,
		Choices:  make(map[string]string),
	}

	for _, ch := range jds.Choices {
		if len(ch) < 2 {
			continue
		}

		choice := ch[0]
		dest := ch[1]
		ds.Choices[choice] = dest
	}

	return ds
}

type jsonPronounSet struct {
	Nominative string `json:"nominative"`
	Objective  string `json:"objective"`
	Possessive string `json:"possessive"`
	Determiner string `json:"determiner"`
	Reflexive  string `json:"reflexive"`

	// Label is only filled when the JSON object was only a string.
	Label string
}

func (jp *jsonPronounSet) UnmarshalText(text []byte) error {
	// if its just text then set Label instead of actually setting the props.
	jp.Label = string(text)
	return nil
}

func (ps PronounSet) toJSON() jsonPronounSet {
	jp := jsonPronounSet{
		Nominative: ps.Nominative,
		Objective:  ps.Objective,
		Possessive: ps.Possessive,
		Determiner: ps.Determiner,
		Reflexive:  ps.Reflexive,
	}

	return jp
}

func (jp jsonPronounSet) toPronounSet() PronounSet {
	ps := PronounSet{
		Nominative: strings.ToUpper(jp.Nominative),
		Objective:  strings.ToUpper(jp.Objective),
		Possessive: strings.ToUpper(jp.Possessive),
		Determiner: strings.ToUpper(jp.Determiner),
		Reflexive:  strings.ToUpper(jp.Reflexive),
	}

	// default to they/them fills
	if ps.Nominative == "" {
		ps.Nominative = "THEY"
	}
	if ps.Objective == "" {
		ps.Objective = "THEM"
	}
	if ps.Possessive == "" {
		ps.Possessive = "THEIRS"
	}
	if ps.Determiner == "" {
		ps.Determiner = "THEIR"
	}
	if ps.Reflexive == "" {
		ps.Reflexive = "THEMSELF"
	}

	return ps
}

type jsonItem struct {
	Label       string   `json:"label"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Aliases     []string `json:"aliases"`
}

func (ji jsonItem) toItem() Item {
	it := Item{
		Label:       ji.Label,
		Name:        ji.Name,
		Description: ji.Description,
		Aliases:     make([]string, len(ji.Aliases)),
	}

	copy(it.Aliases, ji.Aliases)

	return it
}

type jsonEgress struct {
	DestLabel     string   `json:"destLabel"`
	Description   string   `json:"description"`
	TravelMessage string   `json:"travelMessage"`
	Aliases       []string `json:"aliases"`
}

func (je jsonEgress) toEgress() Egress {
	eg := Egress{
		DestLabel:     je.DestLabel,
		Description:   je.Description,
		TravelMessage: je.TravelMessage,
		Aliases:       make([]string, len(je.Aliases)),
	}

	copy(eg.Aliases, je.Aliases)

	return eg
}

type jsonRoom struct {
	Label       string       `json:"label"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Exits       []jsonEgress `json:"exits"`
	Items       []jsonItem   `json:"items"`
}

func (jr jsonRoom) toRoom() Room {
	r := Room{
		Label:       jr.Label,
		Name:        jr.Name,
		Description: jr.Description,
		Exits:       make([]Egress, len(jr.Exits)),
		Items:       make([]Item, len(jr.Items)),
	}

	for i := range jr.Exits {
		r.Exits[i] = jr.Exits[i].toEgress()
	}
	for i := range jr.Items {
		r.Items[i] = jr.Items[i].toItem()
	}

	return r
}

type jsonWorld struct {
	Rooms    []jsonRoom                `json:"rooms"`
	Start    string                    `json:"start"`
	NPCs     []jsonNPC                 `json:"npcs"`
	Pronouns map[string]jsonPronounSet `json:"pronouns"`
}

// ParseWorldFromJSON takes in raw json bytes, reads it for a world definition,
// and returns the rooms as well as the label of the starting room.
//
// Note: Uses module-global variables as part of operation. Absolutely not
// thread-safe and calling more than once concurrently will lead to unexpected
// results.
func ParseWorldFromJSON(jsonData []byte) (world map[string]*Room, startRoom string, err error) {
	var loadedWorld jsonWorld

	if jsonErr := json.Unmarshal(jsonData, &loadedWorld); jsonErr != nil {
		return nil, "", fmt.Errorf("decoding JSON data: %w", jsonErr)
	}

	startRoom = loadedWorld.Start
	world = make(map[string]*Room)

	for idx, r := range loadedWorld.Rooms {
		if roomErr := validateRoomDef(r); roomErr != nil {
			return nil, "", fmt.Errorf("parsing: rooms[%d]: %w", idx, roomErr)
		}

		if _, ok := world[r.Label]; ok {
			return nil, "", fmt.Errorf("parsing: rooms[%d]: duplicate room label %q", idx, r.Label)
		}

		room := r.toRoom()
		world[r.Label] = &room
	}

	pronouns := map[string]jsonPronounSet{
		"SHE/HER":   PronounsFeminine.toJSON(),
		"HE/HIM":    PronounsMasculine.toJSON(),
		"THEY/THEM": PronounsNonBinary.toJSON(),
		"IT/ITS":    PronounsItIts.toJSON(),
	}

	// check loaded pronouns
	for name, ps := range loadedWorld.Pronouns {
		if err := validatePronounSetDef(ps, nil); err != nil {
			return nil, "", fmt.Errorf("parsing: pronouns[%s]: %w", name, err)
		}

		if _, ok := pronouns[name]; ok {
			return nil, "", fmt.Errorf("parsing: pronouns[%s]: duplicate pronoun name %q", name, name)
		}

		pronouns[name] = ps
	}

	npcs := make([]NPC, 0)
	// validate individual npcs
	for idx, npc := range loadedWorld.NPCs {
		if err := validateNPCDef(npc, pronouns); err != nil {
			return nil, "", fmt.Errorf("parsing: npcs[%d]: %w", idx, err)
		}

		// set pronouns to actual
		if npc.Pronouns.Label != "" {
			npc.Pronouns = pronouns[strings.ToUpper(npc.Pronouns.Label)]
		}

		npcs = append(npcs, npc.toNPC())
	}

	// now that they are all loaded and individually checked for validity,
	// ensure that all room egresses are valid existing labels
	for roomIdx, r := range loadedWorld.Rooms {
		for egressIdx, eg := range r.Exits {
			if _, ok := world[eg.DestLabel]; !ok {
				errMsg := "validating: rooms[%d]: exits[%d]: no room with label %q exists"
				return nil, "", fmt.Errorf(errMsg, roomIdx, egressIdx, eg.DestLabel)
			}
		}
	}

	// check that the start actually points to a real location
	if _, ok := world[loadedWorld.Start]; !ok {
		return nil, "", fmt.Errorf("validating: start: no room with label %q exists", startRoom)
	}

	// ensure that all npc routes are valid, that convo trees make sense, then place NPCs in their start rooms
	pf := Pathfinder{World: world}
	for idx, npc := range npcs {
		// first check start label
		_, ok := world[npc.Start]
		if !ok {
			return nil, "", fmt.Errorf("validating: npcs[%d]: start: no room with label %q exists", idx, npc.Start)
		}

		// then check route based on movement type
		switch npc.Movement.Action {
		case RoutePatrol:
			err = pf.ValidatePath(append(npc.Movement.Path, npc.Start), true)
			if err != nil {
				return nil, "", fmt.Errorf("validating: npcs[%d]: %w", idx, err)
			}
		case RouteWander:
			for roomIdx, roomLabel := range npc.Movement.AllowedRooms {
				_, ok := world[roomLabel]
				if !ok {
					errMsg := "validating: npcs[%d]: movement: allowedRooms[%d]: no room with label %q exists"
					return nil, "", fmt.Errorf(errMsg, idx, roomIdx, roomLabel)
				}
			}

			for roomIdx, roomLabel := range npc.Movement.ForbiddenRooms {
				_, ok := world[roomLabel]
				if !ok {
					errMsg := "validating: npcs[%d]: movement: forbiddenRooms[%d]: no room with label %q exists"
					return nil, "", fmt.Errorf(errMsg, idx, roomIdx, roomLabel)
				}
			}

			// if allowed is set, each room needs to have at least some path
			// from start.
			if len(npc.Movement.AllowedRooms) > 0 {
				source := npc.Start

				for aRoomIdx, aRoom := range npc.Movement.AllowedRooms {
					path := pf.Dijkstra(source, aRoom)
					if len(path) < 1 {
						errMsg := "validating: npcs[%d]: movement: allowedRooms[%d]: %q is not reachable from start"
						return nil, "", fmt.Errorf(errMsg, idx, aRoomIdx, aRoom)
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
							errMsg := "validating: npcs[%d]: movement: forbiddenRooms[%d]: %q is not reachable from start"
							return nil, "", fmt.Errorf(errMsg, idx, fRoomIdx, fRoom)
						}
					}
				}
			}
		case RouteStatic:
			// nothing more to validate, they don't move
		default:
			return nil, "", fmt.Errorf("validating: npcs[%d]: movement: action: unknown action type '%v'", idx, npc.Movement.Action)
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
				return nil, "", fmt.Errorf("validating: npcs[%d]: dialog[%d]: duplicate label %q; already used by dialog[%d]", idx, diaIdx, label, seenInIndex)
			}
			seenConvoLabels[label] = diaIdx
		}

		// check convo tree for choice label validity
		for diaIdx, diaStep := range npc.Dialog {
			if diaStep.Action == DialogChoice {
				choiceNum := -1
				for _, dest := range diaStep.Choices {
					choiceNum++
					if _, ok := seenConvoLabels[dest]; ok {
						msg := "validating: npcs[%d]: dialog[%d]: choices[%d]: %q is not a label or index that exists in this NPC's dialog set"
						return nil, "", fmt.Errorf(msg, idx, diaIdx, choiceNum, dest)
					}
				}
			}
		}

		// should be good to go, add the NPC to the world
		world[npc.Start].NPCs = append(world[npc.Start].NPCs, npc)
	}

	// TODO: check that no item overwrites another

	return world, startRoom, nil
}

func validateNPCDef(npc jsonNPC, topLevelPronouns map[string]jsonPronounSet) error {
	if npc.Label == "" {
		return fmt.Errorf("must have non-blank 'label' field")
	}
	if npc.Name == "" {
		return fmt.Errorf("must have non-blank 'name' field")
	}

	// check pronouns are set or refer to one
	err := validatePronounSetDef(npc.Pronouns, topLevelPronouns)
	if err != nil {
		return fmt.Errorf("pronouns: %w", err)
	}

	err = validateRouteDef(npc.Movement)
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

func validateDialogStepDef(ds jsonDialogStep) error {
	diaUpper := strings.ToUpper(ds.Action)
	dia, ok := DialogActionsByString[diaUpper]

	if !ok {
		return fmt.Errorf("action: must be one of 'LINE', 'CHOICE', or 'END', not %q", diaUpper)
	}

	switch dia {
	case DialogLine:
		if len(ds.Choices) > 0 {
			return fmt.Errorf("'LINE' dialog step type does not use 'choices' key")
		}
		if ds.Content == "" {
			return fmt.Errorf("'LINE' dialog step type requires a string as value of 'content' property")
		}
	case DialogChoice:
		if len(ds.Choices) < 2 {
			return fmt.Errorf("'CHOICE' dialog step type must have a list with at least 2 choices as value of 'choices' property")
		}
		if ds.Response != "" {
			return fmt.Errorf("'CHOICE' dialog step type does not use 'response' property")
		}
		if ds.Content == "" {
			return fmt.Errorf("'CHOICE' dialog step type requires a string as value of 'content' property")
		}
	case DialogEnd:
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

func validateRouteDef(ps jsonRoute) error {
	actUpper := strings.ToUpper(ps.Action)
	act, ok := RouteActionsByString[actUpper]

	if !ok {
		return fmt.Errorf("action: must be one of 'STATIC', 'PATROL', or 'WANDER', not %q", actUpper)
	}

	switch act {
	case RoutePatrol:
		if len(ps.Path) < 2 {
			return fmt.Errorf("'PATROL' route type must have a list with at least 2 rooms as value of 'path' property")
		}
		if len(ps.AllowedRooms) > 0 {
			return fmt.Errorf("'PATROL' route type does not use 'allowedRooms' property")
		}
		if len(ps.ForbiddenRooms) > 0 {
			return fmt.Errorf("'PATROL' route type does not use 'forbiddenRooms' property")
		}
	case RouteWander:
		if len(ps.Path) > 0 {
			return fmt.Errorf("'WANDER' route type does not use 'path' property")
		}
	case RouteStatic:
		if len(ps.Path) < 2 {
			return fmt.Errorf("'STATIC' route type does not use 'path' property")
		}
		if len(ps.AllowedRooms) < 2 {
			return fmt.Errorf("'STATIC' route type does not use 'allowedRooms' property")
		}
		if len(ps.ForbiddenRooms) > 0 {
			return fmt.Errorf("'STATIC' route type does not use 'forbiddenRooms' property")
		}
	default:
		// should never happen but you never know
		return fmt.Errorf("unknown route type: %q", act)
	}
	return nil
}

// if topLevel is nil, then the top level is being validated.
func validatePronounSetDef(ps jsonPronounSet, topLevel map[string]jsonPronounSet) error {
	if ps.Label != "" {
		if topLevel == nil {
			return fmt.Errorf("top-level pronoun must be full pronoun definition, not a label (%q)", ps.Label)
		}
		if _, ok := topLevel[strings.ToUpper(ps.Label)]; ok {
			return fmt.Errorf("no pronoun set called %q exists", ps.Label)
		}
	}
	return nil
}

func validateRoomDef(r jsonRoom) error {
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

func validateEgressDef(eg jsonEgress) error {
	if eg.DestLabel == "" {
		return fmt.Errorf("must have non-blank 'destLabel' field")
	}
	if eg.Description == "" {
		return fmt.Errorf("must have non-blank 'description' field")
	}
	if eg.TravelMessage == "" {
		return fmt.Errorf("must have non-blank 'travelMessage' field")
	}

	for idx, al := range eg.Aliases {
		if al == "" {
			return fmt.Errorf("aliases[%d]: must not be blank", idx)
		}
	}

	return nil
}

func validateItemDef(item jsonItem) error {
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
