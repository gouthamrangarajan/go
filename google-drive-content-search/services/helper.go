package services

import (
	"bytes"
	"fmt"
	"google-drive-content-search/models"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var TOKEN_FILE = ""
var CLIENT_SECRET_FILE = ""

var OPEN_ROUTER_GENERAL_PROMPT = `You are a document processing assistant specializing in structured data recovery. Your goal is to convert raw extracted text (OCR or CSV) into a professional, clean Markdown document.

Task Instructions:

1. Handling OCR Input (PDF, Image, Word):

Structure: Reconstruct the logical flow. Identify titles, headings, and sub-headings and format them as #, ##, and ###.
Cleanup: Detect and remove OCR artifacts (e.g., page numbers, broken line breaks, header/footer repetition, or garbled symbols).
Lists: Detect bulleted or numbered lists and format them correctly.
Emphasis: Retain bold or italicized intent based on context.
2. Handling CSV Input (Excel):

Tabular Format: Convert the CSV data into a standard Markdown table.
Headers: Identify the first row as the header. If the CSV contains multiple datasets, separate them with horizontal rules (---) and provide descriptive headers.
Alignment: Ensure columns are logically aligned.
Data Integrity: Do not change any numerical values or specific data points.
3. General Rules:

If the input looks like a form (labels followed by values), format it as a clean list or a two-column table.
If there are mathematical formulas, use LaTeX notation (e.g., $E=mc^2$).
Output only the Markdown content. Do not include introductions, explanations, or "Here is the result."


`

var OPEN_ROUTER_OCR_PROMPT = `The following text is OCR output. Please fix broken words caused by line-wrapping and ignore page numbers.

%s
`
var OPEN_ROUTER_CSV_PROMPT = `The following text is CSV data. Transform this into a readable Markdown table. If the data is too wide, consider breaking it into smaller logical tables or a descriptive list.

%s
`

func CleanTextForVoyage(input string) string {
	re := regexp.MustCompile(`\s+`) // matches any whitespace
	return strings.TrimSpace(re.ReplaceAllString(input, " "))
}

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file:", err)
	}
	TOKEN_FILE = os.Getenv("AUTH_CONFIG_FOLDER") + os.Getenv("TOKEN_FILE_PATH")
	CLIENT_SECRET_FILE = os.Getenv("AUTH_CONFIG_FOLDER") + os.Getenv("CLIENT_SECRET_FILE_PATH")
}

func ConvertFileDataCollectionToDocumentChunkCollection(fileDataCollection []models.FileData) []models.DocumentChunk {
	var documentChunkCollection []models.DocumentChunk
	for _, fileData := range fileDataCollection {
		chunkedContent := []string{}
		switch fileData.MimeType {
		case "application/vnd.google-apps.spreadsheet",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
			{
				chunkedContent = ChunkCSVData(fileData.ExtractedText, 10)
			}
		case "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/msword", "application/pdf",
			"image/png", "image/jpeg", "image/jpg":
			{
				chunkedContent = ChunkOCRData(fileData.ExtractedText)
			}
		default:
			{
				chunkedContent = []string{fileData.ExtractedText}
			}
		}
		for _, chunk := range chunkedContent {
			documentChunk := models.DocumentChunk{
				FileId:              fileData.Id,
				FileName:            fileData.Name,
				ChunkContent:        chunk,
				FullContentMarkdown: fileData.ExtractedTextMarkdown,
				MimeType:            fileData.MimeType,
			}
			documentChunkCollection = append(documentChunkCollection, documentChunk)
		}
	}
	return documentChunkCollection
}

func ConvertDocumentChunkCollectionToSearchResultCollection(documentChunkCollection []models.DocumentChunk) []models.SearchResult {
	var searchResultCollection []models.SearchResult
	fileAlreadyAdded := make(map[string]bool)
	for _, documentChunk := range documentChunkCollection {
		if _, exists := fileAlreadyAdded[documentChunk.FileName]; !exists {
			searchResult := models.SearchResult{
				Id:                  documentChunk.FileId,
				FileName:            documentChunk.FileName,
				MatchingContent:     documentChunk.ChunkContent,
				FileContentMarkdown: documentChunk.FullContentMarkdown,
				MatchPercent:        strconv.Itoa(100 - int(documentChunk.Distance*100)),
			}
			searchResultCollection = append(searchResultCollection, searchResult)
			fileAlreadyAdded[documentChunk.FileName] = true
		}
	}
	return searchResultCollection
}

func ConvertMarkdownToHtml(id string, source []byte, channel chan<- string) {
	var buf bytes.Buffer
	md := goldmark.New(goldmark.WithExtensions(extension.GFM, extension.DefinitionList,
		extension.Footnote, extension.Typographer, extension.CJK))
	if err := md.Convert(source, &buf); err != nil {
		fmt.Printf("Error converting markdown: %v\n", err)
		channel <- ""
		return
	}
	channel <- "<div id='markdown_" + id + "'>" + buf.String() + "</div>"
}
