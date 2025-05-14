package middlewares

import (
	"context"
	"htmx-gemini-chat/services"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

func Authorization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		userId := services.GetUserIdFromRequest(request)
		if userId == "" {
			if strings.ToUpper(request.Method) == "GET" {
				userId = uuid.New().String()
				secure := true
				if os.Getenv("ENV") == "Development" {
					secure = false
				}
				cookie := http.Cookie{
					Name:     "id",
					Value:    services.GenerateSignedStrForCookie("id", userId),
					Path:     "/",
					HttpOnly: true,
					Secure:   secure,
					Expires:  time.Now().Add(365 * 24 * time.Hour),
					SameSite: http.SameSiteLaxMode,
				}
				http.SetCookie(responseWriter, &cookie)
				userChannel := make(chan int)
				defer close(userChannel)
				go services.InsertUser(userId, userChannel)
				<-userChannel
			} else if strings.ToUpper(request.Method) != "GET" {
				http.Error(responseWriter, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		ctx := context.WithValue(request.Context(), services.UserIDKey, userId)
		request = request.WithContext(ctx)
		next.ServeHTTP(responseWriter, request)
	})
}
