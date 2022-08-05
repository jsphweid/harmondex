package chord

import (
	"fmt"
	"testing"

	"github.com/jsphweid/harmondex/model"
	"github.com/stretchr/testify/assert"
)

func TestSortsChordsWithMetadataFirst(t *testing.T) {
	with := model.Chord{FileHasMetadata: true}
	without := model.Chord{FileHasMetadata: false}
	chords := []model.Chord{without, with}
	RankSortChords(chords)

	assert := assert.New(t)
	assert.Equal(chords[0].FileHasMetadata, true)
	assert.Equal(chords[1].FileHasMetadata, false)
}

func TestSortsChordsWithFormedByNoteOnFirst(t *testing.T) {
	with := model.Chord{FormedByNoteOn: true}
	without := model.Chord{FormedByNoteOn: false}
	chords := []model.Chord{without, with}
	RankSortChords(chords)

	assert := assert.New(t)
	assert.Equal(chords[0].FormedByNoteOn, true)
	assert.Equal(chords[1].FormedByNoteOn, false)
}

func TestSortsChordsWithOldestEventWithin1SecFirst(t *testing.T) {
	with := model.Chord{OldestEventWithin1Sec: true}
	without := model.Chord{OldestEventWithin1Sec: false}
	chords := []model.Chord{without, with}
	RankSortChords(chords)

	assert := assert.New(t)
	assert.Equal(chords[0].OldestEventWithin1Sec, false)
	assert.Equal(chords[1].OldestEventWithin1Sec, false)
}

func TestSortsCorrectlyForInterestingExample(t *testing.T) {
	// NOTE: FileNum is being used as an ID here
	chord1 := model.Chord{
		FileNum:               1,
		FileHasMetadata:       true,
		FormedByNoteOn:        true,
		OldestEventWithin1Sec: true,
	}
	chord2 := model.Chord{
		FileNum:               2,
		FileHasMetadata:       false,
		FormedByNoteOn:        true,
		OldestEventWithin1Sec: true,
	}
	chord3 := model.Chord{
		FileNum:               3,
		FileHasMetadata:       true,
		FormedByNoteOn:        false,
		OldestEventWithin1Sec: true,
	}
	chord4 := model.Chord{
		FileNum:               4,
		FileHasMetadata:       true,
		FormedByNoteOn:        false,
		OldestEventWithin1Sec: false,
	}
	chords := []model.Chord{chord4, chord3, chord1, chord2}
	RankSortChords(chords)

	assert := assert.New(t)
	assert.Equal(chords[0].FileNum, uint32(1))
	assert.Equal(chords[1].FileNum, uint32(2))
	assert.Equal(chords[2].FileNum, uint32(3))
	assert.Equal(chords[3].FileNum, uint32(4))
}

func TestFlagsSerializeDeserialize(t *testing.T) {
	cases := []model.ChordFlag{
		{FileHasMetadata: true, FormedByNoteOn: true, OldestEventWithin1Sec: true},
		{FileHasMetadata: false, FormedByNoteOn: false, OldestEventWithin1Sec: false},
		{FileHasMetadata: true, FormedByNoteOn: false, OldestEventWithin1Sec: true},
		{FileHasMetadata: false, FormedByNoteOn: true, OldestEventWithin1Sec: false},
	}

	for _, cf := range cases {
		name := fmt.Sprintf("test serialize/deserialize for ChordFlag: %v", cf)
		t.Run(name, func(t *testing.T) {
			deserialized := deserializeChordFlags(serializeChordFlags(cf))
			if cf != deserialized {
				t.Error()
			}
		})
	}
}

func TestChordSerializeDeserialize(t *testing.T) {
	chord := model.Chord{
		Offset:                1,
		Notes:                 []uint8{1, 2, 3},
		FileNum:               2,
		FileHasMetadata:       true,
		FormedByNoteOn:        true,
		OldestEventWithin1Sec: true,
	}

	assert := assert.New(t)
	assert.Equal(chord, Deserialize(Serialize(chord)))
}
