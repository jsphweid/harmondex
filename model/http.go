package model

type SearchResult struct {
	FileId      uint32 `json:"file_id"`
	TicksOffset uint64 `json:"ticks_offset"`
}

type SearchRequestBody struct {
	Chords []Notes
}

type ErrorResponse struct {
	Error string `json:"detail"`
}
