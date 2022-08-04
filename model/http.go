package model

type SearchResultV2 struct {
	FileId       uint32        `json:"file_id"`
	Offsets      []float32     `json:"offsets"`
	MidiMetadata *MidiMetadata `json:"midi_metadata"`
}

type SearchResponse struct {
	Start      int              `json:"start"`
	NumMatches int              `json:"num_matches"`
	NumFiles   int              `json:"num_files"`
	Results    []SearchResultV2 `json:"results"`
}

type MidiMetadata struct {
	Year    uint   `json:"year"`
	Artist  string `json:"artist`
	Title   string `json:"title`
	Release string `json:"release`
}

type SearchRequestBody struct {
	Chords []Notes
}

type ErrorResponse struct {
	Error string `json:"detail"`
}
