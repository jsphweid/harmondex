package chord

import (
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
	assert.Equal(chords[0], with)
	assert.Equal(chords[1], without)
}
