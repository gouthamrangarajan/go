package services

import (
	"datastar-grocery/components"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/securecookie"
)

const SESSION_DURATION = 365 * 24 * time.Hour

const COOKIE_NAME = "grocery"

func GenerateUserIdCookie() (http.Cookie, error) {
	secure := true
	if os.Getenv("ENV") == "Development" {
		secure = false
	}

	value := map[string]interface{}{
		"user_id": os.Getenv("USER_ID"),
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
func validateUserIdInCookie(request *http.Request) bool {
	hashKey, err := base64.StdEncoding.DecodeString(os.Getenv("COOKIE_HASH_KEY"))
	if err != nil {
		fmt.Printf("Error decoding COOKIE_HASH_KEY: %v\n", err)
		return false
	}
	blockKey, err := base64.StdEncoding.DecodeString(os.Getenv("COOKIE_BLOCK_KEY"))
	if err != nil {
		fmt.Printf("Error decoding COOKIE_BLOCK_KEY: %v\n", err)
		return false
	}

	newSecureCookie := securecookie.New(hashKey, blockKey)

	if cookie, err := request.Cookie(COOKIE_NAME); err == nil {
		value := make(map[string]interface{})
		// This  checks for tampering and expiration automatically
		if err = newSecureCookie.Decode(COOKIE_NAME, cookie.Value, &value); err == nil {
			if value["user_id"] == os.Getenv("USER_ID") &&
				time.Now().Unix()-value["created"].(int64) < int64(SESSION_DURATION.Seconds()) {
				return true
			}
		}
	}
	return false
}

func ChiMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/assets") || request.URL.Path == "/login" {
			next.ServeHTTP(responseWriter, request)
			return
		}
		if validateUserIdInCookie(request) {
			next.ServeHTTP(responseWriter, request)
			return
		}

		if request.Method == "GET" && request.URL.Path == "/" {
			sort := request.URL.Query().Get("sort")
			suggestions := request.URL.Query().Get("suggestions")
			// fmt.Printf("%v\n", base64.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32)))
			// fmt.Printf("%v\n", base64.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32)))
			// fmt.Printf("%v\n", base64.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32)))
			// fmt.Printf("%v\n", base64.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32)))
			components.MainElForLogin(sort, suggestions).Render(request.Context(), responseWriter)
			return

		}
		http.Error(responseWriter, "Unauthorized", http.StatusUnauthorized)
	})
}
