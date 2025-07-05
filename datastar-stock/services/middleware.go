package services

import (
	"datastar-stock/components"
	"fmt"
	"net/http"
	"strings"
)

func LoggedIn(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/" || strings.HasPrefix(request.URL.Path, "/assets") ||
			request.URL.Path == "/login" {
			next.ServeHTTP(responseWriter, request)
			return
		}
		loginComponent := components.Login(request.URL.Path)
		//check and validate cookie
		cookie, err := request.Cookie("token")
		if err != nil || cookie.Value == "" {
			fmt.Println("No cookie")
			loginComponent.Render(request.Context(), responseWriter)
			return
		}
		token := cookie.Value
		channel := make(chan bool)
		defer close(channel)
		go VerifyToken(token, request.Context(), channel)
		isValid := <-channel
		if !isValid {
			fmt.Println("invalid cookie")
			loginComponent.Render(request.Context(), responseWriter)
			return
		}
		next.ServeHTTP(responseWriter, request)
	})
}
