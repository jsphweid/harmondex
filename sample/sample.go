package sample

import (
	"fmt"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/smf"
)

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

func Create(mf *smf.SMF, ticksOffset uint64) *smf.SMF {
	var res smf.SMF
	fmt.Printf("mf.TimeFormat: %v\n", mf.TimeFormat)
	res.TimeFormat = mf.TimeFormat

	oldTracks := mf.Tracks
	for _, track := range oldTracks {
		var newTrack smf.Track
		var absTicks uint64
		var numNoteOnOff int
	TrackEventLoop:
		for _, evt := range track {
			absTicks += uint64(evt.Delta)
			switch {
			case evt.Message.Is(midi.NoteOnMsg),
				evt.Message.Is(midi.NoteOffMsg):
				if absTicks >= ticksOffset {
					newTrack = append(newTrack, evt)
					numNoteOnOff += 1
					if numNoteOnOff >= 10 {
						newTrack.Close(0)
						break TrackEventLoop
					}
				}
			default:
				evt.Delta = min(evt.Delta, 1)
				newTrack = append(newTrack, evt)
			}
		}

		res.Tracks = append(res.Tracks, newTrack)

	}

	return &res
}
