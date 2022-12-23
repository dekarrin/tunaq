package game

import (
	"encoding/json"
	"fmt"
)

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
	Rooms []jsonRoom `json:"rooms"`
	Start string     `json:"start"`
}

// ParseWorldFromJSON takes in raw json bytes, reads it for a world definition,
// and returns the rooms as well as the label of the starting room.
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

	// TODO: check that no item overwrites another

	// check that the start actually points to a real location
	if _, ok := world[loadedWorld.Start]; !ok {
		return nil, "", fmt.Errorf("validating: start: no room with label %q exists", startRoom)
	}

	return world, startRoom, nil
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
