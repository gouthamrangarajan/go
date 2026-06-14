package db

import (
	"datastar-calendar/models"
	"encoding/json"
	"fmt"
	"os"

	"github.com/supabase-community/postgrest-go"
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

func GetData(accessToken string, dateRange []string, channel chan<- []models.EventData) {
	anonKey := os.Getenv("SUPABASE_ANON_KEY")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	response := []models.EventData{}
	client, err := supabase.NewClient(apiUrl, anonKey, &supabase.ClientOptions{
		Headers: map[string]string{"Authorization": "Bearer " + accessToken},
	})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- response
		return
	}
	data, _, err := client.From("calendar").Select("id, task, frequency, date, stopAfter, exact", "exact", false).In("date", dateRange).Order("created_at", &postgrest.OrderOpts{Ascending: true}).Execute()
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

func GetDataById(request struct {
	AccessToken string
	Id          string
}, channel chan<- models.EventData) {
	anonKey := os.Getenv("SUPABASE_ANON_KEY")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	response := models.EventData{}
	client, err := supabase.NewClient(apiUrl, anonKey, &supabase.ClientOptions{
		Headers: map[string]string{"Authorization": "Bearer " + request.AccessToken},
	})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- response
		return
	}
	data, _, err := client.From("calendar").Select("id, task, frequency, date, stopAfter, exact", "exact", false).Eq("id", request.Id).Single().Execute()
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

func UpdateDate(accessToken string, data models.DnD, channel chan<- bool) {
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
	_, count, err := client.From("calendar").Update(map[string]string{"date": data.Date}, "minimal", "exact").Eq("id", data.Id).Execute()
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
	channel <- true
}

func AddData(accessToken string, data models.EventData, channel chan<- models.EventData) {
	anonKey := os.Getenv("SUPABASE_ANON_KEY")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	var emptyData models.EventData
	client, err := supabase.NewClient(apiUrl, anonKey, &supabase.ClientOptions{
		Headers: map[string]string{"Authorization": "Bearer " + accessToken},
	})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- emptyData
		return
	}
	addedData := []models.EventData{}
	dataBytes, count, err := client.From("calendar").Insert(map[string]string{"date": data.Date, "task": data.Task, "frequency": data.Frequency, "user_id": data.UserId, "exact": data.Exact, "stopAfter": data.StopAfter}, false, "", "", "exact").Execute()
	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- emptyData
		return
	}
	if count > 0 {
		if err := json.Unmarshal(dataBytes, &addedData); err == nil && len(addedData) > 0 {
			channel <- addedData[0]
			return
		} else {
			fmt.Printf("Error unmarshalling added data %v\n", err.Error())
		}
	}
	channel <- emptyData
}
func UpdateData(accessToken string, data models.EventData, channel chan<- models.EventData) {
	anonKey := os.Getenv("SUPABASE_ANON_KEY")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	var emptyData models.EventData
	client, err := supabase.NewClient(apiUrl, anonKey, &supabase.ClientOptions{
		Headers: map[string]string{"Authorization": "Bearer " + accessToken},
	})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- emptyData
		return
	}
	updatedData := []models.EventData{}
	dataBytes, count, err := client.From("calendar").Update(map[string]string{"date": data.Date, "task": data.Task, "frequency": data.Frequency, "user_id": data.UserId, "exact": data.Exact, "stopAfter": data.StopAfter}, "", "exact").Eq("id", data.Id).Execute()
	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- emptyData
		return
	}
	if count > 0 {
		if err := json.Unmarshal(dataBytes, &updatedData); err == nil && len(updatedData) > 0 {
			channel <- updatedData[0]
			return
		} else {
			fmt.Printf("Error unmarshalling updated data %v\n", err.Error())
		}
	}
	channel <- emptyData
}

func DeleteEvent(data models.DeleteEvent, channel chan<- bool) {
	anonKey := os.Getenv("SUPABASE_ANON_KEY")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	client, err := supabase.NewClient(apiUrl, anonKey, &supabase.ClientOptions{
		Headers: map[string]string{"Authorization": "Bearer " + data.AccessToken},
	})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- false
		return
	}
	_, count, err := client.From("calendar").Delete("minimal", "exact").Eq("id", data.Id).Execute()
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
	channel <- true
}
