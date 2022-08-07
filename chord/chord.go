package chord

import (
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/jsphweid/harmondex/constants"
	"github.com/jsphweid/harmondex/model"
	"github.com/jsphweid/harmondex/util"
	"gitlab.com/gomidi/midi/v2/smf"
)

type OnNotes = map[uint8]bool

func CreateChordKey(notes []uint8) string {
	sort.Slice(notes, func(i, j int) bool {
		return notes[i] < notes[j]
	})
	var res string
	for i, note := range notes {
		res += fmt.Sprintf("%v", note)
		if i < len(notes)-1 {
			res += "-"
		}
	}
	return res
}

func getChord(pressed map[uint8]int64, evt model.ReducedEvent, hasMetadata bool) model.Chord {
	var notes []uint8
	var c model.Chord
	var oldestTime int64 = 9223372036854775807 // max int64
	for note, microseconds := range pressed {
		notes = append(notes, note)
		if microseconds < oldestTime {
			oldestTime = microseconds
		}
	}

	c.Notes = notes
	c.FormedByNoteOn = !evt.IsNoteOff

	if evt.AbsTickOffset >= 4294967295 {
		panic("Exceeded max tick value of uint32... need to use uint64s probably")
	}
	c.AbsTickOffset = uint32(evt.AbsTickOffset)

	if evt.AbsTimeMicro-oldestTime <= 1000000 {
		c.OldestEventWithin1Sec = true
	}

	c.FileHasMetadata = hasMetadata

	return c
}

func GetChords(s *smf.SMF, hasMetadata bool) ([]model.Chord, error) {
	defer func() {
		// TODO: investigate why this happens someday
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()

	var reducedEvents []model.ReducedEvent

	for _, events := range s.Tracks {
		var absTicks int64
		for _, event := range events {
			absTicks += int64(event.Delta)
			absTimeMicro := s.TimeAt(absTicks)
			var channel uint8
			var key uint8
			var velocity uint8
			switch {
			case event.Message.GetNoteOn(&channel, &key, &velocity):
				rEvent := model.ReducedEvent{
					AbsTickOffset: absTicks,
					AbsTimeMicro:  absTimeMicro,
					IsNoteOff:     false,
					Note:          key,
				}
				if channel != 10 {
					// ignore drum channel messages
					reducedEvents = append(reducedEvents, rEvent)
				}
			case event.Message.GetNoteOff(&channel, &key, &velocity):
				rEvent := model.ReducedEvent{
					AbsTickOffset: absTicks,
					AbsTimeMicro:  absTimeMicro,
					IsNoteOff:     true,
					Note:          key,
				}
				if channel != 10 {
					// ignore drum channel messages
					reducedEvents = append(reducedEvents, rEvent)
				}
			}
		}
	}

	// prioritize smaller offset values, then note off
	sort.Slice(reducedEvents, func(i, j int) bool {
		if reducedEvents[i].AbsTimeMicro != reducedEvents[j].AbsTimeMicro {
			return reducedEvents[i].AbsTimeMicro < reducedEvents[j].AbsTimeMicro
		}
		return reducedEvents[i].IsNoteOff
	})

	pressed := make(map[uint8]int64)
	pressedCount := make(map[uint8]uint32)
	var res []model.Chord
	var lastEvent model.ReducedEvent
	var lastChordKey string

	for i, evt := range reducedEvents {
		// check if pressed should be added
		if i > 0 && evt.AbsTimeMicro > lastEvent.AbsTimeMicro+constants.NewChordThreshold {
			// ignore really short or really long chords
			if len(pressed) >= 2 && len(pressed) <= 16 {
				c := getChord(pressed, lastEvent, hasMetadata)
				key := CreateChordKey(c.Notes)
				if key != lastChordKey {
					res = append(res, c)
				}
				lastChordKey = key
			}
		}

		// maintain pressed state
		lastEvent = evt
		if evt.IsNoteOff {
			pressedCount[evt.Note] = pressedCount[evt.Note] - 1
			if pressedCount[evt.Note] <= 0 {
				delete(pressedCount, evt.Note)
				delete(pressed, evt.Note)
			}
		} else {
			pressedCount[evt.Note] = pressedCount[evt.Note] + 1
			pressed[evt.Note] = evt.AbsTickOffset
		}
	}

	return res, nil
}

func serializeChordFlags(flags model.ChordFlag) uint8 {
	var res uint8

	// bit 1 - FileHasMetadata
	// bit 2 - FormedByNoteOn
	// bit 3 - OldestEventWithin1Sec

	if flags.FileHasMetadata {
		res = 1<<7 | res
	}

	if flags.FormedByNoteOn {
		res = 1<<6 | res
	}

	if flags.OldestEventWithin1Sec {
		res = 1<<5 | res
	}

	return res
}

func deserializeChordFlags(num uint8) model.ChordFlag {
	// for now, we're only using 128 or 0
	var cf model.ChordFlag

	if 1<<7&num != 0 {
		cf.FileHasMetadata = true
	}

	if 1<<6&num != 0 {
		cf.FormedByNoteOn = true
	}

	if 1<<5&num != 0 {
		cf.OldestEventWithin1Sec = true
	}

	return cf
}

func createChordFlags(chord model.Chord) model.ChordFlag {
	var cf model.ChordFlag
	cf.FileHasMetadata = chord.FileHasMetadata
	cf.FormedByNoteOn = chord.FormedByNoteOn
	cf.OldestEventWithin1Sec = chord.OldestEventWithin1Sec
	return cf
}

func Serialize(chord model.Chord) []byte {
	res := make([]byte, constants.ChordSize)
	cf := createChordFlags(chord)
	copy(res[0:16], chord.Notes)
	binary.LittleEndian.PutUint32(res[16:20], chord.AbsTickOffset)
	binary.LittleEndian.PutUint32(res[20:24], chord.FileNum)
	res[24] = serializeChordFlags(cf)
	return res
}

func Deserialize(bytes []byte) model.Chord {
	var chord model.Chord
	chord.Notes = util.FilterZeros(bytes[:16])
	chord.AbsTickOffset = binary.LittleEndian.Uint32(bytes[16:20])
	chord.FileNum = binary.LittleEndian.Uint32(bytes[20:24])

	cf := deserializeChordFlags(bytes[24])
	chord.FileHasMetadata = cf.FileHasMetadata
	chord.FormedByNoteOn = cf.FormedByNoteOn
	chord.OldestEventWithin1Sec = cf.OldestEventWithin1Sec
	return chord
}

func RankSortChords(chords []model.Chord) {
	// add scores
	for i, chord := range chords {
		var score uint8
		if chord.FileHasMetadata {
			score += 1
		}
		if chord.FormedByNoteOn {
			score += 3
		}
		if chord.OldestEventWithin1Sec {
			score += 3
		}
		chords[i].RankScore = score
	}

	// sort
	sort.Slice(chords, func(i, j int) bool {
		return chords[i].RankScore > chords[j].RankScore
	})
}
