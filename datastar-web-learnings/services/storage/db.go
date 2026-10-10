package storage

import (
	"context"
	"datastar-web-learnings/internal/models"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
)

type DbService struct {
	firebaseConfig      models.FirebaseConfig
	firebaseConfigJson  []byte
	firebaseConfigError error
}

func NewDbService() *DbService {
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
	firebaseConfigJson, firebaseConfigError := json.Marshal(firebaseConfig)
	return &DbService{
		firebaseConfig:      firebaseConfig,
		firebaseConfigJson:  firebaseConfigJson,
		firebaseConfigError: firebaseConfigError,
	}
}

func (d *DbService) GetAllVideos(ctx context.Context, channel chan<- []models.VideoResponse) {
	defer close(channel)
	var videos []models.VideoResponse
	// d.firebaseConfigJson, d.firebaseConfigError := getFirebaseConfigJson()
	if d.firebaseConfigError != nil {
		fmt.Printf("Error marshalling FirebaseConfig:%v\n", d.firebaseConfigError)
		channel <- videos
		return
	}
	app, appErr := firebase.NewApp(context.Background(), nil, option.WithCredentialsJSON(
		d.firebaseConfigJson,
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
	docSnaps, err := fireStore.Collection("data").Where("videoId", "!=", "").OrderBy("createdAt", firestore.Desc).Documents(ctx).GetAll()
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

func (d *DbService) GetVideos(ctx context.Context, request models.GetVideosRequest, channel chan<- []models.VideoResponse) {
	defer close(channel)
	var videos []models.VideoResponse
	if d.firebaseConfigError != nil {
		fmt.Printf("Error marshalling FirebaseConfig:%v\n", d.firebaseConfigError)
		channel <- videos
		return
	}
	app, appErr := firebase.NewApp(context.Background(), nil, option.WithCredentialsJSON(
		d.firebaseConfigJson,
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

func (d *DbService) FilterVideos(ctx context.Context, videoIds []string, channel chan<- []models.VideoResponse) {
	defer close(channel)
	var videos []models.VideoResponse
	if d.firebaseConfigError != nil {
		fmt.Printf("Error marshalling FirebaseConfig:%v\n", d.firebaseConfigError)
		channel <- videos
		return
	}
	app, appErr := firebase.NewApp(context.Background(), nil, option.WithCredentialsJSON(
		d.firebaseConfigJson,
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

func (d *DbService) VerifyIdToken(ctx context.Context, idToken string, channel chan<- bool) {
	defer close(channel)
	if d.firebaseConfigError != nil {
		fmt.Printf("Error marshalling FirebaseConfig:%v\n", d.firebaseConfigError)
		channel <- false
		return
	}
	app, appErr := firebase.NewApp(context.Background(), nil, option.WithCredentialsJSON(
		d.firebaseConfigJson,
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
func (d *DbService) UpsertVideo(request models.UISignals, channel chan<- bool) {
	defer close(channel)
	if d.firebaseConfigError != nil {
		fmt.Printf("Error marshalling FirebaseConfig:%v\n", d.firebaseConfigError)
		channel <- false
		return
	}
	app, appErr := firebase.NewApp(context.Background(), nil, option.WithCredentialsJSON(
		d.firebaseConfigJson,
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
		"title":      request.Title,
		"videoId":    request.VideoId,
		"tags":       request.Tags,
		"rank":       request.Rank,
		"subtitle":   request.Subtitle,
		"createdAt":  firestore.ServerTimestamp,
		"transcript": request.Transcript,
	})
	if err != nil {
		fmt.Printf("Error saving documents%v\n:", err)
		channel <- false
		return
	}
	channel <- true
}
func (d *DbService) CheckAndDeleteIfDocIdAndVideoIdAreNotSame(videoId string, channel chan<- bool) {
	defer close(channel)
	if d.firebaseConfigError != nil {
		fmt.Printf("Error marshalling FirebaseConfig:%v\n", d.firebaseConfigError)
		channel <- false
		return
	}
	app, appErr := firebase.NewApp(context.Background(), nil, option.WithCredentialsJSON(
		d.firebaseConfigJson,
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
	docSnaps, err := fireStore.Collection("data").Where("videoId", "==", videoId).Documents(context.Background()).GetAll()
	if err != nil {
		fmt.Printf("Error getting documents:%v\n", err)
		channel <- false
		return
	}
	for _, docSnap := range docSnaps {
		if docSnap.Ref.ID != videoId {
			_, err = fireStore.Collection("data").Doc(docSnap.Ref.ID).Delete(context.Background())
			if err != nil {
				fmt.Printf("Error deleting document:%v\n", err)
				channel <- false
				return
			}
		}
	}
	channel <- true
}
func (d *DbService) DeleteVideo(videoId string, channel chan<- bool) {
	defer close(channel)
	if d.firebaseConfigError != nil {
		fmt.Printf("Error marshalling FirebaseConfig:%v\n", d.firebaseConfigError)
		channel <- false
		return
	}
	app, appErr := firebase.NewApp(context.Background(), nil, option.WithCredentialsJSON(
		d.firebaseConfigJson,
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
	docSnaps, err := fireStore.Collection("data").Where("videoId", "==", videoId).Documents(context.Background()).GetAll()
	if err != nil {
		fmt.Printf("Error getting documents:%v\n", err)
		channel <- false
		return
	}
	if len(docSnaps) == 0 {
		channel <- false
		return
	}
	for _, docSnap := range docSnaps {
		_, err = fireStore.Collection("data").Doc(docSnap.Ref.ID).Delete(context.Background())
		if err != nil {
			fmt.Printf("Error deleting document:%v\n", err)
			channel <- false
			return
		}
	}
	channel <- true
}
func (d *DbService) UpdateTranscriptForQuiz(request models.UISignals, channel chan<- bool) {
	defer close(channel)
	if d.firebaseConfigError != nil {
		fmt.Printf("Error marshalling FirebaseConfig:%v\n", d.firebaseConfigError)
		channel <- false
		return
	}
	app, appErr := firebase.NewApp(context.Background(), nil, option.WithCredentialsJSON(
		d.firebaseConfigJson,
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
	docSnaps, err := fireStore.Collection("data").Where("videoId", "==", request.QuizVideoId).Documents(context.Background()).GetAll()
	if err != nil {
		fmt.Printf("Error getting documents:%v\n", err)
		channel <- false
		return
	}
	if len(docSnaps) == 0 {
		channel <- false
		return
	}
	for _, docSnap := range docSnaps {
		_, err = fireStore.Collection("data").Doc(docSnap.Ref.ID).Update(context.Background(), []firestore.Update{
			{Path: "transcript", Value: request.Transcript},
		})
		if err != nil {
			fmt.Printf("Error updating transcript:%v\n", err)
			channel <- false
			return
		}
		// fmt.Printf("Success updating transcript\n")
	}

	channel <- true
}
