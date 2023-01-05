// Package tqw has functions for loading game data using the TQW (TunaQuest
// Worlds) game data file format, a TOML-based format that is used to define
// game worlds for the engine to run.
package tqw

import (
	"errors"
	"os"
	"unicode"

	"github.com/BurntSushi/toml"
	"github.com/dekarrin/tunaq/internal/game"
)

const MaxManifestRecursionDepth = 32

var (
	// ErrManifestEmpty is the error returned when a manifest file is read
	// successfully but specifies no additional files to load.
	ErrManifestEmpty = errors.New("does not list any valid files to include")

	// ErrManifestStackOverflow is the error returned when the recusion level of
	// MaxManifestRecrusionDepth is reached and an additional Manifest is then
	// specified, which would cause recursion to go deeper.
	ErrManifestStackOverflow = errors.New("too many manifests deep")

	// ErrManifestCircularRef is the error returned when a manifest specifies any
	// series of files that with their own manifests refer back to the original
	// manifest, and therefore cannot be followed.
	ErrManifestCircularRef = errors.New("manifest inclusion chain refers back to itself")
)

// Manifest contains data loaded from one or more TQW Manifest files.
type Manifest struct {
	Files []string
}

// WorldData contains data loaded from one or more TQW World Data files.
type WorldData struct {
	// Rooms has every room in the World, pre-loaded with NPCs and Items and
	// ready for immediate use.
	Rooms map[string]*game.Room

	// Start is the room the character starts in.
	Start string
}

// FileInfo contains the essential information all TQW format files must
// contain. It can be obtained from a file by reading it into memory and calling
// ParseFileInfo on the bytes.
type FileInfo struct {
	Format string `toml:"format"`
	Type   string `toml:"type"`
}

// LoadResourceBundle loads a world up from the given TQW file. The file's
// type is auto-detected and decoding is handled appropriately; the type can
// either be "DATA" type or "MANIFEST" type; if it's manifest type, the files
// listed in it relative to it will also be loaded. All files included will be
// combined into one single set of data before being checked, and if a manifest
// is encountered, all files in it are recursively included.
//
// In the future, once 'resource packs' are available (honestly just tarball or
// zip files containing at least one manifest file at the root), setting path to
// it will result in reading the entire archive starting with the root manifest.
func LoadResourceBundle(path string) (WorldData, error) {
	unmarshaled, err := recursiveUnmarshalResource(path, nil)
	if err != nil {
		return WorldData{}, err
	}

	world, err := parseWorldData(unmarshaled)
	if err != nil {
		return world, err
	}

	return world, nil
}

// LoadManifestFile loads manifest data from a TQW file.
func LoadManifestFile(path string) (manif Manifest, err error) {
	manifestData, loadErr := os.ReadFile(path)
	if loadErr != nil {
		return manif, loadErr
	}

	unmarshaled, err := unmarshalManifest(manifestData)
	if err != nil {
		return manif, err
	}
	return parseManifest(unmarshaled)
}

// LoadWorldDataFile loads a world from a world definition
func LoadWorldDataFile(path string) (world WorldData, err error) {
	worldBinaryData, loadErr := os.ReadFile(path)
	if loadErr != nil {
		return world, loadErr
	}

	unmarshaled, err := unmarshalWorldData(worldBinaryData)
	if err != nil {
		return WorldData{}, err
	}

	return parseWorldData(unmarshaled)
}

// ScanFileInfo takes the given data bytes of bytes and attempts to read the TQW
// format common header info from it. The bytes are read up to the first
// instance of a table definition header and those bytes are parsed for the
// info. If there is an error reading the info, returns a non-nil error.
func ScanFileInfo(data []byte) (FileInfo, error) {
	// only run the toml parser up to the end of the top-lev table
	var topLevelEnd int = -1
	var onNewLine bool
	for b := range data {
		if onNewLine {
			if data[b] == '[' {
				topLevelEnd = b
				break
			}
		}

		if data[b] == '\n' {
			onNewLine = true
		} else if !unicode.IsSpace(rune(data[b])) {
			onNewLine = false
		}
	}

	scanData := data
	if topLevelEnd != -1 {
		scanData = data[:topLevelEnd]
	}

	var info FileInfo
	err := toml.Unmarshal(scanData, &info)
	return info, err
}
