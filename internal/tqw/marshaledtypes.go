package tqw

import (
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/dekarrin/tunaq/internal/game"
)

type topLevelManifest struct {
	Format string   `toml:"format"`
	Type   string   `toml:"type"`
	Files  []string `toml:"files"`
}

type flag struct {
	Label       string         `toml:"label"`
	DefaultPrim toml.Primitive `toml:"default"`
	Default     string         // manually toml decode this one from Prim, either as string, int, or bool
}

// topLevelWorldData is the top-level structure containing all keys in a complete TQW
// 'DATA' type file.
type topLevelWorldData struct {
	Format   string       `toml:"format"`
	Type     string       `toml:"type"`
	Rooms    []room       `toml:"room"`
	World    world        `toml:"world"`
	NPCs     []npc        `toml:"npc"`
	Pronouns []pronounSet `toml:"pronouns"`
	Items    []item       `toml:"item"`
	Flags    []flag       `toml:"flag"`
}

type npc struct {
	Tags        []string     `toml:"tags"`
	Label       string       `toml:"label"`
	Aliases     []string     `toml:"aliases"`
	Name        string       `toml:"name"`
	Pronouns    string       `toml:"pronouns"`
	PronounSet  pronounSet   `toml:"custom_pronoun_set"`
	Description string       `toml:"description"`
	Start       string       `toml:"start"`
	Movement    route        `toml:"movement"`
	Dialogs     []dialogStep `toml:"line"`
	If          string       `toml:"if"`
}

func (tn npc) toGameNPC() game.NPC {
	npc := game.NPC{
		Label:       strings.ToUpper(tn.Label),
		Name:        tn.Name,
		Pronouns:    tn.PronounSet.toGamePronounSet(),
		Description: tn.Description,
		Start:       strings.ToUpper(tn.Start),
		Movement:    tn.Movement.toGameRoute(),
		Dialog:      make([]*game.DialogStep, len(tn.Dialogs)),
		Aliases:     make([]string, len(tn.Aliases)),
		Tags:        make([]string, len(tn.Tags)),
		IfRaw:       tn.If,
	}

	for i := range tn.Dialogs {
		npc.Dialog[i] = tn.Dialogs[i].toGameDialogStep()
	}
	for i := range tn.Aliases {
		npc.Aliases[i] = strings.ToUpper(tn.Aliases[i])
	}
	for i := range tn.Tags {
		tag := tn.Tags[i]
		if !strings.HasPrefix(tag, "@") {
			tag = "@" + tag
		}
		npc.Tags[i] = strings.ToUpper(tag)
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

	for i := range tr.Path {
		r.Path[i] = strings.ToUpper(tr.Path[i])
	}
	for i := range tr.Forbidden {
		r.ForbiddenRooms[i] = strings.ToUpper(tr.Forbidden[i])
	}
	for i := range tr.Allowed {
		r.AllowedRooms[i] = strings.ToUpper(tr.Allowed[i])
	}

	return r
}

type dialogStep struct {
	Action   string     `toml:"action"`
	Label    string     `toml:"label"`
	Content  string     `toml:"content"`
	Response string     `toml:"response"`
	Choices  [][]string `toml:"choices"`
	Continue string     `toml:"continue"`
}

func (tds dialogStep) toGameDialogStep() *game.DialogStep {
	act, ok := game.DialogActionsByString[strings.ToUpper(tds.Action)]
	if !ok {
		act = game.DialogLine
	}

	ds := game.DialogStep{
		Action:   act,
		Label:    strings.ToUpper(tds.Label),
		Content:  tds.Content,
		Response: tds.Response,
		Choices:  make([][2]string, len(tds.Choices)),
		ResumeAt: strings.ToUpper(tds.Continue),
	}

	for i := range tds.Choices {
		if len(tds.Choices[i]) < 2 {
			continue
		}

		choice := tds.Choices[i][0]
		dest := strings.ToUpper(tds.Choices[i][1])

		ds.Choices[i] = [2]string{choice, dest}
	}

	return &ds
}

type pronounSet struct {
	Label      string `toml:"label"`
	Nominative string `toml:"nominative"`
	Objective  string `toml:"objective"`
	Possessive string `toml:"possessive"`
	Determiner string `toml:"determiner"`
	Reflexive  string `toml:"reflexive"`
	Plural     bool   `toml:"plural"`
}

func pronounSetFromGame(gps game.PronounSet) pronounSet {
	ps := pronounSet{
		Nominative: strings.ToUpper(gps.Nominative),
		Objective:  strings.ToUpper(gps.Objective),
		Possessive: strings.ToUpper(gps.Possessive),
		Determiner: strings.ToUpper(gps.Determiner),
		Reflexive:  strings.ToUpper(gps.Reflexive),
		Plural:     gps.Plural,
	}

	return ps
}

func (tp pronounSet) toGamePronounSet() game.PronounSet {
	ps := game.PronounSet{
		Nominative: strings.ToUpper(tp.Nominative),
		Objective:  strings.ToUpper(tp.Objective),
		Possessive: strings.ToUpper(tp.Possessive),
		Determiner: strings.ToUpper(tp.Determiner),
		Reflexive:  strings.ToUpper(tp.Reflexive),
		Plural:     tp.Plural,
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

type useAction struct {
	With []string `toml:"with"`
	If   string   `toml:"if"`
	Do   []string `toml:"do"`
}

func (ua useAction) toGameUseAction() game.UseAction {
	gameAction := game.UseAction{
		With:  make([]string, len(ua.With)),
		IfRaw: ua.If,
		DoRaw: make([]string, len(ua.Do)),
	}

	for i := range ua.With {
		gameAction.With[i] = strings.ToUpper(ua.With[i])
	}

	copy(gameAction.DoRaw, ua.Do)

	return gameAction
}

type item struct {
	Tags        []string    `toml:"tags"`
	Label       string      `toml:"label"`
	Name        string      `toml:"name"`
	Description string      `toml:"description"`
	Aliases     []string    `toml:"aliases"`
	Start       string      `toml:"start"`
	If          string      `toml:"if"`
	OnUse       []useAction `toml:"on_use"`
}

func (ti item) toGameItem() game.Item {
	gameItem := game.Item{
		Label:       strings.ToUpper(ti.Label),
		Name:        ti.Name,
		Description: ti.Description,
		Aliases:     make([]string, len(ti.Aliases)),
		Tags:        make([]string, len(ti.Tags)),
		IfRaw:       ti.If,
		OnUse:       make([]game.UseAction, len(ti.OnUse)),
	}

	for i := range ti.Aliases {
		gameItem.Aliases[i] = strings.ToUpper(ti.Aliases[i])
	}
	for i := range ti.Tags {
		tag := ti.Tags[i]
		if !strings.HasPrefix(tag, "@") {
			tag = "@" + tag
		}
		gameItem.Tags[i] = strings.ToUpper(tag)
	}
	for i := range ti.OnUse {
		gameItem.OnUse[i] = ti.OnUse[i].toGameUseAction()
	}

	return gameItem
}

type egress struct {
	Label       string   `toml:"label"`
	Tags        []string `toml:"tags"`
	Dest        string   `toml:"dest"`
	Description string   `toml:"description"`
	Message     string   `toml:"message"`
	Aliases     []string `toml:"aliases"`
	If          string   `toml:"if"`
}

func (te egress) toGameEgress() game.Egress {
	eg := game.Egress{
		DestLabel:     strings.ToUpper(te.Dest),
		Description:   te.Description,
		TravelMessage: te.Message,
		Aliases:       make([]string, len(te.Aliases)),
		Tags:          make([]string, len(te.Tags)),
		IfRaw:         te.If,
	}

	for i := range te.Aliases {
		eg.Aliases[i] = strings.ToUpper(te.Aliases[i])
	}
	for i := range te.Tags {
		tag := te.Tags[i]
		if !strings.HasPrefix(tag, "@") {
			tag = "@" + tag
		}
		eg.Tags[i] = strings.ToUpper(tag)
	}

	return eg
}

type detail struct {
	Tags        []string `toml:"tags"`
	Label       string   `toml:"label"`
	Aliases     []string `toml:"aliases"`
	Description string   `toml:"description"`
	If          string   `toml:"if"`
}

func (td detail) toGameDetail() game.Detail {
	det := game.Detail{
		Label:       td.Label,
		Aliases:     make([]string, len(td.Aliases)),
		Tags:        make([]string, len(td.Tags)),
		Description: td.Description,
		IfRaw:       td.If,
	}

	for i := range det.Aliases {
		det.Aliases[i] = strings.ToUpper(td.Aliases[i])
	}
	for i := range det.Tags {
		tag := td.Tags[i]
		if !strings.HasPrefix(tag, "@") {
			tag = "@" + tag
		}
		det.Tags[i] = strings.ToUpper(tag)
	}

	return det
}

type room struct {
	Label       string   `toml:"label"`
	Name        string   `toml:"name"`
	Description string   `toml:"description"`
	Exits       []egress `toml:"exit"`
	Details     []detail `toml:"detail"`
}

func (tr room) toGameRoom() game.Room {
	r := game.Room{
		Label:       strings.ToUpper(tr.Label),
		Name:        tr.Name,
		Description: tr.Description,
		Exits:       make([]*game.Egress, len(tr.Exits)),
		NPCs:        make(map[string]*game.NPC),
		Details:     make([]*game.Detail, len(tr.Details)),
	}

	for i := range tr.Exits {
		eggCopy := tr.Exits[i].toGameEgress()
		r.Exits[i] = &eggCopy
	}

	for i := range tr.Details {
		detCopy := tr.Details[i].toGameDetail()
		r.Details[i] = &detCopy
	}

	return r
}

type world struct {
	Start string `toml:"start"`
}
