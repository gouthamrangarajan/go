package openrouter

import (
	"bytes"
	"datastar-web-learnings/internal/models"
	"datastar-web-learnings/services"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type Client struct {
	url                                  string
	key                                  string
	PROMPT2_TO_CHECK_TECH_RELATED_SEARCH string
	PROMPT_TO_GENERATE_QUIZ              string
	PROMPT_TO_EVALUATE_ANSWER            string
	helperService                        *services.HelperSevice
	defaultModelId                       string
	modelIdForTechnologySearchCheck      string
}

func NewClient(helperService *services.HelperSevice) *Client {
	return &Client{
		helperService:                   helperService,
		url:                             os.Getenv("OPENROUTER_API_URL"),
		key:                             os.Getenv("OPENROUTER_API_KEY"),
		defaultModelId:                  os.Getenv("OPENROUTER_API_MODEL"),
		modelIdForTechnologySearchCheck: os.Getenv("OPENROUTER_API_MODEL_FOR_TECHNOLOGY_SEARCH_CHECK"),
		PROMPT2_TO_CHECK_TECH_RELATED_SEARCH: `You are an intelligent search query optimizer for a technology video library.
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

Query: "%v"`,
		PROMPT_TO_GENERATE_QUIZ: `
You are a technical communication coach.

Create a quiz using only the supplied video transcript. The learner's goal is
to explain technical concepts clearly in spoken English.

Requirements:
1. Do not introduce facts that are not supported by the transcript.
2. Generate 5 to 8 useful questions.
3. Focus on definitions, purpose, comparisons, processes, and examples.
4. For every question, provide:
   - a concise answer
   - a natural spoken answer
   - important key terms
   - an exact supporting excerpt
   - a timestamp when available
5. Ignore promotions, sponsorships, calls to action, and unrelated introductions.
6. Do not create a question if the transcript does not contain a clear answer.
7. Keep spoken answers between 2 and 4 sentences.
8. Return only JSON matching the supplied schema.

VIDEO TITLE:
%[1]v

TRANSCRIPT:
%[2]v

OUTPUT FORMAT:
{
  "summary": string,
  "talkingPoints": [
    {
      "title": string,
      "text": string
    }
  ],
  "questions": [
    {
      "id": string e.g "q1",
      "type": string,
      "difficulty": string e.g "beginner", "intermediate", "advanced",
      "question": string,
      "shortAnswer": string,
      "speakingAnswer": string,
      "keyTerms": [string],
      "sourceExcerpt": string,
      "startSeconds": string
    }
  ]
}

`,
		PROMPT_TO_EVALUATE_ANSWER: `You are an expert Technical Communication Coach helping a non-native English speaker practice explaining software engineering concepts.

### CONTEXT:
- Question Asked: %[1]v
- Reference Short Answer: %[2]v
- Ideal Natural Spoken Answer: %[3]v
- Required Key Terms to Learn: %[4]v
- User's Answer: %[5]v

### EVALUATION CRITERIA:
1. Technical Accuracy (0.0 to 1.0): Did the user demonstrate a correct understanding of the core concept?
2. Spoken English Naturalness (0.0 to 1.0): Is the phrasing grammatically sound and natural for a spoken technical discussion?
3. Keyword Usage: Identify which required key terms were used, paraphrased accurately, or completely missed.
4. Paraphrasing Rule: Accept valid paraphrases. Do NOT require exact transcript phrasing.

### INSTRUCTIONS:
- Evaluate technical accuracy and language naturalness independently. (A user can be 100%% accurate technically while sounding awkward grammatically).
- Provide a concise, constructive feedback tip (max 2 sentences).
- Provide a "Better Spoken Version" that keeps the user's original idea but rewrites it into clean, natural spoken English using any missing key terms.

### OUTPUT FORMAT:
Respond strictly in JSON matching this schema:
{
  "is_technically_correct": true,
  "accuracy_score": 0.85,
  "fluency_score": 0.70,
  "used_keywords": ["term 1"],
  "missing_keywords": ["term 2"],
  "feedback_tip": "Great technical explanation! However, try to use 'prompt caching' instead of saying 'saving the input'.",
  "improved_spoken_answer": "Output caching stores the final response, whereas prompt caching stores only the input prompt data to avoid re-processing it."
}
`,
	}
}

func (c *Client) VerifyTechnologyTopicsSearchAndOptimizeQueryUsingOpenRouter(query string, channel chan<- string) {
	defer close(channel)
	responseVal := Response{}
	aiRequestBytes, err := json.Marshal(Request{
		Model: c.modelIdForTechnologySearchCheck,
		Messages: []RequestMessage{
			{
				Role:    "user",
				Content: fmt.Sprintf(c.PROMPT2_TO_CHECK_TECH_RELATED_SEARCH, query),
			},
		},
	})
	// fmt.Printf("OpenRouter Request:%v\n", string(aiRequestBytes))
	if err != nil {
		fmt.Printf("Verify Technology Topic, Error marshaling request: %v\n", err.Error())
		channel <- ""
		return
	}
	client := &http.Client{}
	httpRequest, err := http.NewRequest("POST", c.url, bytes.NewBuffer(aiRequestBytes))
	if err != nil {
		fmt.Printf("Verify Technology Topic,Error creating HTTP request: %v\n", err.Error())
		channel <- ""
		return
	}
	httpRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.key))
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Verify Technology Topic,Error making HTTP request: %v\n", err.Error())
		channel <- ""
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		fmt.Printf("Verify Technology Topic,Error in making openrouter message api call: received status code %d\n", response.StatusCode)
		respBody, err := io.ReadAll(response.Body)
		if err == nil {
			fmt.Printf("Verify Technology Topic,Error in making openrouter message api call %v\n", string(respBody))
		}
		channel <- ""
		return
	}

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Verify Technology Topic,Error reading response body OpenRouter API call: %v\n", err.Error())
		channel <- ""
		return
	}

	err = json.Unmarshal(respBody, &responseVal)
	if err != nil {
		fmt.Printf("Verify Technology Topic,Error unmarshaling response OpenRouter API call: %v\n", err.Error())
		channel <- ""
		return
	}
	if len(responseVal.Choices) > 0 {
		content := responseVal.Choices[0].Message.Content
		contents := strings.SplitN(content, "\n", 2)
		if len(contents) > 1 {
			// fmt.Printf("Sending message to channel: %v\n", contents[1])
			channel <- strings.TrimSpace(contents[1])
			return
		}
	} else {
		fmt.Printf("Verify Technology Topic,No choices in response OpenRouter API call\n")
	}
	channel <- ""
}

func (c *Client) GenerateQuizUsingOpenRouter(inputData models.UISignals, channel chan<- QuizResponse) {
	defer close(channel)
	retVal := QuizResponse{}
	responseVal := Response{}
	aiRequestBytes, err := json.Marshal(Request{
		Model: c.defaultModelId,
		Messages: []RequestMessage{
			{
				Role:    "user",
				Content: fmt.Sprintf(c.PROMPT_TO_GENERATE_QUIZ, inputData.QuizVideoTitle, inputData.Transcript),
			},
		},
	})
	// fmt.Printf("OpenRouter Request:%v\n", string(aiRequestBytes))
	if err != nil {
		fmt.Printf("Generate Quiz,Error marshaling request: %v\n", err.Error())
		channel <- retVal
		return
	}
	client := &http.Client{}
	httpRequest, err := http.NewRequest("POST", c.url, bytes.NewBuffer(aiRequestBytes))
	if err != nil {
		fmt.Printf("Generate Quiz,Error creating HTTP request: %v\n", err.Error())
		channel <- retVal
		return
	}
	httpRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.key))
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Generate Quiz,Error making HTTP request: %v\n", err.Error())
		channel <- retVal
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		fmt.Printf("Generate Quiz,Error in making openrouter message api call: received status code %d\n", response.StatusCode)
		respBody, err := io.ReadAll(response.Body)
		if err == nil {
			fmt.Printf("Generate Quiz,Error in making openrouter message api call %v\n", string(respBody))
		}
		channel <- retVal
		return
	}

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Generate Quiz,Error reading response body OpenRouter API call: %v\n", err.Error())
		channel <- retVal
		return
	}

	err = json.Unmarshal(respBody, &responseVal)
	if err != nil {
		fmt.Printf("Generate Quiz,Error unmarshaling response OpenRouter API call: %v\n", err.Error())
		channel <- retVal
		return
	}
	if len(responseVal.Choices) > 0 {
		content := c.helperService.RemoveJSONCodeFence(responseVal.Choices[0].Message.Content)
		// fmt.Printf("Received message: %v\n", content)
		err = json.Unmarshal([]byte(content), &retVal)
		if err != nil {
			fmt.Printf("Generate Quiz,Error unmarshaling quiz response OpenRouter API call: %v\n", err.Error())
			channel <- retVal
			return
		}
	} else {
		fmt.Printf("Generate Quiz,No choices in response OpenRouter API call\n")
	}
	channel <- retVal
}

// func GenerateQuizUsingOpenRouterMock(inputData models.UISignals, channel chan<- models.QuizResponse) {
// 	retVal := models.QuizResponse{}
// 	err := json.Unmarshal([]byte(QUIZ_MOCK_DATA), &retVal)
// 	if err != nil {
// 		fmt.Printf("Generate Quiz,Error unmarshaling mock quiz response: %v\n", err.Error())
// 		channel <- retVal
// 		return
// 	}
// 	channel <- retVal
// }

func (c *Client) VerifyQuizAnswerUsingOpenRouter(userAnswer string, quizResponse QuizResponse,
	quizIndex int, channel chan<- AnswerEvaluationResponse) {
	defer close(channel)
	retVal := AnswerEvaluationResponse{}
	responseVal := Response{}
	quizInIndex := quizResponse.Questions[quizIndex]
	aiRequestBytes, err := json.Marshal(Request{
		Model: c.defaultModelId,
		Messages: []RequestMessage{
			{
				Role: "user",
				Content: fmt.Sprintf(c.PROMPT_TO_EVALUATE_ANSWER, quizInIndex.Question, quizInIndex.ShortAnswer, quizInIndex.SpeakingAnswer,
					strings.Join(quizInIndex.KeyTerms, ","), userAnswer),
			},
		},
	})
	// fmt.Printf("OpenRouter Request:%v\n", string(aiRequestBytes))
	if err != nil {
		fmt.Printf("Verify Quiz Answer,Error marshaling request: %v\n", err.Error())
		channel <- retVal
		return
	}
	client := &http.Client{}
	httpRequest, err := http.NewRequest("POST", c.url, bytes.NewBuffer(aiRequestBytes))
	if err != nil {
		fmt.Printf("Verify Quiz Answer,Error creating HTTP request: %v\n", err.Error())
		channel <- retVal
		return
	}
	httpRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.key))
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Verify Quiz Answer,Error making HTTP request: %v\n", err.Error())
		channel <- retVal
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		fmt.Printf("Verify Quiz Answer,Error in making openrouter message api call: received status code %d\n", response.StatusCode)
		respBody, err := io.ReadAll(response.Body)
		if err == nil {
			fmt.Printf("Verify Quiz Answer,Error in making openrouter message api call %v\n", string(respBody))
		}
		channel <- retVal
		return
	}

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Verify Quiz Answer,Error reading response body OpenRouter API call: %v\n", err.Error())
		channel <- retVal
		return
	}

	err = json.Unmarshal(respBody, &responseVal)
	if err != nil {
		fmt.Printf("Verify Quiz Answer,Error unmarshaling response OpenRouter API call: %v\n", err.Error())
		channel <- retVal
		return
	}
	if len(responseVal.Choices) > 0 {
		content := c.helperService.RemoveJSONCodeFence(responseVal.Choices[0].Message.Content)
		// fmt.Printf("Received message: %v\n", content)
		err = json.Unmarshal([]byte(content), &retVal)
		if err != nil {
			fmt.Printf("Verify Quiz Answer,Error unmarshaling quiz response OpenRouter API call: %v\n", err.Error())
			channel <- retVal
			return
		}
	} else {
		fmt.Printf("Verify Quiz Answer,No choices in response OpenRouter API call\n")
	}
	channel <- retVal
}
