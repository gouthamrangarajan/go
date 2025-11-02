package models

type PineconeQueryRequest struct {
	Vector          []float32 `json:"vector"`
	TopK            int       `json:"topK"`
	Namespace       string    `json:"namespace"`
	IncludeValues   bool      `json:"includeValues"`
	IncludeMetadata bool      `json:"includeMetadata"`
}

type PineconeQueryResponse struct {
	Matches []struct {
		ID    string  `json:"id"`
		Score float32 `json:"score"`
	} `json:"matches"`
}
