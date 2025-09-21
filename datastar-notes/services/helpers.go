package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"datastar-notes/models"
	"encoding/base64"
	"net/http"
	"os"
)

type contextKey string

const UserTokenKey contextKey = "accessToken"

func GenerateSignedStrForCookie(model models.UICookie) string {
	cookieSecret := os.Getenv("COOKIE_SECRET")
	mac := hmac.New(sha256.New, []byte(cookieSecret))
	mac.Write([]byte(model.Name))
	mac.Write([]byte(model.Value))
	signature := mac.Sum(nil)
	cookieValueSignedBytes := append(signature, []byte(model.Value)...)
	cookieValueSignedStr := base64.URLEncoding.EncodeToString(cookieValueSignedBytes)
	return cookieValueSignedStr
}
func GetAccessTokenFromRequest(request *http.Request) string {
	cookieName := "id"
	cookie, err := request.Cookie("id")
	if err != nil {
		return ""
	}
	cookieVal := cookie.Value
	cookieSecret := os.Getenv("COOKIE_SECRET")
	cookieValueDecoded, err := base64.URLEncoding.DecodeString(cookieVal)
	if err != nil {
		return ""
	}
	if len(cookieValueDecoded) <= sha256.Size {
		return ""
	}
	signatureFromCookie := cookieValueDecoded[:sha256.Size]
	accessTokenFromCookie := cookieValueDecoded[sha256.Size:]
	mac := hmac.New(sha256.New, []byte(cookieSecret))
	mac.Write([]byte(cookieName))
	mac.Write([]byte(accessTokenFromCookie))
	signature := mac.Sum(nil)
	if !hmac.Equal(signature, signatureFromCookie) {
		return ""
	}
	return string(accessTokenFromCookie)
}
