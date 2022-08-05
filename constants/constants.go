package constants

// TODO: consider storing in chords.go
// 16 for chord, 4 for offset, 4 for fileId, 1 for flags
const ChordSize = 25

const PreferredChunkSize = 64 * 1024 * 1024

const AllChunksFilename = "allChunks.dat"

const FileNumToNameFilename = "fileNumsToNames.dat"

// minimum number microseconds of separation between chords to justify saving
const NewChordThreshold = 10000
