package model

type Notes = []uint8

type Chord struct {
	Offset          uint32 // millis
	Notes           Notes
	FileNum         uint32
	FileHasMetadata bool
}

type ChordFlag struct {
	FileHasMetadata bool
}
