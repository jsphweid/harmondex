package constants

import "os"

func GetIndexDir() string {
	path := os.Getenv("INDEX_PATH")
	if path != "" {
		return path
	}
	return "./out"
}

func GetMediaDir() string {
	path := os.Getenv("MEDIA_PATH")
	if path != "" {
		return path
	}

	panic("MEDIA_PATH environment variable is not set!")
}

// TODO: consider storing in chords.go
// 16 for chord, 4 for offset, 4 for fileId, 1 for flags
const ChordSize = 25

const PreferredChunkSize = 64 * 1024 * 1024

// const PreferredChunkSize = 64 * 1024
