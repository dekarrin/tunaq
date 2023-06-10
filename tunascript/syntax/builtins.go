package syntax

// Function defines a function that can be executed in tunascript. It includes
// all info required to check usage of the function, but not the actual
// implementation.
type Function struct {
	// Name is the name of the function. It would be called with $Name(). Name
	// is case-insensitive and must follow identifier naming rules ([A-Z0-9_]+)
	Name string

	// RequiredArgs is the number of required arguments. This many arguments is
	// guaranteed to be passed to the implementation.
	RequiredArgs int

	// OptionalArgs is the number of optional arguments. Up to RequiredArgs +
	// OptionalArgs may be passed to the implementation.
	OptionalArgs int

	// SideEffects tells whether the function has side-effects. Certain contexts
	// such as within an $IF() may restrict the execution of side-effect
	// functions.
	SideEffects bool
}

var (
	// BuiltInFunctions has function definitions info for each function that is
	// built-in to TunaScript. It does *not* contain their implementations, only
	// info on them.
	BuiltInFunctions = map[string]Function{
		"ADD":               {Name: "ADD", RequiredArgs: 2},
		"SUB":               {Name: "SUB", RequiredArgs: 2},
		"MULT":              {Name: "MULT", RequiredArgs: 2},
		"DIV":               {Name: "DIV", RequiredArgs: 2},
		"NEG":               {Name: "NEG", RequiredArgs: 1},
		"OR":                {Name: "OR", RequiredArgs: 2},
		"AND":               {Name: "AND", RequiredArgs: 2},
		"NOT":               {Name: "NOT", RequiredArgs: 1},
		"FLAG_ENABLED":      {Name: "FLAG_ENABLED", RequiredArgs: 1},
		"FLAG_DISABLED":     {Name: "FLAG_DISABLED", RequiredArgs: 1},
		"FLAG_IS":           {Name: "FLAG_IS", RequiredArgs: 2},
		"FLAG_LESS_THAN":    {Name: "FLAG_LESS_THAN", RequiredArgs: 2},
		"FLAG_GREATER_THAN": {Name: "FLAG_GREATER_THAN", RequiredArgs: 2},
		"ENABLE":            {Name: "ENABLE", RequiredArgs: 1, SideEffects: true},
		"DISABLE":           {Name: "DISABLE", RequiredArgs: 1, SideEffects: true},
		"TOGGLE":            {Name: "TOGGLE", RequiredArgs: 1, SideEffects: true},
		"INC":               {Name: "INC", RequiredArgs: 1, OptionalArgs: 1, SideEffects: true},
		"DEC":               {Name: "DEC", RequiredArgs: 1, OptionalArgs: 1, SideEffects: true},
		"SET":               {Name: "SET", RequiredArgs: 2, SideEffects: true},
		"IN_INVEN":          {Name: "IN_INVEN", RequiredArgs: 1},
		"MOVE":              {Name: "MOVE", RequiredArgs: 2, SideEffects: true},
		"OUTPUT":            {Name: "OUTPUT", RequiredArgs: 1, SideEffects: true},
	}
)
