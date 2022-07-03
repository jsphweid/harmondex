package model

type ReducedEvent struct {
	AbsTime   int64
	IsNoteOff bool
	Note      uint8
}
