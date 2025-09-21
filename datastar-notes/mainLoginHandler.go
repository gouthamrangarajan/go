package main

import (
	"datastar-notes/components"
	"datastar-notes/models"
	"datastar-notes/services"
	"net/http"
	"os"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

func loginPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	components.Login().Render(request.Context(), responseWriter)
}
func sendVerificationCodeHandler(responseWriter http.ResponseWriter, request *http.Request) {
	channel := make(chan string)
	defer close(channel)
	go services.SendVerificationCode(channel)
	result := <-channel
	sse := datastar.NewSSE(responseWriter, request)
	if result == "ERROR" {
		sse.PatchElementTempl(components.OTPResult("Error Generating Verification Code. Please try again later.", true), datastar.WithUseViewTransitions(true))
		return
	}
	sse.PatchElementTempl(components.OTPResult("", false), datastar.WithUseViewTransitions(true))
}

func verifyOTPHandler(responseWriter http.ResponseWriter, request *http.Request) {
	channel := make(chan models.OTPVerificationResponse)
	defer close(channel)
	code := request.FormValue("code")
	go services.VerifyCode(code, channel)
	result := <-channel

	if result.AccessToken == "" {
		sse := datastar.NewSSE(responseWriter, request)
		sse.PatchElementTempl(components.OTPResult("The code is invalid or has expired. Please generate a new verification code.", true), datastar.WithUseViewTransitions(true))
	} else {
		secure := true
		if os.Getenv("ENV") == "Development" {
			secure = false
		}
		cookie := http.Cookie{
			Name:     "id",
			Value:    services.GenerateSignedStrForCookie(models.UICookie{Name: "id", Value: result.AccessToken}),
			Path:     "/",
			HttpOnly: true,
			Secure:   secure,
			Expires:  time.Unix(result.ExpiresAt, 0),
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(responseWriter, &cookie)
		sse := datastar.NewSSE(responseWriter, request)
		sse.ExecuteScript("window.location.replace('/')")
	}
}
