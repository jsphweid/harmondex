package chord

import (
	"fmt"
	"sort"

	"github.com/jsphweid/mir1/model"
	"gitlab.com/gomidi/midi/v2/smf"
)

func getChord(pressed map[uint8]int64) model.Chord {
	var notes []uint8
	var c model.Chord
	for note := range pressed {
		notes = append(notes, note)
		c.AbsTime = uint64(pressed[note])
	}
	c.Notes = notes
	return c
}

func ParseChords(s *smf.SMF) ([]model.Chord, error) {

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
			absTime := s.TimeAt(int64(absTicks))
			var channel uint8
			var key uint8
			var velocity uint8
			switch {
			case event.Message.GetNoteOn(&channel, &key, &velocity):
				rEvent := model.ReducedEvent{
					AbsTime:   absTime,
					IsNoteOff: false,
					Note:      key,
				}
				reducedEvents = append(reducedEvents, rEvent)
			case event.Message.GetNoteOff(&channel, &key, &velocity):
				rEvent := model.ReducedEvent{
					AbsTime:   absTime,
					IsNoteOff: true,
					Note:      key,
				}
				reducedEvents = append(reducedEvents, rEvent)
			}
		}
	}
	sort.Slice(reducedEvents, func(i, j int) bool {
		if reducedEvents[i].AbsTime != reducedEvents[j].AbsTime {
			return reducedEvents[i].AbsTime < reducedEvents[j].AbsTime
		}
		return reducedEvents[i].IsNoteOff
	})

	timestampToChords := make(map[int64]model.Chord)
	pressed := make(map[uint8]int64)
	for _, evt := range reducedEvents {
		if evt.IsNoteOff {
			delete(pressed, evt.Note)
			timestampToChords[evt.AbsTime] = getChord(pressed)
		} else {
			pressed[evt.Note] = evt.AbsTime
			timestampToChords[evt.AbsTime] = getChord(pressed)
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
