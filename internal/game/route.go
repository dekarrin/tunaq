package game

import "fmt"

// RouteAction is the type of action that a route has an NPC take.
type RouteAction int

const (
	RouteStatic RouteAction = iota
	RoutePatrol
	RouteWander
)

func (ra RouteAction) String() string {
	switch ra {
	case RouteStatic:
		return "STATIC"
	case RoutePatrol:
		return "PATROL"
	case RouteWander:
		return "WANDER"
	default:
		return fmt.Sprintf("RouteAction(%d)", int(ra))
	}
}

// RouteActionsByString is a map indexing string values to their corresponding
// RouteAction.
var RouteActionsByString map[string]RouteAction = map[string]RouteAction{
	RouteStatic.String(): RouteStatic,
	RoutePatrol.String(): RoutePatrol,
	RouteWander.String(): RouteWander,
}

// Route is a type of movement for an NPC to take
type Route struct {
	// Action is the type of action the route has the NPC move. RouteStatic is
	// not moving, RoutePatrol is follow the steps in 'Patrol', RouteWander is
	// to wander about but stay within AllowedRooms (if defined) or out of
	// ForbiddenRooms (if defined).
	Action RouteAction

	// Path is the steps that the route takes, by their room labels. It is
	// only used if Action is set to RoutePatrol
	Path []string

	// AllowedRooms is the list of rooms by their label that wandering movement
	// will stay within. It is only used if Action is set to RouteWander. If
	// neither this nor ForbiddenRooms has entries, the NPC is permitted to
	// wander anywhere. If both are set and contain the same entry,
	// ForbiddenRooms takes precedent and the room will be forbidden.
	AllowedRooms []string

	// ForbiddenRooms is the list of rooms by their label that wandering
	// movement will stay out of. It is only used if Action is set to
	// RouteWander. If neither this nor AllowedRooms has entries, the NPC is
	// permitted to wander anywhere. If both are set and contain the same entry,
	// ForbiddenRooms takes precedent and the room will be forbidden.
	ForbiddenRooms []string
}

// Copy returns a deeply-copied Route.
func (r Route) Copy() Route {
	rCopy := Route{
		Action:         r.Action,
		Path:           make([]string, len(r.Path)),
		AllowedRooms:   make([]string, len(r.AllowedRooms)),
		ForbiddenRooms: make([]string, len(r.ForbiddenRooms)),
	}

	copy(rCopy.Path, r.Path)
	copy(rCopy.AllowedRooms, r.AllowedRooms)
	copy(rCopy.ForbiddenRooms, r.ForbiddenRooms)

	return rCopy
}

func (r Route) String() string {
	str := fmt.Sprintf("Route<%q", r.Action)

	switch r.Action {
	case RouteStatic:
		return str + ">"
	case RoutePatrol:
		str += " path=["
		for idx, p := range r.Path {
			str += fmt.Sprintf("%q", p)
			if idx+1 < len(r.Path) {
				str += ", "
			}
		}
		str += "]>"
		return str
	case RouteWander:
		str += " allowed=["
		for idx, ar := range r.AllowedRooms {
			str += fmt.Sprintf("%q", ar)
			if idx+1 < len(r.AllowedRooms) {
				str += ", "
			}
		}
		str += "], forbidden=["
		for idx, fr := range r.ForbiddenRooms {
			str += fmt.Sprintf("%q", fr)
			if idx+1 < len(r.ForbiddenRooms) {
				str += ", "
			}
		}
		str += "]>"
		return str
	default:
		return str + " (UNKNOWN TYPE)>"
	}
}
