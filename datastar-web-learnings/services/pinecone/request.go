package pinecone

type QueryRequest struct {
	Vector          []float32 `json:"vector"`
	TopK            int       `json:"topK"`
	Namespace       string    `json:"namespace"`
	IncludeValues   bool      `json:"includeValues"`
	IncludeMetadata bool      `json:"includeMetadata"`
}

type UpsertRequest struct {
	Vectors []struct {
		Id     string    `json:"id"`
		Values []float32 `json:"values"`
	} `json:"vectors"`
}
