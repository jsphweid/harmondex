package chord

import (
	"fmt"
	"sort"

	"github.com/jsphweid/mir1/model"
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

func getChord(pressed map[uint8]uint64) model.Chord {
	var notes []uint8
	var c model.Chord
	for note := range pressed {
		notes = append(notes, note)
		c.TicksOffset = uint64(pressed[note])
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
		var absTicks uint64
		for _, event := range events {
			absTicks += uint64(event.Delta)
			var channel uint8
			var key uint8
			var velocity uint8
			switch {
			case event.Message.GetNoteOn(&channel, &key, &velocity):
				rEvent := model.ReducedEvent{
					TicksOffset: absTicks,
					IsNoteOff:   false,
					Note:        key,
				}
				reducedEvents = append(reducedEvents, rEvent)
			case event.Message.GetNoteOff(&channel, &key, &velocity):
				rEvent := model.ReducedEvent{
					TicksOffset: absTicks,
					IsNoteOff:   true,
					Note:        key,
				}
				reducedEvents = append(reducedEvents, rEvent)
			}
		}
	}
	sort.Slice(reducedEvents, func(i, j int) bool {
		if reducedEvents[i].TicksOffset != reducedEvents[j].TicksOffset {
			return reducedEvents[i].TicksOffset < reducedEvents[j].TicksOffset
		}
		return reducedEvents[i].IsNoteOff
	})

	timestampToChords := make(map[uint64]model.Chord)
	pressed := make(map[uint8]uint64)
	for _, evt := range reducedEvents {
		if evt.IsNoteOff {
			delete(pressed, evt.Note)
			timestampToChords[evt.TicksOffset] = getChord(pressed)
		} else {
			pressed[evt.Note] = evt.TicksOffset
			timestampToChords[evt.TicksOffset] = getChord(pressed)
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
