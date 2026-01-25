package models

type DocumentChunk struct {
	Id                  int
	FileId              string
	FileName            string
	MimeType            string
	FullContentMarkdown string
	ChunkContent        string
	ChunkEmbedding      []float32
	Distance            float32
}
