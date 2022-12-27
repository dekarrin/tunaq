package game

import (
	"fmt"
	"os"
)

// LoadWorldDefFile loads a world from a world definition
func LoadWorldDefFile(path string) (world map[string]*Room, startRoom string, err error) {
	jsonData, loadErr := os.ReadFile(path)
	if loadErr != nil {
		return nil, "", fmt.Errorf("reading world file: %w", loadErr)
	}

	world, startRoom, err = ParseWorldFromJSON(jsonData)
	if err != nil {
		return nil, "", fmt.Errorf("loading world file: %w", err)
	}

	return world, startRoom, nil
}
