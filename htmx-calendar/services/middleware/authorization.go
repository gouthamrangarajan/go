package middleware

import (
	"context"
	"htmx-calendar/components"
	"htmx-calendar/services"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func Authorization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if strings.Contains(request.URL.Path, "/assets/") {
			next.ServeHTTP(responseWriter, request)
			return
		}
		unAuthorized := false
		cookie, err := request.Cookie("token")
		if err != nil {
			unAuthorized = true
		} else {
			token := cookie.Value
			parsedToken, _, err := jwt.NewParser().ParseUnverified(token, jwt.MapClaims{})
			//to do validate signature
			if err != nil {
				unAuthorized = true
			} else {
				issuer, err := parsedToken.Claims.GetIssuer()
				if err != nil || os.Getenv("SUPABASE_AUTH_ISSUER") != issuer {
					unAuthorized = true
				} else {
					ctx := context.WithValue(request.Context(), services.TokenKey, token)
					request = request.WithContext(ctx)
				}
			}
		}
		switch {
		case unAuthorized && strings.Contains(request.URL.Path, "/login"):
			next.ServeHTTP(responseWriter, request)
		case unAuthorized && request.Method == http.MethodGet:
			path := request.URL.Path
			query := request.URL.RawQuery
			components.LoginPage(path, query).Render(request.Context(), responseWriter)
		case unAuthorized && request.Method == http.MethodPost:
			http.Error(responseWriter, "Unauthorized", http.StatusUnauthorized)
		case !unAuthorized && strings.Contains(request.URL.Path, "/login"):
			http.Error(responseWriter, "Method not allowed", http.StatusMethodNotAllowed)
		default:
			next.ServeHTTP(responseWriter, request)
		}
	})
}
