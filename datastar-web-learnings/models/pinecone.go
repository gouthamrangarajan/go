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

type PineconeUpsertRequest struct {
	Vectors []struct {
		Id     string    `json:"id"`
		Values []float32 `json:"values"`
	} `json:"vectors"`
}

type PineconeUpsertResponse struct {
	UpsertedCount int `json:"upsertedCount"`
}
