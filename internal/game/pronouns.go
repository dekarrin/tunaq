package game

// File pronouns.go has symbols related to pronoun definitions and retrievals.

import "fmt"

// PronounSet is a set of pronouns for an NPC that can be used by code that
// generates references to NPCs without being specific to a particular one. You
// can create your own Pronouns struct, or just stick to the previously-defined
// ones. These should be all upper-case.
type PronounSet struct {
	// Nominative is the pronoun used in the nominative case. 'SHE', 'HE', or
	// 'THEY' for example; the pronoun that would be used to replace 'NPC' in
	// the following sentence: "NPC WENT TO THE STORE."
	Nominative string

	// Objective is the pronoun used in the objective case. 'HER', 'HIM', or
	// 'THEM' for example; the pronoun that would be used to replace 'NPC' in
	// the following sentence: "YOU TALK TO NPC."
	Objective string

	// Possessive the pronoun used in the possesive case. 'HERS', 'HIS', or
	// 'THEIRS' for example; the pronoun that would be used to replace "NPC'S"
	// in the following sentence: "THAT ITEM IS NPC'S."
	Possessive string

	// Determiner is the pronoun used in the possesive case when using a pronoun
	// as an adjective of some noun to show ownership. 'HER', 'HIS', or 'THEIR'
	// for example; the pronoun that would be used to replace "NPC's" in the
	// following sentence: "THAT IS NPC'S ITEM."
	Determiner string

	// Reflexive is the pronoun used to show the pronoun-user refering back to
	// themself. 'HERSELF', 'HIMSELF', or 'THEMSELF' for example; the pronoun
	// that would be used to replace "NPCSELF" is the following sentence: "NPC
	// THINKS ABOUT NPCSELF A LOT".
	Reflexive string

	// Plural is whether the pronoun requires plural conjugation of verbs; e.g.
	// use 'NOMINATIVE are' instead of 'NOMINATIVE is'.
	Plural bool
}

func (ps PronounSet) String() string {
	conjug := "SINGULAR"
	if ps.Plural {
		conjug = "PLURAL"
	}
	return fmt.Sprintf("PronounSet<%q/%q/%q/%q/%q/%s>", ps.Nominative, ps.Objective, ps.Possessive, ps.Determiner, ps.Reflexive, conjug)
}

// Short returns the string representing the shortened version of the pronoun
// set. Only nominative, objective, and possessive are shown.
//
// If Nominative isn't set, the output will show it as "<?>". If Objective isn't
// set, the output will show it as the same as whatever Nominative is set to. If
// Possessive isn't set, the output will show it as the same as whatever
// Objective is set to but with an 's' added.
//
// If Possessive is just Objective but with 's' at the end, it will be omitted
// from the output.
func (ps PronounSet) Short() string {
	showPossessive := false

	if ps.Nominative == "" {
		ps.Nominative = "<?>"
	}
	if ps.Objective == "" {
		ps.Objective = ps.Nominative
	}
	if ps.Possessive == "" {
		showPossessive = false
	}

	if showPossessive {
		// don't show possessive if it is the same as objective but with S
		if ps.Possessive == ps.Objective+"S" || ps.Possessive == ps.Objective+"s" {
			showPossessive = false
		}
	}

	if showPossessive {
		return fmt.Sprintf("%s/%s/%s", ps.Nominative, ps.Objective, ps.Possessive)
	}
	return fmt.Sprintf("%s/%s", ps.Nominative, ps.Objective)
}

var (

	// PronounsFeminine is the predefined set of feminine pronouns, commonly
	// referred to as "she/her" pronouns.
	PronounsFeminine = PronounSet{"SHE", "HER", "HERS", "HER", "HERSELF", false}

	// PronounsMasculine is the predefined set of masculine pronouns, commonly
	// referred to as "he/him" pronouns.
	PronounsMasculine = PronounSet{"HE", "HIM", "HIS", "HIS", "HIMSELF", false}

	// PronounsNonBinary is the predefined set of non-binary pronouns, commonly
	// referred to as "they/them" pronouns.
	PronounsNonBinary = PronounSet{"THEY", "THEM", "THEIRS", "THEIR", "THEMSELF", true}

	// PronounsItIts is the predefined set of pronouns commonly referred to as
	// "it/its".
	PronounsItIts = PronounSet{"IT", "IT", "ITS", "ITS", "ITSELF", false}
)
