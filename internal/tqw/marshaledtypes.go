package tqw

import (
	"strings"

	"github.com/dekarrin/tunaq/internal/game"
)

type topLevelManifest struct {
	Format string   `toml:"format"`
	Type   string   `toml:"type"`
	Files  []string `toml:"files"`
}

// topLevelWorldData is the top-level structure containing all keys in a complete TQW
// 'DATA' type file.
type topLevelWorldData struct {
	Format   string                `toml:"format"`
	Type     string                `toml:"type"`
	Rooms    []room                `toml:"rooms"`
	World    world                 `toml:"world"`
	NPCs     []npc                 `toml:"npcs"`
	Pronouns map[string]pronounSet `toml:"pronouns"`
}

type npc struct {
	Label       string       `toml:"label"`
	Name        string       `toml:"name"`
	Pronouns    string       `toml:"pronouns"`
	PronounSet  pronounSet   `toml:"custom_pronoun_set"`
	Description string       `toml:"description"`
	Start       string       `toml:"start"`
	Movement    route        `toml:"movement"`
	Dialog      []dialogStep `toml:"dialog"`
}

func (tn npc) toGameNPC() game.NPC {
	npc := game.NPC{
		Label:       strings.ToUpper(tn.Label),
		Name:        tn.Name,
		Pronouns:    tn.PronounSet.toGamePronounSet(),
		Description: tn.Description,
		Start:       tn.Start,
		Movement:    tn.Movement.toGameRoute(),
		Dialog:      make([]game.DialogStep, len(tn.Dialog)),
	}

	for i := range tn.Dialog {
		npc.Dialog[i] = tn.Dialog[i].toGameDialogStep()
	}

	return npc
}

type route struct {
	Action    string   `toml:"action"`
	Path      []string `toml:"path"`
	Forbidden []string `toml:"forbidden"`
	Allowed   []string `toml:"allowed"`
}

func (tr route) toGameRoute() game.Route {
	act, ok := game.RouteActionsByString[strings.ToUpper(tr.Action)]
	if !ok {
		act = game.RouteStatic
	}

	r := game.Route{
		Action:         act,
		Path:           make([]string, len(tr.Path)),
		ForbiddenRooms: make([]string, len(tr.Forbidden)),
		AllowedRooms:   make([]string, len(tr.Allowed)),
	}

	copy(r.Path, tr.Path)
	copy(r.ForbiddenRooms, tr.Forbidden)
	copy(r.AllowedRooms, tr.Allowed)

	return r
}

type dialogStep struct {
	Action   string     `toml:"action"`
	Label    string     `toml:"label"`
	Content  string     `toml:"content"`
	Response string     `toml:"response"`
	Choices  [][]string `toml:"choices"`
}

func (tds dialogStep) toGameDialogStep() game.DialogStep {
	act, ok := game.DialogActionsByString[strings.ToUpper(tds.Action)]
	if !ok {
		act = game.DialogLine
	}

	ds := game.DialogStep{
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

type pronounSet struct {
	Nominative string `toml:"nominative"`
	Objective  string `toml:"objective"`
	Possessive string `toml:"possessive"`
	Determiner string `toml:"determiner"`
	Reflexive  string `toml:"reflexive"`
}

func (tp pronounSet) toGamePronounSet() game.PronounSet {
	ps := game.PronounSet{
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

type item struct {
	Label       string   `toml:"label"`
	Name        string   `toml:"name"`
	Description string   `toml:"description"`
	Aliases     []string `toml:"aliases"`
}

func (ti item) toGameItem() game.Item {
	gameItem := game.Item{
		Label:       ti.Label,
		Name:        ti.Name,
		Description: ti.Description,
		Aliases:     make([]string, len(ti.Aliases)),
	}

	copy(gameItem.Aliases, ti.Aliases)

	return gameItem
}

type egress struct {
	Dest        string   `toml:"dest"`
	Description string   `toml:"description"`
	Message     string   `toml:"message"`
	Aliases     []string `toml:"aliases"`
}

func (te egress) toGameEgress() game.Egress {
	eg := game.Egress{
		DestLabel:     te.Dest,
		Description:   te.Description,
		TravelMessage: te.Message,
		Aliases:       make([]string, len(te.Aliases)),
	}

	copy(eg.Aliases, te.Aliases)

	return eg
}

type room struct {
	Label       string   `toml:"label"`
	Name        string   `toml:"name"`
	Description string   `toml:"description"`
	Exits       []egress `toml:"exits"`
	Items       []item   `toml:"items"`
}

func (tr room) toGameRoom() game.Room {
	r := game.Room{
		Label:       tr.Label,
		Name:        tr.Name,
		Description: tr.Description,
		Exits:       make([]game.Egress, len(tr.Exits)),
		Items:       make([]game.Item, len(tr.Items)),
		NPCs:        make(map[string]*game.NPC),
	}

	for i := range tr.Exits {
		r.Exits[i] = tr.Exits[i].toGameEgress()
	}
	for i := range tr.Items {
		r.Items[i] = tr.Items[i].toGameItem()
	}

	return r
}

type world struct {
	Start string `toml:"start"`
}
