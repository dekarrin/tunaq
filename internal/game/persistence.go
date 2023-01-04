package game

import (
	"fmt"
	"os"
	"strings"
)

// LoadWorldDefFile loads a world from a world definition
func LoadWorldDefFile(path string) (world map[string]*Room, startRoom string, err error) {
	worldData, loadErr := os.ReadFile(path)
	if loadErr != nil {
		return nil, "", fmt.Errorf("reading world file: %w", loadErr)
	}

	if strings.HasSuffix(strings.ToUpper(path), ".JSON") {
		world, startRoom, err = ParseWorldFromJSON(worldData)
		if err != nil {
			return nil, "", fmt.Errorf("loading world file: %w", err)
		}
	} else {
		world, startRoom, err = ParseWorldFromTOML(worldData)
		if err != nil {
			return nil, "", fmt.Errorf("loading world file: %w", err)
		}
	}

	return world, startRoom, nil
}
