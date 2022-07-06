package model

type ChunkOverview struct {
	Start    string
	End      string
	Filename string
}

type ChunkIndex = map[string]Pair
type ChunkNum = uint32
type ChunkNumToFilename = map[ChunkNum]string
