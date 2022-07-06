package model

type ReducedEvent struct {
	TicksOffset uint64
	IsNoteOff   bool
	Note        uint8
}
