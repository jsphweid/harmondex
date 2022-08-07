package model

type ReducedEvent struct {
	AbsTickOffset int64
	AbsTimeMicro  int64
	IsNoteOff     bool
	Note          uint8
}
