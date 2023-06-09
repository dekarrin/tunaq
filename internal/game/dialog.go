package game

import (
	"fmt"
	"strings"

	"github.com/dekarrin/rosed"
	"github.com/dekarrin/tunaq/tunascript"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type DialogAction int

const (
	DialogEnd DialogAction = iota
	DialogLine
	DialogChoice
	DialogPause
)

func (dst DialogAction) String() string {
	switch dst {
	case DialogEnd:
		return "END"
	case DialogLine:
		return "LINE"
	case DialogChoice:
		return "CHOICE"
	case DialogPause:
		return "PAUSE"
	default:
		return fmt.Sprintf("DialogAction(%d)", int(dst))
	}
}

// DialogActionsByString is a map indexing string values to their corresponding
// DialogAction.
var DialogActionsByString map[string]DialogAction = map[string]DialogAction{
	DialogEnd.String():    DialogEnd,
	DialogLine.String():   DialogLine,
	DialogChoice.String(): DialogChoice,
	DialogPause.String():  DialogPause,
}

// DialogStep is a single step of a Dialog tree. It instructions a Dialog as to
// what should happen in it and how to do it. A step either specifies an end,
// a line, or a choice.
type DialogStep struct {
	Action DialogAction

	// The line of dialog. Not used if action is END.
	Content string

	// How the player responds to the line. Only used if action is LINE, else
	// use choices.
	Response string

	// Choices is a series of options for a CHOICE type dialog step.
	Choices [][2]string

	// Label of the DialogStep. If not set it will just be the index within the
	// conversation tree.
	Label string

	// Resume is the label to resume the conversation with after coming back
	// after a PAUSE step. It is only relevant if Action is DialogPause. If
	// blank on a PAUSE step, it will resume with the next step in the tree.
	ResumeAt string

	// tsResponse is the pre-computed tunascript expansion tree for the
	// Response. It can only be created by a Tunascript engine and will not be
	// present on initial load of DialogStep from disk.
	tsResponse *tunascript.ExpansionAST

	// tsContent is the pre-computed tunascript expansion tree for the Content.
	// It can only be created by a Tunascript engine and will not be present on
	// initial load of DialogStep from disk.
	tsContent *tunascript.ExpansionAST

	// tsChoices is the pre-computed tunascript expansion trees for the player
	// response parts of Choices. It can only be created by a Tunascript engine
	// and will not be present on initial load of DialogStep from disk.
	tsChoices []*tunascript.ExpansionAST
}

// Copy returns a deeply-copied DialogStep.
func (ds DialogStep) Copy() DialogStep {
	dsCopy := DialogStep{
		Action:   ds.Action,
		Label:    ds.Label,
		Response: ds.Response,
		Choices:  make([][2]string, len(ds.Choices)),
		Content:  ds.Content,
		ResumeAt: ds.ResumeAt,
	}

	for idx, v := range ds.Choices {
		dsCopy.Choices[idx] = [2]string{v[0], v[1]}
	}

	return dsCopy
}

func (ds DialogStep) String() string {
	str := fmt.Sprintf("DialogStep<%q", ds.Action)

	switch ds.Action {
	case DialogEnd:
		return str + ">"
	case DialogLine:
		if ds.Label != "" {
			str += fmt.Sprintf(" (%q)", ds.Label)
		}
		return str + fmt.Sprintf(" %q>", ds.Content)
	case DialogChoice:
		str += " ("
		choiceCount := len(ds.Choices)
		gotChoices := 0
		for text, dest := range ds.Choices {
			str += fmt.Sprintf("%q->%q", text, dest)
			gotChoices++
			if gotChoices+1 < choiceCount {
				str += ", "
			}
		}
		str += ")>"
		return str
	case DialogPause:
		if ds.ResumeAt != "" {
			str += fmt.Sprintf(" (RESUME: %q)", ds.ResumeAt)
		}
		str += ">"
		return str
	default:
		return str + " (UNKNOWN TYPE)>"
	}
}

// Conversation includes the dialog tree and the current position within it. It
// can be created simply by manually creating a Conversation and assigning a
// sequence of steps to Dialog.
type Conversation struct {
	Dialog  []DialogStep
	cur     int
	aliases map[string]int
}

// NextStep gets the next DialogStep in the conversation. If it returns a
// DialogStep with an Action of DialogEnd, the dialog is over and there is
// nothing further to do. If it returns a DialogStep with an Action of
// DialogLine, NextStep can be safely called to go to the next DialogStep after
// that one is processed. If it returns a DialogStep with an Action of
// DialogChoice, JumpTo should be used to jump to the given alias before calling
// NextStep again.
func (convo *Conversation) NextStep() DialogStep {
	if convo.cur >= len(convo.Dialog) {
		return DialogStep{Action: DialogEnd}
	}

	current := convo.Dialog[convo.cur]
	convo.cur++
	return current
}

// JumpTo makes the Conversation JumpTo the step with the given label, so that
// the next call to NextStep returns that DialogStep. If an invalid label is
// given, the next call to NextStep will return an END step. This is a valid
// option for moving the convo to the appropriate position after the user has
// entered a choice.
func (convo *Conversation) JumpTo(label string) {
	if convo.aliases == nil {
		convo.buildAliases()
	}

	pos, ok := convo.aliases[label]

	if !ok {
		convo.buildAliases()
		pos, ok = convo.aliases[label]
	}

	convo.cur = len(convo.Dialog)

	if ok {
		convo.cur = pos
	}
}

func (convo *Conversation) buildAliases() {
	convo.aliases = make(map[string]int)
	for i := range convo.Dialog {
		if convo.Dialog[i].Label == "" {
			convo.Dialog[i].Label = fmt.Sprintf("%d", i)
		}

		convo.aliases[convo.Dialog[i].Label] = i
	}
}

func (gs *State) RunConversation(npc *NPC) error {
	// in case we need to end convo early, so we can reset it
	oldConvoCur := npc.Convo.cur

	var output string
	if len(npc.Dialog) < 1 {
		nomPro := cases.Title(language.AmericanEnglish).String(npc.Pronouns.Nominative)
		doesNot := "doesn't"
		if npc.Pronouns.Plural {
			doesNot = "don't"
		}
		if err := gs.io.Output("\n%s %s have much to say.\n", nomPro, doesNot); err != nil {
			return err
		}
		return nil
	} else {
		nomObj := strings.ToLower(npc.Pronouns.Objective)

		gs.io.Output("\nYou start talking to %s.\n", nomObj)

		for {
			step := npc.Convo.NextStep()
			switch step.Action {
			case DialogEnd:
				npc.Convo = nil
				return nil
			case DialogLine:
				line := gs.Expand(step.tsContent, fmt.Sprintf("CONTENT FOR NPC %q DIALOG LINE %q", npc.Label, step.Label))
				ed := rosed.Edit("\n"+strings.ToUpper(npc.Name)+":\n").WithOptions(textFormatOptions).
					CharsFrom(rosed.End).
					Insert(rosed.End, "\""+strings.TrimSpace(line)+"\"").
					Wrap(gs.io.Width).
					Insert(rosed.End, "\n")
				if step.Response != "" {
					resp := gs.Expand(step.tsResponse, fmt.Sprintf("RESPONSE FOR NPC %q DIALOG LINE %q", npc.Label, step.Label))
					ed = ed.
						Insert(rosed.End, "\nYOU:\n").
						Insert(rosed.End, rosed.Edit("\""+strings.TrimSpace(resp)+"\"").Wrap(gs.io.Width).String()).
						Insert(rosed.End, "\n")
				}
				output = ed.String()

				if err := gs.io.Output(output); err != nil {
					return err
				}

				stopCmd, err := gs.io.Input("(Enter to continue, 'STOP' to end) ==>")
				if err != nil {
					return err
				}
				stopCmdUpper := strings.ToUpper(strings.TrimSpace(stopCmd))
				if stopCmdUpper == "STOP" || stopCmdUpper == "QUIT" || stopCmdUpper == "LEAVE" {
					npc.Convo.cur = oldConvoCur
					return nil
				}
			case DialogChoice:
				line := gs.Expand(step.tsContent, fmt.Sprintf("CONTENT FOR NPC %q DIALOG LINE %q", npc.Label, step.Label))
				ed := rosed.Edit("\n"+strings.ToUpper(npc.Name)+":\n").WithOptions(textFormatOptions).
					CharsFrom(rosed.End).
					Insert(rosed.End, "\""+strings.TrimSpace(line)+"\"").
					Wrap(gs.io.Width).
					Insert(rosed.End, "\n\n").
					CharsFrom(rosed.End)

				var choiceOut = make([]string, len(step.Choices))
				for idx := range step.Choices {
					tsCh := step.tsChoices[idx]
					chDest := step.Choices[idx][1]
					choiceOut[idx] = chDest

					ch := gs.Expand(tsCh, fmt.Sprintf("CHOICE FOR NPC %q DIALOG LINE %q CHOICE #%d", npc.Label, step.Label, idx+1))

					ed = ed.Insert(rosed.End, fmt.Sprintf("%d) \"%s\"\n", idx+1, strings.TrimSpace(ch)))
				}
				ed = ed.Apply(func(idx int, line string) []string {
					return []string{rosed.Edit(line).Wrap(gs.io.Width).String()}
				})

				var err error

				err = gs.io.Output(ed.String())
				if err != nil {
					return err
				}

				var validNum bool
				var choiceNum int
				for !validNum {
					choiceNum, err = gs.io.InputInt("==> ")
					if err != nil {
						return err
					} else {
						if choiceNum < 1 || len(step.Choices) < choiceNum {
							err = gs.io.Output("Please enter a number between 1 and %d\n", len(step.Choices))
							if err != nil {
								return err
							}
						} else {
							validNum = true
						}
					}
				}

				dest := choiceOut[choiceNum-1]
				npc.Convo.JumpTo(dest)
			case DialogPause:
				// if resumeAt is not set, no jump needs to be made, convo will
				// resume with NextStep.
				if step.ResumeAt != "" {
					npc.Convo.JumpTo(step.ResumeAt)
				}
				return nil
			default:
				// should never happen
				panic("unknown line type")
			}
		}
	}
}
