package model

type Notes = []uint8

type Chord struct {
	TicksOffset uint64
	Notes       Notes
	FileNum     uint32
}
