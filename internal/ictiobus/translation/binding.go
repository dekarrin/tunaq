package translation

import "fmt"

type SDDBinding struct {
	// Synthesized is whether the binding is for a
	Synthesized bool

	// BoundRuleSymbol is the head symbol of the rule the binding is on.
	BoundRuleSymbol string

	// BoundRuleProduction is the list of produced symbols of the rule the
	// binding is on.
	BoundRuleProduction []string

	// Requirements is the attribute references that this binding needs to
	// compute its value. Values corresponding to the references are passed in
	// to calls to Setter via its args slice in the order they are specified
	// here.
	Requirements []AttrRef

	// Dest is the destination.
	Dest AttrRef

	// Setter is the call to calculate a value of the node by the binding.
	Setter AttributeSetter
}

func (bind SDDBinding) Copy() SDDBinding {
	newBind := SDDBinding{
		Synthesized:         bind.Synthesized,
		BoundRuleSymbol:     bind.BoundRuleSymbol,
		BoundRuleProduction: make([]string, len(bind.BoundRuleProduction)),
		Requirements:        make([]AttrRef, len(bind.Requirements)),
		Dest:                bind.Dest,
		Setter:              bind.Setter,
	}

	copy(newBind.BoundRuleProduction, bind.BoundRuleProduction)
	copy(newBind.Requirements, bind.Requirements)

	return newBind
}

// Invoke calls the given binding while visiting an annotated parse tree node.
func (bind SDDBinding) Invoke(apt AnnotatedParseTree) NodeAttrValue {
	// sanity checks; can we even call this?
	if bind.Setter == nil {
		panic("attempt to invoke nil attribute setter func")
	}
	if bind.Dest.Relation.Type == RelHead && !bind.Synthesized {
		panic("cannot invoke inherited attribute SDD binding on head of rule")
	} else if bind.Dest.Relation.Type != RelHead && bind.Synthesized {
		panic("cannot invoke synthesized attribute SDD binding on production of rule")
	}

	// symbol of who it is for
	forSymbol, ok := apt.SymbolOf(bind.Dest.Relation)
	if !ok {
		// invalid dest
		panic(fmt.Sprintf("bound-to rule does not contain a %s", bind.Dest.Relation.String()))
	}

	// gather args
	args := []NodeAttrValue{}
	for i := range bind.Requirements {
		req := bind.Requirements[i]
		reqVal, ok := apt.AttributeValueOf(req)
		if !ok {
			// should never happen, creation of Binding should ensure this.
			_, refNodeExists := apt.AttributesOf(req.Relation)
			if !refNodeExists {
				// reference itself was invalid
				panic(fmt.Sprintf("bound-to rule does not contain a %s", req.Relation.String()))
			} else {
				panic(fmt.Sprintf("attribute %s not yet defined for %s in bound-to-rule", req.Name, req.Relation.String()))
			}
		}

		args = append(args, reqVal)
	}

	// call func
	val := bind.Setter(forSymbol, bind.Dest.Name, args)

	return val
}
