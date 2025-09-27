package services

import (
	"datastar-notes/models"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/supabase-community/gotrue-go/types"
	"github.com/supabase-community/postgrest-go"
	"github.com/supabase-community/supabase-go"
)

func SendVerificationCode(email string, channel chan<- string) {
	publicKey := os.Getenv("SUPABASE_PUBLISHABLE_KEY")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	client, err := supabase.NewClient(apiUrl, publicKey, &supabase.ClientOptions{})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err)
		channel <- "ERROR"
		return
	}
	req := types.OTPRequest{
		Email:      email,
		Data:       map[string]interface{}{},
		CreateUser: true,
	}
	req.Data["emailRedirectTo"] = os.Getenv("LOGIN_REDIRECT_TO")
	err = client.Auth.OTP(req)

	if err != nil {
		fmt.Printf("Error sending verification code %v\n", err)
		channel <- "ERROR"
		return
	}
	channel <- "SUCCESS"
}

func VerifyCode(otpForm models.OTPForm, channel chan<- models.OTPVerificationResponse) {
	var retData models.OTPVerificationResponse
	publicKey := os.Getenv("SUPABASE_PUBLISHABLE_KEY")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	client, err := supabase.NewClient(apiUrl, publicKey, &supabase.ClientOptions{})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err)
		channel <- retData
		return
	}
	req := types.VerifyForUserRequest{
		Token:      otpForm.Code,
		Type:       types.VerificationType(types.VerificationTypeMagiclink),
		Email:      otpForm.Email,
		RedirectTo: os.Getenv("LOGIN_REDIRECT_TO"),
	}
	resp, err := client.Auth.VerifyForUser(req)
	if err != nil {
		dataStr := err.Error()
		if strings.Contains(dataStr, "code 200:") {
			dataStrSplit := strings.Split(dataStr, "200: ")
			if len(dataStrSplit) > 1 {
				dataVal := strings.TrimSpace(dataStrSplit[1])

				unMarshalErr := json.Unmarshal([]byte(dataVal), &resp)
				if unMarshalErr != nil {
					fmt.Printf("Error unmarshalling response %v\n", unMarshalErr)
					channel <- retData
					return
				}
				if resp.AccessToken != "" && resp.RefreshToken != "" {
					retData.AccessToken = resp.AccessToken
					retData.RefreshToken = resp.RefreshToken
					retData.ExpiresAt = resp.ExpiresAt
				}
			}
		}
	}
	channel <- retData
}

func GetAllNotes(accessToken string, channel chan<- []models.NoteData) {
	publishableKey := os.Getenv("SUPABASE_PUBLISHABLE_KEY")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	response := []models.NoteData{}
	client, err := supabase.NewClient(apiUrl, publishableKey, &supabase.ClientOptions{
		Headers: map[string]string{"Authorization": "Bearer " + accessToken},
	})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- response
		return
	}
	data, _, err := client.From("notes").Select("id, title, content_editorjs, updated_at, user_id", "exact", false).Order("order", &postgrest.OrderOpts{Ascending: true}).Execute()
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
func GetNote(accessToken string, req models.UINote, channel chan<- models.NoteData) {
	publishableKey := os.Getenv("SUPABASE_PUBLISHABLE_KEY")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	response := models.NoteData{}
	client, err := supabase.NewClient(apiUrl, publishableKey, &supabase.ClientOptions{
		Headers: map[string]string{"Authorization": "Bearer " + accessToken},
	})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- response
		return
	}
	data, _, err := client.From("notes").Select("id, title,order, content_editorjs, updated_at, user_id", "exact", false).Eq("id", req.Id).Execute()
	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- response
		return
	}
	var output []models.NoteData
	if err := json.Unmarshal(data, &output); err != nil {
		fmt.Printf("Error unmarshalling results %v\n", err.Error())
	}
	if len(output) == 1 {
		response = output[0]
	} else if len(output) == 0 {
		fmt.Printf("No matching record found in Get Note\n")
	} else if len(output) > 1 {
		fmt.Printf("Multiple matching records found in Get Note\n")
	}
	channel <- response
}
func UpdateContentFromEditorJs(accessToken string, req models.UINote, channel chan<- bool) {
	publishableKey := os.Getenv("SUPABASE_PUBLISHABLE_KEY")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	client, err := supabase.NewClient(apiUrl, publishableKey, &supabase.ClientOptions{
		Headers: map[string]string{"Authorization": "Bearer " + accessToken},
	})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- false
		return
	}
	_, count, err := client.From("notes").Update(map[string]any{"content_editorjs": req.Content, "updated_at": time.Now()}, "minimal", "exact").Eq("id", req.Id).Execute()
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
func UpdateTitle(accessToken string, req models.UINote, channel chan<- bool) {
	publishableKey := os.Getenv("SUPABASE_PUBLISHABLE_KEY")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	client, err := supabase.NewClient(apiUrl, publishableKey, &supabase.ClientOptions{
		Headers: map[string]string{"Authorization": "Bearer " + accessToken},
	})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- false
		return
	}
	_, count, err := client.From("notes").Update(map[string]any{"title": req.Title, "updated_at": time.Now()}, "minimal", "exact").Eq("id", req.Id).Execute()
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
func GetMaxOrder(accessToken string, channel chan<- int) {
	publishableKey := os.Getenv("SUPABASE_PUBLISHABLE_KEY")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	response := 0
	client, err := supabase.NewClient(apiUrl, publishableKey, &supabase.ClientOptions{
		Headers: map[string]string{"Authorization": "Bearer " + accessToken},
	})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- response
		return
	}
	data, _, err := client.From("notes").Select("order", "exact", false).Order("order", &postgrest.OrderOpts{Ascending: false}).Execute()
	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- response
		return
	}
	var notes []models.NoteData
	if err := json.Unmarshal(data, &notes); err != nil {
		fmt.Printf("Error unmarshalling results %v\n", err.Error())
	}
	if len(notes) > 0 {
		channel <- notes[0].Order
		return
	}
	channel <- response
}
func InsertNote(accessToken string, order int, channel chan<- models.NoteData) {
	response := models.NoteData{}
	parsedToken, _, err := jwt.NewParser().ParseUnverified(accessToken, jwt.MapClaims{})
	if err == nil {
		if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok {
			if userId, ok := claims["sub"].(string); ok {
				response.UserId = userId
			}
		}
	}
	publishableKey := os.Getenv("SUPABASE_PUBLISHABLE_KEY")
	apiUrl := os.Getenv("SUPABASE_API_URL")

	client, err := supabase.NewClient(apiUrl, publishableKey, &supabase.ClientOptions{
		Headers: map[string]string{"Authorization": "Bearer " + accessToken},
	})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- response
		return
	}
	data, _, err := client.From("notes").Insert(map[string]any{"user_id": response.UserId, "order": order}, false, "", "", "exact").Execute()
	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- response
		return
	}
	output := []models.NoteData{}
	err = json.Unmarshal(data, &output)
	if err != nil {
		fmt.Printf("Error unmarshalling results %v\n", err.Error())
	} else {
		response = output[0]
	}
	channel <- response
}
func DeleteNote(accessToken string, req models.UINote, channel chan<- bool) {
	publishableKey := os.Getenv("SUPABASE_PUBLISHABLE_KEY")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	client, err := supabase.NewClient(apiUrl, publishableKey, &supabase.ClientOptions{
		Headers: map[string]string{"Authorization": "Bearer " + accessToken},
	})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- false
		return
	}
	_, count, err := client.From("notes").Delete("minimal", "exact").Eq("id", req.Id).Execute()
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
func UpdateOrder(accessToken string, req models.ReorderNote, channel chan<- bool) {
	publishableKey := os.Getenv("SUPABASE_PUBLISHABLE_KEY")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	client, err := supabase.NewClient(apiUrl, publishableKey, &supabase.ClientOptions{
		Headers: map[string]string{"Authorization": "Bearer " + accessToken},
	})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- false
		return
	}
	_, count, err := client.From("notes").Update(map[string]any{"order": req.Info.NewIndex, "updated_at": time.Now()}, "minimal", "exact").Eq("id", req.Info.Id).Execute()
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
