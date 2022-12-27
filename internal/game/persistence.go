package game

import (
	"fmt"
	"os"
)

// LoadWorldDefFile loads a world from a world definition
func LoadWorldDefFile(path string) (world map[string]*Room, startRoom string, npcs []NPC, err error) {
	jsonData, loadErr := os.ReadFile(path)
	if loadErr != nil {
		return nil, "", nil, fmt.Errorf("reading world file: %w", loadErr)
	}

	world, startRoom, npcs, err = ParseWorldFromJSON(jsonData)
	if err != nil {
		return nil, "", nil, fmt.Errorf("loading world file: %w", err)
	}

	return world, startRoom, npcs, nil
}
