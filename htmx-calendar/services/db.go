package services

import (
	"encoding/json"
	"fmt"
	"htmx-calendar/models"
	"os"

	"github.com/supabase-community/supabase-go"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int32  `json:"expires_in"`
	ErrorCode   string `json:"error_code"`
}

func Login(request LoginRequest, channel chan<- LoginResponse) {
	response := LoginResponse{}
	anonKey := os.Getenv("SUPABASE_ANON_KEY")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	client, err := supabase.NewClient(apiUrl, anonKey, &supabase.ClientOptions{})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err)
		response.ErrorCode = fmt.Sprintf("Error connecting to supabase %v+\n", err)
		channel <- response
		return
	}
	resp, err := client.SignInWithEmailPassword(request.Email, request.Password)
	if err != nil {
		fmt.Printf("Error signing in with credentials %v\n", err)
		response.ErrorCode = fmt.Sprintf("Error signing in with credentials %v+\n", err)
		channel <- response
		return
	}
	response.AccessToken = resp.AccessToken
	response.ExpiresIn = int32(resp.ExpiresIn)
	channel <- response
}

func GetData(accessToken string, dateRange []string, channel chan<- []models.CalendarData) {
	anonKey := os.Getenv("SUPABASE_ANON_KEY")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	response := []models.CalendarData{}
	client, err := supabase.NewClient(apiUrl, anonKey, &supabase.ClientOptions{
		Headers: map[string]string{"Authorization": "Bearer " + accessToken},
	})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- response
		return
	}
	data, _, err := client.From("calendar").Select("id, task, frequency, date", "exact", false).In("date", dateRange).Execute()
	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- response
		return
	}
	if err := json.Unmarshal(data, &response); err != nil {
		fmt.Printf("Error unmarshalling results %v\n", err.Error())
	}
	channel <- response
}

func UpdateDate(accessToken string, id string, date string, channel chan<- bool) {
	anonKey := os.Getenv("SUPABASE_ANON_KEY")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	client, err := supabase.NewClient(apiUrl, anonKey, &supabase.ClientOptions{
		Headers: map[string]string{"Authorization": "Bearer " + accessToken},
	})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- false
		return
	}
	_, count, err := client.From("calendar").Update(map[string]string{"date": date}, "", "exact").Eq("id", id).Execute()
	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- false
		return
	}
	if count == 0 {
		fmt.Printf("No records affected\n")
		channel <- false
		return
	}
	// fmt.Println(string(data))
	channel <- true
}

// func LoginUsingHttpClient(request LoginRequest, channel chan<- LoginResponse) {
// 	response := LoginResponse{}
// 	anonKey := os.Getenv("SUPABASE_ANON_KEY")
// 	loginUrl := os.Getenv("SUPABASE_LOGIN_URL")
// 	jsonData, err := json.Marshal(request)
// 	if err != nil {
// 		fmt.Printf("Error converting request to json %v+\n", err)
// 		response.ErrorCode = fmt.Sprintf("Error converting request to json %v+\n", err)
// 		channel <- response
// 		return
// 	}
// 	httpRequest, err := http.NewRequest("POST", loginUrl, bytes.NewBuffer(jsonData))
// 	if err != nil {
// 		fmt.Printf("Error creating request object to supabase auth %v+\n", err)
// 		response.ErrorCode = fmt.Sprintf("Error creating request object to supabase auth %v+\n", err)
// 		channel <- response
// 		return
// 	}
// 	httpRequest.Header.Add("content-type", "application/json")
// 	httpRequest.Header.Add("apiKey", anonKey)
// 	httpClient := &http.Client{}
// 	resp, err := httpClient.Do(httpRequest)
// 	if err != nil {
// 		fmt.Printf("Error calling supabase auth %v+\n", err)
// 		response.ErrorCode = fmt.Sprintf("Error calling supabase auth %v+\n", err)
// 		channel <- response
// 		return
// 	}
// 	defer resp.Body.Close()
// 	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
// 		fmt.Printf("Error parsing response from supabase auth %v+\n", err)
// 		response.ErrorCode = fmt.Sprintf("Error parsing response from supabase auth %v+\n", err)
// 		channel <- response
// 		return
// 	}
// 	channel <- response
// }
