package game

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const MaxManifestRecursionDepth = 32

// Manifest contains data loaded from one or more TQW Manifest files.
type Manifest struct {
	Files []string
}

// WorldData contains data loaded from one or more TQW World Data files.
type WorldData struct {
	// Rooms has every room in the World, pre-loaded with NPCs and Items and
	// ready for immediate use.
	Rooms map[string]*Room

	// Start is the room the character starts in.
	Start string
}

// ErrManifestEmpty is the error returned when a manifest file is read
// successfully but specifies no additional files to load.
var ErrManifestEmpty = errors.New("does not list any valid files to include")

// ErrManifestStackOverflow is the error returned when the recusion level of
// MaxManifestRecrusionDepth is reached and an additional Manifest is then
// specified, which would cause recursion to go deeper.
var ErrManifestStackOverflow = errors.New("too many manifests deep")

// ErrManifestCircularRef is the error returned when a manifest specifies any
// series of files that with their own manifests refer back to the original
// manifest, and therefore cannot be followed.
var ErrManifestCircularRef = errors.New("manifest inclusion chain refers back to itself")

// LoadTQWResourceBundle loads a world up from the given TQW file. The file's
// type is auto-detected and decoding is handled appropriately; the type can
// either be "DATA" type or "MANIFEST" type; if it's manifest type, the files
// listed in it relative to it will also be loaded. All files included will be
// combined into one single set of data before being checked, and if a manifest
// is encountered, all files in it are recursively included.
//
// In the future, once 'resource packs' are available (honestly just tarball or
// zip files containing at least one manifest file at the root), setting path to
// it will result in reading the entire archive starting with the root manifest.
func LoadTQWResourceBundle(path string) (WorldData, error) {
	unmarshaled, err := recursiveUnmarshalTQWResource(path, nil)
	if err != nil {
		return WorldData{}, err
	}

	world, err := parseUnmarshaledData(unmarshaled)
	if err != nil {
		return world, err
	}

	return world, nil
}

// LoadManifestFile loads manifest data from a TQW file.
func LoadManifestFile(path string) (manif Manifest, err error) {
	manifestData, loadErr := os.ReadFile(path)
	if loadErr != nil {
		return manif, fmt.Errorf("reading manifest file: %w", loadErr)
	}

	manif, err = ParseManifestFromTOML(manifestData)
	if err != nil {
		return manif, fmt.Errorf("loading manifest file: %w", err)
	}

	return manif, nil
}

// LoadWorldDataFile loads a world from a world definition
func LoadWorldDataFile(path string) (world WorldData, err error) {
	worldBinaryData, loadErr := os.ReadFile(path)
	if loadErr != nil {
		return world, fmt.Errorf("reading world file: %w", loadErr)
	}

	world, err = ParseWorldDataFromTOML(worldBinaryData)
	if err != nil {
		return world, fmt.Errorf("loading world file: %w", err)
	}

	return world, nil
}

// manifStack is for two reasons ->
// * detect circular deps (not an error, but we need to know to avoid them)
// * avoid infinite recursion (allow up to MaxManifestRecursionDepth levels)
//
// Returnes ErrManifestEmpty if and only if the first manifest in the stack is
// empty, otherwise it is not an error.
func recursiveUnmarshalTQWResource(path string, manifStack []string) (data tqwWorldData, err error) {
	path = filepath.Clean(path)

	fileData, loadErr := os.ReadFile(path)
	if loadErr != nil {
		return tqwWorldData{}, fmt.Errorf("%q: reading from disk: %w", path, loadErr)
	}

	tqwFileInfo, err := scanFileInfo(fileData)
	if err != nil {
		return tqwWorldData{}, fmt.Errorf("%q: detecting file type: %w", path, err)
	}

	if strings.ToUpper(tqwFileInfo.Format) != "TUNA" {
		return tqwWorldData{}, fmt.Errorf("%q: file does not have a 'format = \"TUNA\" entry", path)
	}

	fileType := strings.ToUpper(tqwFileInfo.Type)
	switch fileType {
	case "DATA":
		unmarshaled, err := UnmarshalTOMLWorldData(fileData)
		if err != nil {
			return unmarshaled, fmt.Errorf("world data file %q: %w", path, err)
		}
		return unmarshaled, nil
	case "MANIFEST":
		// check the stack to be sure we havent recursed too far and to be sure
		// we aren't about to re-scan a circular-ref'd manifest file we've
		// already brought in.
		if len(manifStack) >= MaxManifestRecursionDepth {
			return tqwWorldData{}, fmt.Errorf("manifest file %q: %w", path, ErrManifestStackOverflow)
		}
		for i := range manifStack {
			if manifStack[i] == path {
				return tqwWorldData{}, fmt.Errorf("manifest file %q: %w", path, ErrManifestCircularRef)
			}
		}

		manif, err := ParseManifestFromTOML(fileData)
		if err != nil {
			return tqwWorldData{}, fmt.Errorf("manifest file %q: %w", path, err)
		}

		// the len of manifStack is included in the check because an empty
		// manifest error is really only a problem for the very first manifest.
		if len(manif.Files) < 1 && len(manifStack) == 0 {
			return tqwWorldData{}, fmt.Errorf("manifest file %q: %w", path, ErrManifestEmpty)
		}

		// combine all referred to files in one single unmarshaled data struct

		unmarshaled := tqwWorldData{
			Pronouns: make(map[string]tqwPronounSet),
		}

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

			unmarshaledFileData, err := recursiveUnmarshalTQWResource(includedFilePath, manifSubStack)
			if err != nil {
				// if it's a circular reference, that's actually okay. we will
				// just skip reading it and move on to the next entry.
				if errors.Is(err, ErrManifestCircularRef) {
					continue
				}

				return tqwWorldData{}, fmt.Errorf("in file referred to by manifest file:\n    %q\n%w", path, err)
			}

			// combine the loaded data
			if unmarshaledFileData.World.Start != "" {
				if unmarshaled.World.Start != "" {
					return unmarshaled, fmt.Errorf("world data file %q: duplicate start; start has already been defined as %q", path, unmarshaled.World.Start)
				}
				unmarshaled.World.Start = unmarshaledFileData.World.Start
			}
			if len(unmarshaledFileData.Pronouns) > 0 {
				for k := range unmarshaledFileData.Pronouns {
					unmarshaled.Pronouns[k] = unmarshaledFileData.Pronouns[k]
				}
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
		return tqwWorldData{}, fmt.Errorf("%q: file does not have 'type = ' entry set to either \"DATA\" or \"MANIFEST\"", path)
	}
}
