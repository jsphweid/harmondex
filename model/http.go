package model

type SearchResult struct {
	FileId uint32  `json:"file_id"`
	Offset float32 `json:"offset"`
}

type SearchRequestBody struct {
	Chords []Notes
}

type ErrorResponse struct {
	Error string `json:"detail"`
}
