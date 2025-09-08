package main

import (
	"datastar-stock/components"
	"datastar-stock/components/shared"
	"datastar-stock/models"
	"datastar-stock/services"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/starfederation/datastar/sdk/go/datastar"
)

func landingPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	component := components.Landing()
	component.Render(request.Context(), responseWriter)
}
func loginHandler(responseWriter http.ResponseWriter, request *http.Request) {
	email := strings.TrimSpace(request.FormValue("email"))
	password := request.FormValue("password")
	redirect := strings.TrimSpace(request.FormValue("redirect"))
	if redirect == "" {
		redirect = "/home/populars"
	}
	signInResponse := models.SignInResponse{}
	if email != "" && password != "" {
		channel := make(chan models.SignInResponse)
		defer close(channel)
		go services.SignInEmailPassword(email, password, channel)
		signInResponse = <-channel
	}

	if email == "" || password == "" || signInResponse.ErrorMessage != "" {
		sse := datastar.NewSSE(responseWriter, request)
		sse.PatchElementTempl(shared.FormSubmitEmptyResult(), datastar.WithUseViewTransitions(true))
		sse.PatchElementTempl(shared.FormSubmitResult("Error! Please provide valid Email & Password", true), datastar.WithUseViewTransitions(true))
	} else {
		expiresIn := time.Now().Add(24 * 60 * time.Minute) // Default to 1 day
		// expiresIn := time.Now().Add(55 * time.Minute) // Default to 1 hour
		// expiresInParsed, err := strconv.Atoi(signInResponse.ExpiresIn)
		// if err == nil {
		// 	expiresIn = time.Now().Add(time.Duration(expiresInParsed-120) * time.Second) // add expiry 2 mins lesser , expiresin is seconds
		// }

		http.SetCookie(responseWriter, &http.Cookie{
			Name:     "token",
			Value:    signInResponse.IDToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   !(os.Getenv("ENVIRONMENT") == "Development"),
			Expires:  expiresIn,
			SameSite: http.SameSiteLaxMode,
		})
		channel := make(chan string)
		go services.CacheRefreshToken(models.Tokens{IdToken: signInResponse.IDToken, RefreshToken: signInResponse.RefreshToken}, channel)
		sse := datastar.NewSSE(responseWriter, request)
		sse.PatchElementTempl(shared.FormSubmitEmptyResult(), datastar.WithUseViewTransitions(true))
		sse.PatchElementTempl(shared.FormSubmitResult("Successfully logged in.", false), datastar.WithUseViewTransitions(true))
		sse.PatchElementTempl(components.LoginInSubmitBtn(true), datastar.WithUseViewTransitions(true))
		sse.ExecuteScript("window.location.href = window.location.href", datastar.WithExecuteScriptAutoRemove(true))
		<-channel
	}
}
