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

func VerifyTechnologyTopicsSearchAndOptimizeQueryUsingOpenRouter(query string, channel chan<- string) {
	url := os.Getenv("OPENROUTER_API_URL")
	key := os.Getenv("OPENROUTER_API_KEY")
	responseVal := models.OpenRouterResponse{}
	aiRequestBytes, err := json.Marshal(models.OpenRouterRequest{
		Model: os.Getenv("OPENROUTER_API_MODEL_FOR_TECHNOLOGY_SEARCH_CHECK"),
		Messages: []models.OpenRouterRequestMessage{
			{
				Role:    "user",
				Content: fmt.Sprintf(PROMPT2_TO_CHECK_TECH_RELATED_SEARCH, query),
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
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(aiRequestBytes))
	if err != nil {
		fmt.Printf("Verify Technology Topic,Error creating HTTP request: %v\n", err.Error())
		channel <- ""
		return
	}
	httpRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
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

func GenerateQuizUsingOpenRouter(inputData models.UISignals, channel chan<- models.QuizResponse) {
	url := os.Getenv("OPENROUTER_API_URL")
	key := os.Getenv("OPENROUTER_API_KEY")
	retVal := models.QuizResponse{}
	responseVal := models.OpenRouterResponse{}
	aiRequestBytes, err := json.Marshal(models.OpenRouterRequest{
		Model: os.Getenv("OPENROUTER_API_MODEL"),
		Messages: []models.OpenRouterRequestMessage{
			{
				Role:    "user",
				Content: fmt.Sprintf(PROMPT_TO_GENERATE_QUIZ, inputData.QuizVideoTitle, inputData.Transcript),
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
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(aiRequestBytes))
	if err != nil {
		fmt.Printf("Generate Quiz,Error creating HTTP request: %v\n", err.Error())
		channel <- retVal
		return
	}
	httpRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
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
		content := removeJSONCodeFence(responseVal.Choices[0].Message.Content)
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

func GenerateQuizUsingOpenRouterMock(inputData models.UISignals, channel chan<- models.QuizResponse) {
	retVal := models.QuizResponse{}
	err := json.Unmarshal([]byte(QUIZ_MOCK_DATA), &retVal)
	if err != nil {
		fmt.Printf("Generate Quiz,Error unmarshaling mock quiz response: %v\n", err.Error())
		channel <- retVal
		return
	}
	channel <- retVal
}

func VerifyQuizAnswerUsingOpenRouter(userAnswer string, quizResponse models.QuizResponse, quizIndex int, channel chan<- models.AnswerEvaluation) {
	url := os.Getenv("OPENROUTER_API_URL")
	key := os.Getenv("OPENROUTER_API_KEY")
	retVal := models.AnswerEvaluation{}
	responseVal := models.OpenRouterResponse{}
	quizInIndex := quizResponse.Questions[quizIndex]
	aiRequestBytes, err := json.Marshal(models.OpenRouterRequest{
		Model: os.Getenv("OPENROUTER_API_MODEL"),
		Messages: []models.OpenRouterRequestMessage{
			{
				Role: "user",
				Content: fmt.Sprintf(PROMPT_TO_EVALUATE_ANSWER, quizInIndex.Question, quizInIndex.ShortAnswer, quizInIndex.SpeakingAnswer,
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
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(aiRequestBytes))
	if err != nil {
		fmt.Printf("Verify Quiz Answer,Error creating HTTP request: %v\n", err.Error())
		channel <- retVal
		return
	}
	httpRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
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
		content := removeJSONCodeFence(responseVal.Choices[0].Message.Content)
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
