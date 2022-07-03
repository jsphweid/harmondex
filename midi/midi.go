package midi

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"gitlab.com/gomidi/midi/v2/smf"
)

func ReadMidiFile(filepath string) (s *smf.SMF, e error) {
	var blank smf.SMF
	var err error

	// handle panics
	// https://github.com/gomidi/midi/issues/20
	defer func() {
		if r, ok := recover().(string); ok {
			e = errors.New(r)
		}
	}()

	dat, err := os.ReadFile(filepath)

	if err != nil {
		errText := fmt.Sprintf("Error reading midi file... %s", err.Error())
		return &blank, errors.New(errText)
	}
	res, err := smf.ReadFrom(bytes.NewReader(dat))

	if err != nil {
		errText := fmt.Sprintf("Error parsing midi file... %s", err.Error())
		return &blank, errors.New(errText)
	}

	return res, nil
}
