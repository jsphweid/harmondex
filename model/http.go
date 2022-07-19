package model

type SearchResult struct {
	FileId       uint32        `json:"file_id"`
	Offset       float32       `json:"offset"`
	MidiMetadata *MidiMetadata `json:"midi_metadata"`
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
