package middlewares

import (
	"context"
	"datastar-claude-chat/services"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

func Authorization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		userId := services.GetUserIdFromRequest(request)
		if userId != "" {
			userCheckChannel := make(chan bool)
			defer close(userCheckChannel)
			go services.CheckUserExistsInTable(userId, userCheckChannel)
			if !<-userCheckChannel {
				userId = ""
			}
		}
		fmt.Printf("userId:%v\n", userId)
		if userId == "" {
			if strings.ToUpper(request.Method) == "GET" && request.URL.Path == "/" {
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
			} else {
				http.Error(responseWriter, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		ctx := context.WithValue(request.Context(), services.UserIDKey, userId)
		request = request.WithContext(ctx)
		next.ServeHTTP(responseWriter, request)
	})
}
