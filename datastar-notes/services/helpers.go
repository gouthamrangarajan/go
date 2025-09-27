package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"datastar-notes/models"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type contextKey string

const UserTokenKey contextKey = "accessToken"

const SummarizePrompt = `You are an intelligent assistant designed to summarize notes.
I will provide you with the content of a note in Editor.js JSON format.
Your task is to provide a concise, clear, and relevant summary of the note.

Here are the guidelines for your summary:
1.  **Conciseness:** Aim for 3-5 sentences, unless the note is extremely long and complex, in which case a slightly longer summary (up to 7 sentences) is acceptable.
2.  **Clarity:** Use simple, direct language. Avoid jargon where possible.
3.  **Key Information:** Focus on extracting the main ideas, key facts, decisions, and action items.
4.  **Structure Awareness:** Understand that the input is structured data from a block editor. Prioritize headings, significant paragraphs, and list items. Ignore purely decorative or structural blocks if they don't contain content.
5.  **Identify Action Items:** If there are clear tasks or to-dos, highlight them.
6.  **No Extraneous Text:** Do not include any conversational filler, disclaimers, or introductory phrases like "Here is your summary:" Just provide the summary directly.
7.  **Format:** Output the summary as plain text.

Here is the Editor.js JSON content:

%v
`

func GenerateSignedStrForCookie(model models.UICookie) string {
	cookieSecret := os.Getenv("COOKIE_SECRET")
	mac := hmac.New(sha256.New, []byte(cookieSecret))
	mac.Write([]byte(model.Name))
	mac.Write([]byte(model.Value))
	signature := mac.Sum(nil)
	cookieValueSignedBytes := append(signature, []byte(model.Value)...)
	cookieValueSignedStr := base64.URLEncoding.EncodeToString(cookieValueSignedBytes)
	return cookieValueSignedStr
}
func GetAccessTokenFromRequest(request *http.Request) string {
	cookieName := "id"
	cookie, err := request.Cookie("id")
	if err != nil {
		return ""
	}
	cookieVal := cookie.Value
	cookieSecret := os.Getenv("COOKIE_SECRET")
	cookieValueDecoded, err := base64.URLEncoding.DecodeString(cookieVal)
	if err != nil {
		return ""
	}
	if len(cookieValueDecoded) <= sha256.Size {
		return ""
	}
	signatureFromCookie := cookieValueDecoded[:sha256.Size]
	accessTokenFromCookie := cookieValueDecoded[sha256.Size:]
	mac := hmac.New(sha256.New, []byte(cookieSecret))
	mac.Write([]byte(cookieName))
	mac.Write([]byte(accessTokenFromCookie))
	signature := mac.Sum(nil)
	if !hmac.Equal(signature, signatureFromCookie) {
		return ""
	}
	return string(accessTokenFromCookie)
}

func CallGeminiAPI(request models.GeminiRequest, channel chan<- string) {
	url := os.Getenv("GEMINI_URL")

	jsonData, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("Error converting request to json data to call Gemini API %v\n", err)
		channel <- "ERROR"
		return
	}
	httpClient := &http.Client{}
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating request for Gemini API %v\n", err)
		channel <- "ERROR"
		return
	}
	httpRequest.Header.Add("x-goog-api-key", os.Getenv("GEMINI_KEY"))
	httpRequest.Header.Add("Content-Type", "application/json")

	resp, err := httpClient.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error calling Gemini API %v\n", err)
		channel <- "ERROR"
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errorMsg, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error in Gemini API call: %v+\n", resp.Status)
		} else {
			fmt.Printf("Error in Gemini API call: %v+\n", string(errorMsg))
		}
		channel <- "ERROR"
		return
	}
	var responseParsed models.GeminiResponse
	err = json.NewDecoder(resp.Body).Decode(&responseParsed)
	if err != nil {
		fmt.Printf("Error parsing JSON response from Gemini API %v\n", err)
		channel <- "ERROR"
		return
	} else {
		channel <- *responseParsed.Candidates[0].Content.Parts[0].Text
	}
}
