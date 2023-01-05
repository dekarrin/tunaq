package tqw

import (
	"strings"

	"github.com/dekarrin/tunaq/internal/game"
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

func (tn twqNPC) toNPC() game.NPC {
	npc := game.NPC{
		Label:       strings.ToUpper(tn.Label),
		Name:        tn.Name,
		Pronouns:    tn.PronounSet.toPronounSet(),
		Description: tn.Description,
		Start:       tn.Start,
		Movement:    tn.Movement.toRoute(),
		Dialog:      make([]game.DialogStep, len(tn.Dialog)),
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

func (tr tqwRoute) toRoute() game.Route {
	act, ok := game.RouteActionsByString[strings.ToUpper(tr.Action)]
	if !ok {
		act = game.RouteStatic
	}

	r := game.Route{
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

func (tds tqwDialogStep) toDialogStep() game.DialogStep {
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

type tqwPronounSet struct {
	Nominative string `toml:"nominative"`
	Objective  string `toml:"objective"`
	Possessive string `toml:"possessive"`
	Determiner string `toml:"determiner"`
	Reflexive  string `toml:"reflexive"`
}

func pronounSetToTWQ(ps game.PronounSet) tqwPronounSet {
	tp := tqwPronounSet(ps)

	return tp
}

func (tp tqwPronounSet) toPronounSet() game.PronounSet {
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

type tqwItem struct {
	Label       string   `toml:"label"`
	Name        string   `toml:"name"`
	Description string   `toml:"description"`
	Aliases     []string `toml:"aliases"`
}

func (ti tqwItem) toItem() game.Item {
	item := game.Item{
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

func (te tqwEgress) toEgress() game.Egress {
	eg := game.Egress{
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

func (tr tqwRoom) toRoom() game.Room {
	r := game.Room{
		Label:       tr.Label,
		Name:        tr.Name,
		Description: tr.Description,
		Exits:       make([]game.Egress, len(tr.Exits)),
		Items:       make([]game.Item, len(tr.Items)),
		NPCs:        make(map[string]*game.NPC),
	}

	for i := range tr.Exits {
		r.Exits[i] = tr.Exits[i].toEgress()
	}
	for i := range tr.Items {
		r.Items[i] = tr.Items[i].toItem()
	}

	return r
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
