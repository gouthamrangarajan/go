package services

import (
	"context"
	"datastar-stock/components"
	"fmt"
	"net/http"
	"strings"

	datastar "github.com/starfederation/datastar/sdk/go"
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
				sse.MergeFragmentTempl(components.LoginUI(request.URL.Path), datastar.WithUseViewTransitions(true), datastar.WithSelector("main"), datastar.WithMergeMode(datastar.FragmentMergeModeInner))
				return
			}
			loginComponent.Render(request.Context(), responseWriter)
			return
		}
		token := cookie.Value
		channel := make(chan string)
		defer close(channel)
		go VerifyToken(token, request.Context(), channel)
		userId := <-channel
		if userId == "ERROR" {
			fmt.Println("invalid cookie")
			if request.Header.Get("Datastar-Request") == "true" {
				sse := datastar.NewSSE(responseWriter, request)
				sse.MergeFragmentTempl(components.LoginUI(request.URL.Path), datastar.WithUseViewTransitions(true), datastar.WithSelector("main"), datastar.WithMergeMode(datastar.FragmentMergeModeInner))
				return
			}
			loginComponent.Render(request.Context(), responseWriter)
			return

		}
		ctx := context.WithValue(request.Context(), UserIDKey, userId)
		next.ServeHTTP(responseWriter, request.WithContext(ctx))
	})
}
