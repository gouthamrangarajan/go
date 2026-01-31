package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/securecookie"
)

// Define the duration 365 days
const SESSION_DURATION = 365 * 24 * time.Hour

const COOKIE_NAME = "chat"

func generateUserIdCookie(uuidString string) (http.Cookie, error) {
	secure := true
	if os.Getenv("ENV") == "Development" {
		secure = false
	}

	value := map[string]interface{}{
		"user_id": uuidString,
		"created": time.Now().Unix(),
	}

	hashKey, err := base64.StdEncoding.DecodeString(os.Getenv("COOKIE_HASH_KEY"))
	if err != nil {
		fmt.Printf("Error decoding COOKIE_HASH_KEY: %v\n", err)
		return http.Cookie{}, err
	}
	blockKey, err := base64.StdEncoding.DecodeString(os.Getenv("COOKIE_BLOCK_KEY"))
	if err != nil {
		fmt.Printf("Error decoding COOKIE_BLOCK_KEY: %v\n", err)
		return http.Cookie{}, err
	}

	newSecureCookie := securecookie.New(hashKey, blockKey)
	cookieValue, err := newSecureCookie.Encode(COOKIE_NAME, value)
	if err != nil {
		fmt.Printf("Error encoding cookie: %v\n", err)
		return http.Cookie{}, err
	}

	cookie := http.Cookie{
		Name:     COOKIE_NAME,
		Value:    cookieValue,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		MaxAge:   int(SESSION_DURATION.Seconds()),
		SameSite: http.SameSiteLaxMode,
	}
	return cookie, nil
}
func getUserIdInCookie(r *http.Request) string {
	retVal := ""
	hashKey, err := base64.StdEncoding.DecodeString(os.Getenv("COOKIE_HASH_KEY"))
	if err != nil {
		fmt.Printf("Error decoding COOKIE_HASH_KEY: %v\n", err)
		return retVal
	}
	blockKey, err := base64.StdEncoding.DecodeString(os.Getenv("COOKIE_BLOCK_KEY"))
	if err != nil {
		fmt.Printf("Error decoding COOKIE_BLOCK_KEY: %v\n", err)
		return retVal
	}

	newSecureCookie := securecookie.New(hashKey, blockKey)

	if cookie, err := r.Cookie(COOKIE_NAME); err == nil {
		value := make(map[string]interface{})
		// This  checks for tampering and expiration automatically
		if err = newSecureCookie.Decode(COOKIE_NAME, cookie.Value, &value); err == nil {
			if time.Now().Unix()-value["created"].(int64) < int64(SESSION_DURATION.Seconds()) {
				retVal = value["user_id"].(string)
			}
		}
	}
	return retVal
}
func AuthorizationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/assets") {
			next.ServeHTTP(responseWriter, request)
			return
		}
		userId := getUserIdInCookie(request)
		if userId != "" {
			userCheckChannel := make(chan bool)
			defer close(userCheckChannel)
			go CheckUserExistsInTable(userId, userCheckChannel)
			if !<-userCheckChannel {
				userId = ""
			}
		}
		if userId == "" {
			if strings.ToUpper(request.Method) == "GET" && request.URL.Path == "/" {
				userId = uuid.New().String()
				cookie, err := generateUserIdCookie(userId)
				if err == nil {
					http.SetCookie(responseWriter, &cookie)
				}
				userChannel := make(chan int)
				defer close(userChannel)
				go InsertUser(userId, userChannel)
				<-userChannel
			} else {
				http.Error(responseWriter, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		ctx := context.WithValue(request.Context(), UserIDKey, userId)
		request = request.WithContext(ctx)
		next.ServeHTTP(responseWriter, request)
	})
}
