package http

type ChatRequest struct {
	Prompt string `json:"prompt" binding:"required"`
}
type ChatResponse struct {
	Answer string `json:"answer"`
}
type AskRequest struct {
	Question string `json:"question" binding:"required"`
}
type AskResponse struct {
	Answer string `json:"answer"`
}
type errorResponse struct {
	Error string `json:"error"`
}
type IngestResponse struct {
	Source         string `json:"source"`
	ChunksIngested int    `json:"chunks_ingested"`
}
type StatsResponse struct {
	DocumentChunks int `json:"document_chunks"`
}
