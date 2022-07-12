package model

type ReducedEvent struct {
	Offset    int64
	IsNoteOff bool
	Note      uint8
}
