package file

import (
	"github.com/jsphweid/harmondex/model"
)

func CreateFileNumMap(paths []string) model.FileNumToMidiPath {
	res := make(model.FileNumToMidiPath)
	for i, v := range paths {
		// i + 1 to avoid a fileId == 0
		res[uint32(i+1)] = v
	}
	return res
}
