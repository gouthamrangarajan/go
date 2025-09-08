package services

import (
	"context"
	"datastar-stock/components"
	"datastar-stock/models"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/starfederation/datastar/sdk/go/datastar"
)

func LoggedInMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {

		if request.URL.Path == "/" || strings.HasPrefix(request.URL.Path, "/assets") ||
			request.URL.Path == "/login" {
			next.ServeHTTP(responseWriter, request)
			return
		}
		loginComponent := components.LoginPage(request.URL.Path)
		//check and validate cookie
		cookie, err := request.Cookie("token")
		if err != nil || cookie.Value == "" {
			fmt.Println("No cookie")
			if request.Header.Get("Datastar-Request") == "true" {
				sse := datastar.NewSSE(responseWriter, request)

				// revisit below
				// sse.ExecuteScript("removeConfineFocusToModal()", datastar.WithExecuteScriptAutoRemove(true))
				sse.RemoveElement("#overlay")
				sse.PatchElementTempl(components.LoginUI(request.URL.Path), datastar.WithUseViewTransitions(true), datastar.WithSelector("main"), datastar.WithMode(datastar.ElementPatchModeInner))
				return
			}
			loginComponent.Render(request.Context(), responseWriter)
			return
		}
		token := cookie.Value
		verifyTokenChannel := make(chan string)
		defer close(verifyTokenChannel)
		go VerifyToken(token, request.Context(), verifyTokenChannel)
		possibleUserId := <-verifyTokenChannel
		if possibleUserId == "TOKEN EXPIRED" {
			fmt.Println("Token expired, refreshing token")
			refreshTokenChannel := make(chan string)
			defer close(refreshTokenChannel)
			go GetCachedRefreshToken(token, refreshTokenChannel)
			refreshToken := <-refreshTokenChannel
			if refreshToken == "" {
				possibleUserId = "ERROR"
			} else {
				refreshTokenSignInChannel := make(chan models.SignInRefreshTokenResponse)
				go SignInWithRefreshToken(refreshToken, refreshTokenSignInChannel)
				refreshTokenResponse := <-refreshTokenSignInChannel
				if refreshTokenResponse.IDToken == "" {
					possibleUserId = "ERROR"
				} else {
					cacheRefreshTokenChannel := make(chan string)
					possibleUserId = refreshTokenResponse.Email
					go CacheRefreshToken(models.Tokens{IdToken: refreshTokenResponse.IDToken, RefreshToken: refreshTokenResponse.RefreshToken}, cacheRefreshTokenChannel)
					expiresIn := time.Now().Add(24 * 60 * time.Minute) // Default to 1 day
					http.SetCookie(responseWriter, &http.Cookie{
						Name:     "token",
						Value:    refreshTokenResponse.IDToken,
						Path:     "/",
						HttpOnly: true,
						Secure:   !(os.Getenv("ENVIRONMENT") == "Development"),
						Expires:  expiresIn,
						SameSite: http.SameSiteLaxMode,
					})
					if request.Header.Get("Datastar-Request") == "true" {
						sse := datastar.NewSSE(responseWriter, request)
						sse.ExecuteScript("window.location.href = window.location.href", datastar.WithExecuteScriptAutoRemove(true))
						<-cacheRefreshTokenChannel
						return
					} else {
						<-cacheRefreshTokenChannel
					}

				}
			}

		}
		if possibleUserId == "ERROR" {
			fmt.Println("invalid cookie")
			if request.Header.Get("Datastar-Request") == "true" {
				sse := datastar.NewSSE(responseWriter, request)
				//revisit below
				// sse.ExecuteScript("removeConfineFocusToModal()", datastar.WithExecuteScriptAutoRemove(true))
				sse.RemoveElement("#overlay")
				sse.PatchElementTempl(components.LoginUI(request.URL.Path), datastar.WithUseViewTransitions(true), datastar.WithSelector("main"), datastar.WithMode(datastar.ElementPatchModeInner))
				return
			}
			loginComponent.Render(request.Context(), responseWriter)
			return

		}
		ctx := context.WithValue(request.Context(), UserIDKey, possibleUserId)
		next.ServeHTTP(responseWriter, request.WithContext(ctx))
	})
}
