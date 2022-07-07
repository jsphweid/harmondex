package file

import (
	"github.com/jsphweid/harmondex/model"
)

func CreateFileNumMap(paths []string) model.FileNumToMidiPath {
	res := make(model.FileNumToMidiPath)
	for i, v := range paths {
		res[uint32(i)] = v
	}
	return res
}
