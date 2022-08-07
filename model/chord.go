package model

type Notes = []uint8

type Chord struct {
	AbsTickOffset         uint32
	Notes                 Notes
	FileNum               uint32
	FileHasMetadata       bool
	FormedByNoteOn        bool
	OldestEventWithin1Sec bool

	// NOTE: not guaranteed to be meaningful
	RankScore uint8
}

type ChordFlag struct {
	FileHasMetadata       bool
	FormedByNoteOn        bool
	OldestEventWithin1Sec bool
}
