package pinecone

type UpsertResponse struct {
	UpsertedCount int `json:"upsertedCount"`
}

type QueryResponse struct {
	Matches []struct {
		ID    string  `json:"id"`
		Score float32 `json:"score"`
	} `json:"matches"`
}
