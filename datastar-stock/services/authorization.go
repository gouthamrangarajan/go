package services

import (
	"bytes"
	"context"
	"datastar-stock/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

func VerifyToken(token string, ctx context.Context, channel chan<- bool) {
	firebaseConfig := models.FirebaseConfig{
		Type:                    os.Getenv("FIREBASE_TYPE"),
		ProjectID:               os.Getenv("FIREBASE_PROJECT_ID"),
		PrivateKeyID:            os.Getenv("FIREBASE_PRIVATE_KEY_ID"),
		PrivateKey:              strings.ReplaceAll(os.Getenv("FIREBASE_PRIVATE_KEY"), "\\n", "\n"),
		ClientEmail:             os.Getenv("FIREBASE_CLIENT_EMAIL"),
		ClientID:                os.Getenv("FIREBASE_CLIENT_ID"),
		AuthURI:                 os.Getenv("FIREBASE_AUTH_URI"),
		TokenURI:                os.Getenv("FIREBASE_TOKEN_URI"),
		AuthProviderX509CertURL: os.Getenv("FIREBASE_AUTH_PROVIDER_X509_CERT_URL"),
		ClientX509CertURL:       os.Getenv("FIREBASE_CLIENT_X509_CERT_URL"),
		UniverseDomain:          os.Getenv("FIREBASE_UNIVERSE_DOMAIN"),
	}
	firebaseConfigJson, firebaseConfigErr := json.Marshal(firebaseConfig)
	if firebaseConfigErr != nil {
		fmt.Println("Error marshalling FirebaseConfig:", firebaseConfigErr)
		channel <- false
		return
	}
	app, appErr := firebase.NewApp(context.Background(), nil, option.WithCredentialsJSON(
		firebaseConfigJson,
	))

	if appErr != nil {
		fmt.Println("Error initializing Firebase app:", appErr)
		channel <- false
		return
	}
	auth, err := app.Auth(ctx)
	if err != nil {
		fmt.Println("Error getting Auth client:", err)
		channel <- false
		return
	}
	tokenParsed, err := auth.VerifyIDToken(ctx, token)
	if err != nil {
		fmt.Println("Error verifying ID token:", err)
		channel <- false
		return
	}
	timeParsed := time.Unix(tokenParsed.Expires, 0)
	if time.Since(timeParsed) > 0 {
		fmt.Println("Token has expired", token)
		channel <- false
		return
	}
	channel <- true
}
func GetExpiresIn(token string, ctx context.Context, channel chan<- int64) {
	firebaseConfig := models.FirebaseConfig{
		Type:                    os.Getenv("FIREBASE_TYPE"),
		ProjectID:               os.Getenv("FIREBASE_PROJECT_ID"),
		PrivateKeyID:            os.Getenv("FIREBASE_PRIVATE_KEY_ID"),
		PrivateKey:              strings.ReplaceAll(os.Getenv("FIREBASE_PRIVATE_KEY"), "\\n", "\n"),
		ClientEmail:             os.Getenv("FIREBASE_CLIENT_EMAIL"),
		ClientID:                os.Getenv("FIREBASE_CLIENT_ID"),
		AuthURI:                 os.Getenv("FIREBASE_AUTH_URI"),
		TokenURI:                os.Getenv("FIREBASE_TOKEN_URI"),
		AuthProviderX509CertURL: os.Getenv("FIREBASE_AUTH_PROVIDER_X509_CERT_URL"),
		ClientX509CertURL:       os.Getenv("FIREBASE_CLIENT_X509_CERT_URL"),
		UniverseDomain:          os.Getenv("FIREBASE_UNIVERSE_DOMAIN"),
	}
	firebaseConfigJson, firebaseConfigErr := json.Marshal(firebaseConfig)
	if firebaseConfigErr != nil {
		fmt.Println("Error marshalling FirebaseConfig:", firebaseConfigErr)
		channel <- 0
		return
	}
	app, appErr := firebase.NewApp(context.Background(), nil, option.WithCredentialsJSON(
		firebaseConfigJson,
	))

	if appErr != nil {
		fmt.Println("Error initializing Firebase app:", appErr)
		channel <- 0
		return
	}
	auth, err := app.Auth(ctx)
	if err != nil {
		fmt.Println("Error getting Auth client:", err)
		channel <- 0
		return
	}
	tokenParsed, err := auth.VerifyIDToken(ctx, token)
	if err != nil {
		fmt.Println("Error verifying ID token:", err)
		channel <- 0
		return
	}
	channel <- tokenParsed.Expires
}
func SignInEmailPassword(email, password string, channel chan<- models.SignInResponse) {
	signInResponse := models.SignInResponse{
		IDToken: "ERROR",
	}
	signInUrl := os.Getenv("GOOGLE_IDENTITY_SIGNIN_URL")
	jsonBody, err := json.Marshal(models.SignInRequest{
		Email:             email,
		Password:          password,
		ReturnSecureToken: true,
	})
	if err != nil {
		fmt.Println("Error marshalling sign-in request:", err)
		channel <- signInResponse
		return
	}
	resp, err := http.Post(signInUrl, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Println("Error making sign-in request:", err)
		channel <- signInResponse
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Sign-in failed with status code:", resp.StatusCode)
		channel <- signInResponse
		return
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		channel <- signInResponse
		return
	}
	err = json.Unmarshal(bodyBytes, &signInResponse)
	if err != nil {
		fmt.Println("Error unmarshalling sign-in response:", err)
		channel <- signInResponse
		return
	}
	channel <- signInResponse
}
