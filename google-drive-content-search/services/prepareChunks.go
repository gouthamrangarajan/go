package services

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/textsplitter"
)

func ChunkCSVData(csvData string, rowsPerChunk int) []string {
	reader := csv.NewReader(strings.NewReader(csvData))
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Printf("Error reading CSV data: %v\n", err)
		return []string{csvData}
	}

	if len(records) == 0 {
		return []string{csvData}
	}
	headers := records[0]
	var chunks []string
	for idx := 1; idx < len(records); idx += rowsPerChunk {
		endIdx := idx + rowsPerChunk
		if endIdx > len(records) {
			endIdx = len(records)
		}
		chunkBuilder := strings.Builder{}
		chunkBuilder.WriteString(strings.Join(headers, ",") + "\n")
		for _, record := range records[idx:endIdx] {
			chunkBuilder.WriteString(strings.Join(record, ",") + "\n")
		}
		chunks = append(chunks, chunkBuilder.String())
	}
	return chunks
}

func ChunkOCRData(text string) []string {
	// 1000 characters is ~250 tokens.
	// Overlap of 200 ensures context is not lost mid-sentence.
	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(1000),
		textsplitter.WithChunkOverlap(200),
	)
	chunks, err := splitter.SplitText(text)
	if err != nil {
		return []string{text}
	}
	return chunks
}
