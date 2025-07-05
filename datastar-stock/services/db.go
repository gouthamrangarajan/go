package services

import (
	"context"
	"datastar-stock/models"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	firebase "firebase.google.com/go/v4"

	"google.golang.org/api/option"
)

func GetPopulars(ctx context.Context, channel chan<- models.PopularsFromDb) {
	populars := models.PopularsFromDb{}
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
		channel <- populars
		return
	}
	app, appErr := firebase.NewApp(context.Background(), nil, option.WithCredentialsJSON(
		firebaseConfigJson,
	))

	if appErr != nil {
		fmt.Println("Error initializing Firebase app:", appErr)
		channel <- populars
		return
	}

	fireStore, err := app.Firestore(ctx)

	if err != nil {
		fmt.Println("Error getting Firestore client:", err)
		channel <- populars
		return
	}
	defer fireStore.Close()
	docSnap, err := fireStore.Collection("populars").Doc("tickers").Get(ctx)
	if err != nil {
		fmt.Println("Error getting document:", err)
		channel <- populars
		return
	}
	if err := docSnap.DataTo(&populars); err != nil {
		// Handle error
		fmt.Println("Error converting document data:", err)
		channel <- populars
		return
	}
	channel <- populars
}
