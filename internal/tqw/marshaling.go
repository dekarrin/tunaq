package tqw

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// manifStack is for two reasons ->
// * detect circular deps (not an error, but we need to know to avoid them)
// * avoid infinite recursion (allow up to MaxManifestRecursionDepth levels)
//
// Returnes ErrManifestEmpty if and only if the first manifest in the stack is
// empty, otherwise it is not an error.
func recursiveUnmarshalResource(path string, manifStack []string) (data topLevelWorldData, err error) {
	path = filepath.Clean(path)

	fileData, loadErr := os.ReadFile(path)
	if loadErr != nil {
		return topLevelWorldData{}, fmt.Errorf("%q: reading from disk: %w", path, loadErr)
	}

	fileInfo, err := ScanFileInfo(fileData)
	if err != nil {
		return topLevelWorldData{}, fmt.Errorf("%q: detecting file type: %w", path, err)
	}

	if strings.ToUpper(fileInfo.Format) != "TUNA" {
		return topLevelWorldData{}, fmt.Errorf("%q: file does not have a 'format = \"TUNA\" entry", path)
	}

	fileType := strings.ToUpper(fileInfo.Type)
	switch fileType {
	case "DATA":
		unmarshaled, err := unmarshalWorldData(fileData)
		if err != nil {
			return unmarshaled, fmt.Errorf("world data file %q: %w", path, err)
		}
		return unmarshaled, nil
	case "MANIFEST":
		// check the stack to be sure we havent recursed too far and to be sure
		// we aren't about to re-scan a circular-ref'd manifest file we've
		// already brought in.
		if len(manifStack) >= MaxManifestRecursionDepth {
			return topLevelWorldData{}, fmt.Errorf("manifest file %q: %w", path, ErrManifestStackOverflow)
		}
		for i := range manifStack {
			if manifStack[i] == path {
				return topLevelWorldData{}, fmt.Errorf("manifest file %q: %w", path, ErrManifestCircularRef)
			}
		}

		unmarshaledManif, err := unmarshalManifest(fileData)
		if err != nil {
			return topLevelWorldData{}, fmt.Errorf("manifest file %q: %w", path, err)
		}
		manif, err := parseManifest(unmarshaledManif)
		if err != nil {
			return topLevelWorldData{}, fmt.Errorf("manifest file %q: %w", path, err)
		}

		// the len of manifStack is included in the check because an empty
		// manifest error is really only a problem for the very first manifest.
		if len(manif.Files) < 1 && len(manifStack) == 0 {
			return topLevelWorldData{}, fmt.Errorf("manifest file %q: %w", path, ErrManifestEmpty)
		}

		// combine all referred to files in one single unmarshaled data struct

		unmarshaled := topLevelWorldData{}

		// copy the manif stack into a new value and add self to it for recursive calls
		manifSubStack := make([]string, len(manifStack)+1)
		copy(manifSubStack, manifStack)
		manifSubStack[len(manifSubStack)-1] = path

		manifDir := filepath.Dir(path)

		// good to know an actual count of non-skipped files so we can error on
		// the specific case of first file was manifest and referred only to
		// unreadable files
		processedFiles := 0

		for _, manifRelPath := range manif.Files {
			includedFilePath := filepath.Join(manifDir, manifRelPath)

			unmarshaledFileData, err := recursiveUnmarshalResource(includedFilePath, manifSubStack)
			if err != nil {
				// if it's a circular reference, that's actually okay. we will
				// just skip reading it and move on to the next entry.
				if errors.Is(err, ErrManifestCircularRef) {
					continue
				}

				return topLevelWorldData{}, fmt.Errorf("in file referred to by manifest file:\n    %q\n%w", path, err)
			}

			// combine the loaded data
			if unmarshaledFileData.World.Start != "" {
				if unmarshaled.World.Start != "" {
					return unmarshaled, fmt.Errorf("world data file %q: duplicate start; start has already been defined as %q", path, unmarshaled.World.Start)
				}
				unmarshaled.World.Start = unmarshaledFileData.World.Start
			}
			if len(unmarshaledFileData.Pronouns) > 0 {
				unmarshaled.Pronouns = append(unmarshaled.Pronouns, unmarshaledFileData.Pronouns...)
			}
			if len(unmarshaledFileData.NPCs) > 0 {
				unmarshaled.NPCs = append(unmarshaled.NPCs, unmarshaledFileData.NPCs...)
			}
			if len(unmarshaledFileData.Rooms) > 0 {
				unmarshaled.Rooms = append(unmarshaled.Rooms, unmarshaledFileData.Rooms...)
			}
			processedFiles++
		}

		if len(manifStack) == 0 && processedFiles == 0 {
			// then we are in a case of the first file is a manifest file, and
			// gave NO valid definitions. This is an error, fail immediately
			return unmarshaled, fmt.Errorf("manifest file %q: %w", path, ErrManifestEmpty)
		}
		return unmarshaled, nil

	default:
		return topLevelWorldData{}, fmt.Errorf("%q: file does not have 'type = ' entry set to either \"DATA\" or \"MANIFEST\"", path)
	}
}

// unmarshalWorldData unmarshals world data from the given bytes. It does not
// parse or check world data.
func unmarshalWorldData(tomlData []byte) (topLevelWorldData, error) {
	var tqw topLevelWorldData
	if tomlErr := toml.Unmarshal(tomlData, &tqw); tomlErr != nil {
		return tqw, tomlErr
	}

	if strings.ToUpper(tqw.Format) != "TUNA" {
		return tqw, fmt.Errorf("in header: 'format' key must exist and be set to 'TUNA'")
	}
	if strings.ToUpper(tqw.Type) != "DATA" {
		return tqw, fmt.Errorf("in header: 'type' must exist and be set to 'DATA'")
	}

	return tqw, nil
}

// unmarshalManifest unmarshals a TQW manifest from the given bytes. It does not
// parse or check world data.
func unmarshalManifest(tomlData []byte) (topLevelManifest, error) {
	var tqw topLevelManifest
	if tomlErr := toml.Unmarshal(tomlData, &tqw); tomlErr != nil {
		return tqw, tomlErr
	}

	if strings.ToUpper(tqw.Format) != "TUNA" {
		return tqw, fmt.Errorf("in header: 'format' key must exist and be set to 'TUNA'")
	}
	if strings.ToUpper(tqw.Type) != "MANIFEST" {
		return tqw, fmt.Errorf("in header: 'type' must exist and be set to 'MANIFEST'")
	}

	return tqw, nil
}
