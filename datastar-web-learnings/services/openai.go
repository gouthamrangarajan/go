package services

import (
	"bytes"
	"datastar-web-learnings/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

var prompt = `You will be given a search query from a user. Your task is to decide whether this query is related to technology or tech topics.

Consider these as related to technology:
- General tech concepts (e.g., programming, AI, blockchain, gadgets)
- Technology companies or products
- Names of technology personalities, developers, engineers, or influencers (e.g., "Evan You", "Dan Abramov")
- Technology events or conferences
- Anything involving software, hardware, coding, IT, or digital innovations

If the query is related to technology, respond with: YES  
If it is not related to technology, respond with: NO  

Do not provide any explanation or additional text, only YES or NO.

Example:  
Query: "Evan You"  
Answer: YES

Query: "Best cooking recipes"  
Answer: NO

Now, decide for this query:  
"%v"`

var prompt2 = `You are an intelligent search query optimizer for a technology video library.
Your primary task is to assess a user's search query.

**Part 1: Technology Relevance Check**
First, determine if the query is primarily related to "technology" in a broad sense. This includes programming, software development, hardware, gadgets, AI, machine learning, data science, cybersecurity, cloud computing, specific tech products (like Python, JavaScript, AWS, iPhone, Nvidia GPUs), tech companies, **and the names of prominent figures or creators within the technology domain (e.g., Linus Torvalds, Elon Musk, Grace Hopper, Evan You)**.

**Part 2: Query Optimization (if technology-related)**
If the query is technology-related, suggest an optimized version of the query for vector similarity search. The optimized query should be:
- More descriptive and comprehensive.
- Include relevant keywords or concepts that clarify the user's intent.
- Avoid conversational filler words.
- Expand abbreviations if commonly understood (e.g., "AI" -> "Artificial Intelligence").
- **Crucially, if the original query is a person's name (e.g., "Evan You", "Linus Torvalds"), expand it to something like "videos by Evan You" or "contributions of Linus Torvalds" to better capture intent for video search.**

If the query is NOT technology-related, the optimized query should be an empty string.

**Output Format:**
Respond with two lines.
Line 1: "TECHNOLOGY" or "NOT_TECHNOLOGY"
Line 2: The optimized search query (or an empty string if NOT_TECHNOLOGY)

Examples:

Query: "how to build a website"
TECHNOLOGY
web development tutorial website creation from scratch

Query: "best coffee recipes"
NOT_TECHNOLOGY


Query: "machine learning explained"
TECHNOLOGY
explain machine learning concepts and applications

Query: "by Elon Musk"
TECHNOLOGY
videos by Elon Musk interviews presentations

Query: "latest iPhone release"
TECHNOLOGY
latest Apple iPhone model review features

Query: "history of ancient Rome"
NOT_TECHNOLOGY


Query: "Vue.js tutorial"
TECHNOLOGY
Vue JavaScript framework tutorial guide

Query: "Evan You"
TECHNOLOGY
videos by Evan You Vue.js creator

Query: "what is blockchain"
TECHNOLOGY
explain blockchain technology concepts decentralized ledger

Query: "how to tie a knot"
NOT_TECHNOLOGY


Query: "Nvidia GPU review"
TECHNOLOGY
Nvidia graphics card GPU review performance

Query: "healthy breakfast ideas"
NOT_TECHNOLOGY


Query: "Linus Torvalds contributions"
TECHNOLOGY
Linus Torvalds open source Linux contributions

Query: "cloud security best practices"
TECHNOLOGY
cloud computing security best practices implementation guide

Query: "%v"`

func GetOpenAIEmbeddings(text string, channel chan<- []float32) {
	url := os.Getenv("OPENAI_API_EMBEDDING_URL")
	key := os.Getenv("OPENAI_API_KEY")
	model := os.Getenv("OPENAI_API_EMBEDDING_MODEL")

	requestBody := models.OpenAIRequest{
		Input: text,
		Model: model,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Printf("Error marshalling request body for embedding request: %v\n", err)
		channel <- nil
		return
	}
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating HTTP request for embedding request: %v\n", err)
		channel <- nil
		return
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+key)
	client := &http.Client{}
	response, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error making HTTP request for embedding request: %v\n", err)
		channel <- nil
		return
	}
	defer response.Body.Close()
	responseData, err := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		fmt.Printf("Non 200 respone received calling open ai embedding: %v\n", response.StatusCode)
		if err == nil {
			fmt.Printf("Response body: %v\n", string(responseData))
		}
		channel <- nil
		return
	}
	if err != nil {
		fmt.Printf("Error reading response body for embedding request: %v\n", err)
		channel <- nil
		return
	}
	// fmt.Printf("OpenAI Embedding response: %v\n", string(responseData))
	var embeddingResponse models.OpenAIEmbeddingResponse
	err = json.Unmarshal(responseData, &embeddingResponse)
	if err != nil {
		fmt.Printf("Error unmarshalling response body for embedding request: %v\n", err)
		channel <- nil
		return
	}
	if len(embeddingResponse.Data) == 0 {
		fmt.Printf("No embedding data found in response, response body: %v\n", string(responseData))
		channel <- nil
		return
	}
	channel <- embeddingResponse.Data[0].Embedding
}

func VerifyTechnologyTopicsSearch(text string, channel chan<- bool) {
	url := os.Getenv("OPENAI_API_URL")
	key := os.Getenv("OPENAI_API_KEY")
	model := os.Getenv("OPENAI_API_MODEL")

	requestBody := models.OpenAIRequest{
		Input: fmt.Sprintf(prompt, text),
		Model: model,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Printf("Error marshalling request body for open ai request: %v\n", err)
		channel <- false
		return
	}
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating HTTP request for open ai request: %v\n", err)
		channel <- false
		return
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+key)
	client := &http.Client{}
	response, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error making HTTP request for open ai request: %v\n", err)
		channel <- false
		return
	}
	defer response.Body.Close()
	responseData, err := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		fmt.Printf("Non 200 respone received calling open ai request: %v\n", response.StatusCode)
		if err == nil {
			fmt.Printf("Response body: %v\n", string(responseData))
		}
		channel <- false
		return
	}
	if err != nil {
		fmt.Printf("Error reading response body for open ai request: %v\n", err)
		channel <- false
		return
	}
	// fmt.Printf("OpenAI response: %v\n", string(responseData))
	var embeddingResponse models.OpenAIResponse
	err = json.Unmarshal(responseData, &embeddingResponse)
	if err != nil {
		fmt.Printf("Error unmarshalling response body for open ai request: %v\n", err)
		channel <- false
		return
	}
	if len(embeddingResponse.Output) == 0 {
		fmt.Printf("No output data found in open ai response, response body: %v\n", string(responseData))
		channel <- false
		return
	}
	for _, output := range embeddingResponse.Output {
		if output.Role == "assistant" {
			for _, content := range output.Content {
				if content.Text == "YES" {
					channel <- true
					return
				}
			}
		}
	}
	channel <- false
}
func VerifyTechnologyTopicsSearchAndOptimizeQuery(query string, channel chan<- string) {
	url := os.Getenv("OPENAI_API_URL")
	key := os.Getenv("OPENAI_API_KEY")
	model := os.Getenv("OPENAI_API_MODEL")

	requestBody := models.OpenAIRequest{
		Input: fmt.Sprintf(prompt2, query),
		Model: model,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Printf("Error marshalling request body for open ai request: %v\n", err)
		channel <- ""
		return
	}
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating HTTP request for open ai request: %v\n", err)
		channel <- ""
		return
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+key)
	client := &http.Client{}
	response, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error making HTTP request for open ai request: %v\n", err)
		channel <- ""
		return
	}
	defer response.Body.Close()
	responseData, err := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		fmt.Printf("Non 200 respone received calling open ai request: %v\n", response.StatusCode)
		if err == nil {
			fmt.Printf("Response body: %v\n", string(responseData))
		}
		channel <- ""
		return
	}
	if err != nil {
		fmt.Printf("Error reading response body for open ai request: %v\n", err)
		channel <- ""
		return
	}
	// fmt.Printf("OpenAI response: %v\n", string(responseData))
	var embeddingResponse models.OpenAIResponse
	err = json.Unmarshal(responseData, &embeddingResponse)
	if err != nil {
		fmt.Printf("Error unmarshalling response body for open ai request: %v\n", err)
		channel <- ""
		return
	}
	if len(embeddingResponse.Output) == 0 {
		fmt.Printf("No output data found in open ai response, response body: %v\n", string(responseData))
		channel <- ""
		return
	}
	for _, output := range embeddingResponse.Output {
		if output.Role == "assistant" {
			for _, content := range output.Content {
				contents := strings.SplitN(content.Text, "\n", 2)
				if len(contents) > 1 {
					// fmt.Printf("Sending message to channel: %v\n", contents[1])
					channel <- strings.TrimSpace(contents[1])
					return
				}
			}
		}
	}
	channel <- ""
}
