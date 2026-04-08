package services

import (
	"context"
	"datastar-web-learnings/models"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var PROMPT_TO_CHECK_TECH_RELATED_SEARCH = `You will be given a search query from a user. Your task is to decide whether this query is related to technology or tech topics.

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

var PROMPT2_TO_CHECK_TECH_RELATED_SEARCH = `You are an intelligent search query optimizer for a technology video library.
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

const VECTOR_DATA_TEMPLATE = `Title: %v
Subtitle: %v
Tags: %v
Description: %v`

func ConstructTextToVectorize(data models.VideoResponse, description string) models.VideoResponse {
	data.TextToVectorize = fmt.Sprintf(VECTOR_DATA_TEMPLATE, data.Title, data.Subtitle, strings.Join(data.Tags, ", "), description)
	data.DescriptionFromYTAPI = description
	return data
}
func GetFirstSetOfVideos(ctxt context.Context) []models.VideoResponse {
	noOfItemsStr := os.Getenv("ITEMS_PER_PAGE")
	noOfItems, err := strconv.Atoi(noOfItemsStr)
	if err != nil {
		noOfItems = 12
	}
	channel := make(chan []models.VideoResponse)
	go GetVideos(ctxt, models.GetVideosRequest{Limit: noOfItems, Offset: 0}, channel)
	defer close(channel)
	videos := <-channel
	return videos
}
