package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"os"
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

func Middleware(next func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ValidateUserIdInCookie(r) {
			next(w, r)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
		}
	})
}
