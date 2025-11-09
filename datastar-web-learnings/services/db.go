package services

import (
	"context"
	"datastar-web-learnings/models"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
)

func getFirebaseConfigJson() ([]byte, error) {
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
	return firebaseConfigJson, firebaseConfigErr
}

func GetVideos(ctx context.Context, request models.GetVideosRequest, channel chan<- []models.VideoResponse) {
	var videos []models.VideoResponse
	firebaseConfigJson, firebaseConfigErr := getFirebaseConfigJson()
	if firebaseConfigErr != nil {
		fmt.Printf("Error marshalling FirebaseConfig:%v\n", firebaseConfigErr)
		channel <- videos
		return
	}
	app, appErr := firebase.NewApp(context.Background(), nil, option.WithCredentialsJSON(
		firebaseConfigJson,
	))

	if appErr != nil {
		fmt.Printf("Error initializing Firebase app:%v\n", appErr)
		channel <- videos
		return
	}

	fireStore, err := app.Firestore(ctx)

	if err != nil {
		fmt.Printf("Error getting Firestore client:%v\n", err)
		channel <- videos
		return
	}
	defer fireStore.Close()
	docSnaps, err := fireStore.Collection("data").Where("videoId", "!=", "").OrderBy("createdAt", firestore.Desc).Limit(request.Limit).Offset(request.Offset).Documents(ctx).GetAll()
	if err != nil {
		fmt.Printf("Error getting documents%v\n:", err)
		channel <- videos
		return
	}
	for _, docSnap := range docSnaps {
		video := models.VideoResponse{}
		docSnap.DataTo(&video)
		videos = append(videos, video)
	}
	channel <- videos
}

func FilterVideos(ctx context.Context, videoIds []string, channel chan<- []models.VideoResponse) {
	var videos []models.VideoResponse
	firebaseConfigJson, firebaseConfigErr := getFirebaseConfigJson()
	if firebaseConfigErr != nil {
		fmt.Printf("Error marshalling FirebaseConfig:%v\n", firebaseConfigErr)
		channel <- videos
		return
	}
	app, appErr := firebase.NewApp(context.Background(), nil, option.WithCredentialsJSON(
		firebaseConfigJson,
	))

	if appErr != nil {
		fmt.Printf("Error initializing Firebase app:%v\n", appErr)
		channel <- videos
		return
	}

	fireStore, err := app.Firestore(ctx)

	if err != nil {
		fmt.Printf("Error getting Firestore client:%v\n", err)
		channel <- videos
		return
	}
	defer fireStore.Close()
	docSnaps, err := fireStore.Collection("data").Where("videoId", "in", videoIds).OrderBy("createdAt", firestore.Desc).Documents(ctx).GetAll()
	if err != nil {
		fmt.Printf("Error getting documents:%v\n", err)
		channel <- videos
		return
	}
	for _, docSnap := range docSnaps {
		video := models.VideoResponse{}
		docSnap.DataTo(&video)
		videos = append(videos, video)
	}
	channel <- videos
}

func VerifyIdToken(ctx context.Context, idToken string, channel chan<- bool) {
	firebaseConfigJson, firebaseConfigErr := getFirebaseConfigJson()
	if firebaseConfigErr != nil {
		fmt.Printf("Error marshalling FirebaseConfig:%v\n", firebaseConfigErr)
		channel <- false
		return
	}
	app, appErr := firebase.NewApp(context.Background(), nil, option.WithCredentialsJSON(
		firebaseConfigJson,
	))

	if appErr != nil {
		fmt.Printf("Error initializing Firebase app:%v\n", appErr)
		channel <- false
		return
	}
	auth, err := app.Auth(ctx)
	if err != nil {
		fmt.Printf("Error getting Auth client:%v\n", err)
		channel <- false
		return
	}
	_, err = auth.VerifyIDToken(ctx, idToken)
	if err != nil {
		fmt.Printf("Error verifying ID token:%v\n", err)
		channel <- false
		return
	}
	channel <- true
}
func UpsertVideo(request models.UISignals, channel chan<- bool) {
	firebaseConfigJson, firebaseConfigErr := getFirebaseConfigJson()
	if firebaseConfigErr != nil {
		fmt.Printf("Error marshalling FirebaseConfig:%v\n", firebaseConfigErr)
		channel <- false
		return
	}
	app, appErr := firebase.NewApp(context.Background(), nil, option.WithCredentialsJSON(
		firebaseConfigJson,
	))

	if appErr != nil {
		fmt.Printf("Error initializing Firebase app:%v\n", appErr)
		channel <- false
		return
	}

	fireStore, err := app.Firestore(context.Background())

	if err != nil {
		fmt.Printf("Error getting Firestore client:%v\n", err)
		channel <- false
		return
	}
	defer fireStore.Close()
	_, err = fireStore.Collection("data").Doc(request.VideoId).Set(context.Background(), map[string]interface{}{
		"title":     request.Title,
		"videoId":   request.VideoId,
		"tags":      request.Tags,
		"rank":      request.Rank,
		"subtitle":  request.Subtitle,
		"createdAt": firestore.ServerTimestamp,
	})
	if err != nil {
		fmt.Printf("Error saving documents%v\n:", err)
		channel <- false
		return
	}
	channel <- true
}
