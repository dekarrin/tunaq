package game

import (
	"fmt"
	"math"
)

type pathCache map[[2]string][]string

// Pather performs pathfinding operations on the given world. It also retains
// already-completed lookups so that it becomes faster with use. Changing World
// does not invalidate the cache, therefore World should not be set to anything
// else once it is initially set and the first operation is called on the
// Pathfinder.
type Pathfinder struct {
	World         map[string]*Room
	dijkstraTable pathCache
}

// ValidateRoute verifies whether the given route is a sequence of labels of
// rooms in the world that can be visited in the order given. If it is not a
// valid route, a non-nil error is returned containing info on why the route is
// not possible.
func (pf *Pathfinder) ValidatePath(path []string, loop bool) error {
	lastRoom, ok := pf.World[path[0]]
	if !ok {
		return fmt.Errorf("path[0]: no room with label %q exists", path[0])
	}

	for i := 1; i < len(path); i++ {
		nextRoom, ok := pf.World[path[i]]
		if !ok {
			return fmt.Errorf("path[%d]: no room with label %q exists", i, path[i])
		}

		lastRoom = pf.World[path[i-1]]

		reachable := false
		for _, egress := range lastRoom.Exits {
			if egress.DestLabel == nextRoom.Label {
				reachable = true
				break
			}
		}
		if !reachable {
			return fmt.Errorf("path[%d]: room %q does not have an exit to %q", i, lastRoom.Label, nextRoom.Label)
		}

		lastRoom = nextRoom
	}

	// check loop around if we are checking loop
	if loop {
		nextRoom := pf.World[path[0]]

		reachable := false
		for _, egress := range lastRoom.Exits {
			if egress.DestLabel == nextRoom.Label {
				reachable = true
				break
			}
		}
		if !reachable {
			return fmt.Errorf("path[%d]: room %q does not have an exit to %q", len(path)-1, lastRoom.Label, nextRoom.Label)
		}
	}

	return nil
}

// Dijkstra uses Dijkstra's Algorithm to find the shortest path from one
// node to another in the world. This only checks if it is ever possible, not if
// it is *currently* possible to traverse with no further actions. Callers
// should verify that the returned sequence of rooms is traversable before
// attempting to use it for such purposes.
//
// Returns nil or empty []string if the startLabel does not exist in the world,
// if the endLabel does not exist in the world, or if the path is not possible.
func (pf *Pathfinder) Dijkstra(startLabel, endLabel string) []string {
	if pf.dijkstraTable != nil {
		if solution, ok := pf.dijkstraTable[[2]string{startLabel, endLabel}]; ok {
			solCopy := make([]string, len(solution))
			copy(solCopy, solution)
			return solCopy
		}
	}

	source := pf.World[startLabel]
	target := pf.World[endLabel]

	if source == nil || target == nil || startLabel == endLabel {
		if pf.dijkstraTable == nil {
			pf.dijkstraTable = pathCache{}
		}

		pf.dijkstraTable[[2]string{startLabel, endLabel}] = make([]string, 0)
		return []string{}
	}

	dist := map[string]uint{}
	prev := map[string]*Room{}
	searchSetQ := map[string]*Room{}

	for _, room := range pf.World {
		dist[room.Label] = math.MaxUint
		prev[room.Label] = nil
		searchSetQ[room.Label] = room
	}
	dist[source.Label] = 0

	for len(searchSetQ) > 0 {
		var minDist uint = math.MaxUint
		uLabel := ""
		for label, d := range dist {
			if d <= minDist {
				uLabel = label
				minDist = d
			}
		}

		if uLabel == target.Label {
			break
		}
		u := searchSetQ[uLabel]
		delete(searchSetQ, uLabel)

		for _, vEgress := range u.Exits {
			vLabel := vEgress.DestLabel
			v, vOK := searchSetQ[vLabel]
			if !vOK {
				continue
			}

			const costUToV = 1 // every room movement has edge length of 1 in world graph

			// check if adding to our maxUint; if so dont add it, we are treating it as infinity so just leave it as is
			alt := dist[u.Label]
			if alt < math.MaxUint {
				alt += costUToV
			}

			if alt < dist[v.Label] {
				dist[v.Label] = alt
				prev[v.Label] = u
			}
		}
	}

	solution := []string{}
	if prev[target.Label] != nil { // only do this if target is reachable
		u := target
		for u != nil {
			solution = append(solution, u.Label)
			u = prev[u.Label]
		}
	}

	if pf.dijkstraTable == nil {
		pf.dijkstraTable = pathCache{}
	}
	pf.dijkstraTable[[2]string{startLabel, endLabel}] = make([]string, len(solution))
	copy(pf.dijkstraTable[[2]string{startLabel, endLabel}], solution)

	return solution
}
