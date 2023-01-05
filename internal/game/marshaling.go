package game

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

type tqwManifest struct {
	Format string   `toml:"format"`
	Type   string   `toml:"type"`
	Files  []string `toml:"files"`
}

type twqNPC struct {
	Label       string          `toml:"label"`
	Name        string          `toml:"name"`
	Pronouns    string          `toml:"pronouns"`
	PronounSet  tqwPronounSet   `toml:"customPronounSet"`
	Description string          `toml:"description"`
	Start       string          `toml:"start"`
	Movement    tqwRoute        `toml:"movement"`
	Dialog      []tqwDialogStep `toml:"dialog"`
}

func (tn twqNPC) toNPC() NPC {
	npc := NPC{
		Label:       strings.ToUpper(tn.Label),
		Name:        tn.Name,
		Pronouns:    tn.PronounSet.toPronounSet(),
		Description: tn.Description,
		Start:       tn.Start,
		Movement:    tn.Movement.toRoute(),
		Dialog:      make([]DialogStep, len(tn.Dialog)),
	}

	for i := range tn.Dialog {
		npc.Dialog[i] = tn.Dialog[i].toDialogStep()
	}

	return npc
}

type tqwRoute struct {
	Action         string   `toml:"action"`
	Path           []string `toml:"path"`
	ForbiddenRooms []string `toml:"forbiddenRooms"`
	AllowedRooms   []string `toml:"allowedRooms"`
}

func (tr tqwRoute) toRoute() Route {
	act, ok := RouteActionsByString[strings.ToUpper(tr.Action)]
	if !ok {
		act = RouteStatic
	}

	r := Route{
		Action:         act,
		Path:           make([]string, len(tr.Path)),
		ForbiddenRooms: make([]string, len(tr.ForbiddenRooms)),
		AllowedRooms:   make([]string, len(tr.AllowedRooms)),
	}

	copy(r.Path, tr.Path)
	copy(r.ForbiddenRooms, tr.ForbiddenRooms)
	copy(r.AllowedRooms, tr.AllowedRooms)

	return r
}

type tqwDialogStep struct {
	Action   string     `toml:"action"`
	Label    string     `toml:"label"`
	Content  string     `toml:"content"`
	Response string     `toml:"response"`
	Choices  [][]string `toml:"choices"`
}

func (tds tqwDialogStep) toDialogStep() DialogStep {
	act, ok := DialogActionsByString[strings.ToUpper(tds.Action)]
	if !ok {
		act = DialogLine
	}

	ds := DialogStep{
		Action:   act,
		Label:    tds.Label,
		Content:  tds.Content,
		Response: tds.Response,
		Choices:  make(map[string]string),
	}

	for _, ch := range tds.Choices {
		if len(ch) < 2 {
			continue
		}

		choice := ch[0]
		dest := ch[1]
		ds.Choices[choice] = dest
	}

	return ds
}

type tqwPronounSet struct {
	Nominative string `toml:"nominative"`
	Objective  string `toml:"objective"`
	Possessive string `toml:"possessive"`
	Determiner string `toml:"determiner"`
	Reflexive  string `toml:"reflexive"`
}

func pronounSetToTWQ(ps PronounSet) tqwPronounSet {
	tp := tqwPronounSet{
		Nominative: ps.Nominative,
		Objective:  ps.Objective,
		Possessive: ps.Possessive,
		Determiner: ps.Determiner,
		Reflexive:  ps.Reflexive,
	}

	return tp
}

func (tp tqwPronounSet) toPronounSet() PronounSet {
	ps := PronounSet{
		Nominative: strings.ToUpper(tp.Nominative),
		Objective:  strings.ToUpper(tp.Objective),
		Possessive: strings.ToUpper(tp.Possessive),
		Determiner: strings.ToUpper(tp.Determiner),
		Reflexive:  strings.ToUpper(tp.Reflexive),
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

type tqwItem struct {
	Label       string   `toml:"label"`
	Name        string   `toml:"name"`
	Description string   `toml:"description"`
	Aliases     []string `toml:"aliases"`
}

func (ti tqwItem) toItem() Item {
	item := Item{
		Label:       ti.Label,
		Name:        ti.Name,
		Description: ti.Description,
		Aliases:     make([]string, len(ti.Aliases)),
	}

	copy(item.Aliases, ti.Aliases)

	return item
}

type tqwEgress struct {
	DestLabel     string   `toml:"destLabel"`
	Description   string   `toml:"description"`
	TravelMessage string   `toml:"travelMessage"`
	Aliases       []string `toml:"aliases"`
}

func (te tqwEgress) toEgress() Egress {
	eg := Egress{
		DestLabel:     te.DestLabel,
		Description:   te.Description,
		TravelMessage: te.TravelMessage,
		Aliases:       make([]string, len(te.Aliases)),
	}

	copy(eg.Aliases, te.Aliases)

	return eg
}

type tqwRoom struct {
	Label       string      `toml:"label"`
	Name        string      `toml:"name"`
	Description string      `toml:"description"`
	Exits       []tqwEgress `toml:"exits"`
	Items       []tqwItem   `toml:"items"`
}

func (tr tqwRoom) toRoom() Room {
	r := Room{
		Label:       tr.Label,
		Name:        tr.Name,
		Description: tr.Description,
		Exits:       make([]Egress, len(tr.Exits)),
		Items:       make([]Item, len(tr.Items)),
		NPCs:        make(map[string]*NPC),
	}

	for i := range tr.Exits {
		r.Exits[i] = tr.Exits[i].toEgress()
	}
	for i := range tr.Items {
		r.Items[i] = tr.Items[i].toItem()
	}

	return r
}

type tqwFileInfo struct {
	Format string `toml:"format"`
	Type   string `toml:"type"`
}

type tqwWorld struct {
	Start string `toml:"start"`
}

// tqwWorldData is the top-level structure containing all keys in a complete TQW
// 'DATA' type file.
type tqwWorldData struct {
	Format   string                   `toml:"format"`
	Type     string                   `toml:"type"`
	Rooms    []tqwRoom                `toml:"rooms"`
	World    tqwWorld                 `toml:"world"`
	NPCs     []twqNPC                 `toml:"npcs"`
	Pronouns map[string]tqwPronounSet `toml:"pronouns"`
}

// ParseManifestFromTOML takes in raw TOML bytes, reads it for manifest data,
// and returns the files in the manifest.
func ParseManifestFromTOML(manifestData []byte) (Manifest, error) {
	var loaded tqwManifest

	if unmarshalErr := toml.Unmarshal(manifestData, &loaded); unmarshalErr != nil {
		return Manifest{}, unmarshalErr
	}

	if strings.ToUpper(loaded.Format) != "TUNA" {
		return Manifest{}, fmt.Errorf("in header: 'format' key must exist and be set to 'TUNA'")
	}

	if strings.ToUpper(loaded.Type) != "MANIFEST" {
		return Manifest{}, fmt.Errorf("in header: type set to %q, not \"MANIFEST\"", loaded.Type)
	}

	manif := Manifest{
		Files: loaded.Files,
	}

	return manif, nil
}

// ParseWorldFromTOML takes in raw TOML bytes, reads it for a world definition,
// and returns the rooms as well as the label of the starting room.
//
// Note: Uses module-global variables as part of operation. Absolutely not
// thread-safe and calling more than once concurrently will lead to unexpected
// results.
func ParseWorldDataFromTOML(tomlData []byte) (WorldData, error) {
	translatedData, err := UnmarshalTOMLWorldData(tomlData)
	if err != nil {
		return WorldData{}, err
	}
	return parseUnmarshaledData(translatedData)
}

// UnmarshalTOMLWorldData unmarshals but does not parse or check the loaded
// world data.
func UnmarshalTOMLWorldData(tomlData []byte) (tqwWorldData, error) {
	var tqw tqwWorldData
	if tomlErr := toml.Unmarshal(tomlData, &tqw); tomlErr != nil {
		return tqw, tomlErr
	}

	if strings.ToUpper(tqw.Format) != "TUNA" {
		return tqw, fmt.Errorf("in header: 'format' key must exist and be set to 'TUNA'")
	}
	if strings.ToUpper(tqw.Type) != "DATA" {
		return tqw, fmt.Errorf("in header: 'type' must exist and be set to 'DATA'")
	}

	return tqw, nil
}

// scan the first lines for format info before doing anything else
func scanFileInfo(data []byte) (tqwFileInfo, error) {
	// only run the toml parser up to the end of the top-lev table

	var topLevelEnd int = -1
	var onNewLine bool
	for b := range data {
		if onNewLine {
			if data[b] == '[' {
				topLevelEnd = b
				break
			}
		}

		if data[b] == '\n' {
			onNewLine = true
		} else {
			onNewLine = false
		}
	}

	scanData := data
	if topLevelEnd != -1 {
		scanData = data[:topLevelEnd]
	}

	var info tqwFileInfo
	err := toml.Unmarshal(scanData, &info)
	return info, err
}

func parseUnmarshaledData(tqw tqwWorldData) (WorldData, error) {
	var err error

	var world WorldData
	world.Start = tqw.World.Start
	world.Rooms = make(map[string]*Room)

	for idx, r := range tqw.Rooms {
		if roomErr := validateRoomDef(r); roomErr != nil {
			return world, fmt.Errorf("parsing: rooms[%d (%q)]: %w", idx, r.Label, roomErr)
		}

		if _, ok := world.Rooms[r.Label]; ok {
			return world, fmt.Errorf("parsing: rooms[%d (%q)]: room label %q has already been used", idx, r.Label, r.Label)
		}

		room := r.toRoom()
		world.Rooms[r.Label] = &room
	}

	pronouns := map[string]tqwPronounSet{
		"SHE/HER":   pronounSetToTWQ(PronounsFeminine),
		"HE/HIM":    pronounSetToTWQ(PronounsMasculine),
		"THEY/THEM": pronounSetToTWQ(PronounsNonBinary),
		"IT/ITS":    pronounSetToTWQ(PronounsItIts),
	}

	// check loaded pronouns
	for name, ps := range tqw.Pronouns {
		if err := validatePronounSetDef(ps, "", nil); err != nil {
			return world, fmt.Errorf("parsing: pronouns[%s]: %w", name, err)
		}

		if _, ok := pronouns[name]; ok {
			return world, fmt.Errorf("parsing: pronouns[%s]: duplicate pronoun name %q", name, name)
		}

		pronouns[name] = ps
	}

	npcs := make([]NPC, 0)
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
			empty := tqwPronounSet{}
			if npc.PronounSet != empty {
				return world, fmt.Errorf("parsing: npcs[%d (%q)]: can't define custom pronoun set because pronouns is set to %q", idx, npc.Label, npc.Pronouns)
			}
			npc.PronounSet = pronouns[strings.ToUpper(npc.Pronouns)]
		}

		npcs = append(npcs, npc.toNPC())
	}

	// now that they are all loaded and individually checked for validity,
	// ensure that all room egresses are valid existing labels
	for roomIdx, r := range tqw.Rooms {
		for egressIdx, eg := range r.Exits {
			if _, ok := world.Rooms[eg.DestLabel]; !ok {
				errMsg := "validating: rooms[%d (%q)]: exits[%d]: no room with label %q exists"
				return world, fmt.Errorf(errMsg, roomIdx, r.Label, egressIdx, eg.DestLabel)
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
	pf := Pathfinder{World: world.Rooms}
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
		case RoutePatrol:
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
		case RouteWander:
			for roomIdx, roomLabel := range npc.Movement.AllowedRooms {
				_, ok := world.Rooms[roomLabel]
				if !ok {
					errMsg := "validating: npcs[%d (%q)]: movement: allowedRooms[%d]: no room with label %q exists"
					return world, fmt.Errorf(errMsg, idx, npc.Label, roomIdx, roomLabel)
				}
			}

			for roomIdx, roomLabel := range npc.Movement.ForbiddenRooms {
				_, ok := world.Rooms[roomLabel]
				if !ok {
					errMsg := "validating: npcs[%d (%q)]: movement: forbiddenRooms[%d]: no room with label %q exists"
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
						errMsg := "validating: npcs[%d (%q)]: movement: allowedRooms[%d]: %q is not reachable from start"
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
							errMsg := "validating: npcs[%d (%q)]: movement: forbiddenRooms[%d]: %q is not reachable from start"
							return world, fmt.Errorf(errMsg, idx, npc.Label, fRoomIdx, fRoom)
						}
					}
				}
			}
		case RouteStatic:
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
			if diaStep.Action == DialogChoice {
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

func validateNPCDef(npc twqNPC, topLevelPronouns map[string]tqwPronounSet) error {
	if npc.Label == "" {
		return fmt.Errorf("must have non-blank 'label' field")
	}
	if npc.Name == "" {
		return fmt.Errorf("must have non-blank 'name' field")
	}

	// check pronouns are set or refer to one
	err := validatePronounSetDef(npc.PronounSet, npc.Pronouns, topLevelPronouns)
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

func validateDialogStepDef(ds tqwDialogStep) error {
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

func validateRouteDef(ps tqwRoute) error {
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
		if len(ps.Path) > 0 {
			return fmt.Errorf("'STATIC' route type does not use 'path' property")
		}
		if len(ps.AllowedRooms) > 0 {
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
func validatePronounSetDef(ps tqwPronounSet, label string, topLevel map[string]tqwPronounSet) error {
	if label != "" {
		if topLevel == nil {
			return fmt.Errorf("top-level pronoun must be full pronoun definition, not a label (%q)", label)
		}
		if _, ok := topLevel[strings.ToUpper(label)]; !ok {
			return fmt.Errorf("no pronoun set called %q exists", label)
		}
	}
	return nil
}

func validateRoomDef(r tqwRoom) error {
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

func validateEgressDef(eg tqwEgress) error {
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

func validateItemDef(item tqwItem) error {
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
