package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"htmx-grocery/components"
	"net/http"
	"os"
	"strings"
	"time"
)

func GenerateUserIdCookie() http.Cookie {
	secure := true
	if os.Getenv("ENV") == "Development" {
		secure = false
	}
	cookieSecretKey := os.Getenv("COOKIE_SECRET")
	cookieName := "id"
	userId := os.Getenv("USER_ID")

	mac := hmac.New(sha256.New, []byte(cookieSecretKey))
	mac.Write([]byte(cookieName))
	mac.Write([]byte(userId))
	signature := mac.Sum(nil)

	cookieValueSignedBytes := append(signature, []byte(userId)...)
	cookieValueSignedStr := base64.URLEncoding.EncodeToString(cookieValueSignedBytes)

	cookie := http.Cookie{
		Name:     cookieName,
		Value:    cookieValueSignedStr,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		Expires:  time.Now().Add(365 * 24 * time.Hour),
		SameSite: http.SameSiteLaxMode,
	}
	return cookie
}
func ValidateUserIdInCookie(r *http.Request) bool {
	cookieName := "id"
	userIdFromConfig := os.Getenv("USER_ID")
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return false
	}
	cookieValueBase64Encoded := cookie.Value
	cookieValueSignedStr, err := base64.URLEncoding.DecodeString(cookieValueBase64Encoded)
	if err != nil {
		return false
	}

	cookieValueSignedBytes := []byte(cookieValueSignedStr)
	signature := cookieValueSignedBytes[:sha256.Size]

	userIdFromCookie := cookieValueSignedBytes[sha256.Size:]

	cookieSecretKey := os.Getenv("COOKIE_SECRET")
	mac := hmac.New(sha256.New, []byte(cookieSecretKey))
	mac.Write([]byte(cookieName))
	mac.Write([]byte(userIdFromConfig))
	expectedSignature := mac.Sum(nil)

	if !hmac.Equal(signature, expectedSignature) {
		return false
	}
	return string(userIdFromCookie) == userIdFromConfig
}

func ChiMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/assets") || request.URL.Path == "/login" {
			next.ServeHTTP(responseWriter, request)
			return
		}
		if ValidateUserIdInCookie(request) {
			next.ServeHTTP(responseWriter, request)
			return
		}

		if request.Method == "GET" && request.URL.Path == "/" {
			sort := request.URL.Query().Get("sort")
			suggestions := request.URL.Query().Get("suggestions")
			components.MainElForLogin(sort, suggestions).Render(request.Context(), responseWriter)

		} else {
			http.Error(responseWriter, "Unauthorized", http.StatusUnauthorized)
			return
		}
	})
}
