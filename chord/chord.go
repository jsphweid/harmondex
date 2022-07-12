package chord

import (
	"fmt"
	"sort"

	"github.com/jsphweid/harmondex/model"
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

func getChord(pressed map[uint8]int64) model.Chord {
	var notes []uint8
	var c model.Chord
	for note := range pressed {
		notes = append(notes, note)

		// storing it in millis for space savings (32 vs. 64)
		// millis is accurate enough. And if we used microseconds for 32
		// max offset would be around 1.2 hours... there could reasonably
		// be midi files that long. Millis gives us 1200 hours max length
		// which is obviously totally sufficient
		c.Offset = uint32(pressed[note] / 1000)
	}
	c.Notes = notes
	return c
}

func GetChords(s *smf.SMF) ([]model.Chord, error) {

	// TODO: investigate
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()

	var chords []model.Chord

	var reducedEvents []model.ReducedEvent

	for _, events := range s.Tracks {
		var absTicks int64
		for _, event := range events {
			absTicks += int64(event.Delta)
			absTime := s.TimeAt(absTicks)
			var channel uint8
			var key uint8
			var velocity uint8
			switch {
			case event.Message.GetNoteOn(&channel, &key, &velocity):
				rEvent := model.ReducedEvent{
					Offset:    absTime,
					IsNoteOff: false,
					Note:      key,
				}
				reducedEvents = append(reducedEvents, rEvent)
			case event.Message.GetNoteOff(&channel, &key, &velocity):
				rEvent := model.ReducedEvent{
					Offset:    absTime,
					IsNoteOff: true,
					Note:      key,
				}
				reducedEvents = append(reducedEvents, rEvent)
			}
		}
	}

	// prioritize smaller offset values then note off
	sort.Slice(reducedEvents, func(i, j int) bool {
		if reducedEvents[i].Offset != reducedEvents[j].Offset {
			return reducedEvents[i].Offset < reducedEvents[j].Offset
		}
		return reducedEvents[i].IsNoteOff
	})

	timestampToChords := make(map[int64]model.Chord)
	pressed := make(map[uint8]int64)
	for _, evt := range reducedEvents {
		if evt.IsNoteOff {
			delete(pressed, evt.Note)
			timestampToChords[evt.Offset] = getChord(pressed)
		} else {
			pressed[evt.Note] = evt.Offset
			timestampToChords[evt.Offset] = getChord(pressed)
		}
	}

	for k := range timestampToChords {
		c := timestampToChords[k]
		if len(c.Notes) > 0 {
			chords = append(chords, c)
		}
	}
	return chords, nil
}
