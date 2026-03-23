package main

import (
	"encoding/json"
	"fmt"
	"google-drive-content-search/components"
	"google-drive-content-search/models"
	"google-drive-content-search/services"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/starfederation/datastar-go/datastar"
	"golang.org/x/crypto/bcrypt"
)

var idMap sync.Map

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	} else {
		fmt.Println("Loaded .env file successfully")
	}
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Compress(5))
	router.Use(services.AuthorizationMiddleware)

	router.Post("/login", func(responseWriter http.ResponseWriter, request *http.Request) {
		var clientSignals models.ClientSignals
		reqBody, err := io.ReadAll(request.Body)
		if err == nil {
			json.Unmarshal(reqBody, &clientSignals)
		}
		clientSignals.Token = strings.TrimSpace(clientSignals.Token)
		if clientSignals.Token != "" {
			err := bcrypt.CompareHashAndPassword([]byte(os.Getenv("TOKEN")), []byte(clientSignals.Token))
			if err == nil {
				cookie, err := services.GenerateUserIdCookie()
				if err != nil {
					sse := datastar.NewSSE(responseWriter, request)
					sse.PatchSignals([]byte("{errMsg:'Error validating token. Please try again later.'}"))
					return
				}
				http.SetCookie(responseWriter, &cookie)
				sse := datastar.NewSSE(responseWriter, request)
				sse.ExecuteScript("window.location.href=window.location.href;", datastar.WithExecuteScriptAutoRemove(true))
				return
			}
		}
		sse := datastar.NewSSE(responseWriter, request)
		sse.PatchSignals([]byte("{errMsg:'Invalid Token.'}"))
	})
	router.Get("/", func(responseWriter http.ResponseWriter, request *http.Request) {
		id := uuid.New().String()
		idMap.Store(id, make(chan []models.DocumentChunk))
		components.Main(id).Render(request.Context(), responseWriter)
	})
	router.Get("/sse", func(responseWriter http.ResponseWriter, request *http.Request) {

		var clientSignals models.ClientSignals
		datastar.ReadSignals(request, &clientSignals)

		sse := datastar.NewSSE(responseWriter, request)
		if session, exists := idMap.Load(clientSignals.Id); exists {
			for {
				select {
				case <-request.Context().Done():
					close(session.(chan []models.DocumentChunk))
					idMap.Delete(clientSignals.Id)
					return
				case dbResults := <-session.(chan []models.DocumentChunk):
					uiData := services.ConvertDocumentChunkCollectionToSearchResultCollection(dbResults)
					sse.PatchElementTempl(components.SearchResults(uiData), datastar.WithUseViewTransitions(true))
					markDownToHtmlChannel := make(chan string, len(uiData))
					defer close(markDownToHtmlChannel)
					for _, singleData := range uiData {
						go services.ConvertMarkdownToHtml(singleData.Id, []byte(singleData.FileContentMarkdown), markDownToHtmlChannel)
					}
					for range uiData {
						markdownToHtmlData := <-markDownToHtmlChannel
						if markdownToHtmlData != "" {
							select {
							case <-request.Context().Done():
								continue
							default:
								sse.PatchElements(markdownToHtmlData, datastar.WithUseViewTransitions(true))
								sse.ExecuteScript("window.mermaid.run()", datastar.WithExecuteScriptAutoRemove(true))
							}
						}
					}
					sse.PatchSignals([]byte("{searching: false}"))
				}
			}
		}
	})
	router.Post("/search", func(responseWriter http.ResponseWriter, request *http.Request) {
		var clientSignals models.ClientSignals
		datastar.ReadSignals(request, &clientSignals)
		clientSignals.Query = strings.TrimSpace(clientSignals.Query)
		if clientSignals.Query != "" {
			sse := datastar.NewSSE(responseWriter, request)
			sse.PatchSignals([]byte("{searching: true}"))
			dataSentToChannel := false
			embeddingRequest := models.VoyageEmbeddingRequest{
				Input: []string{clientSignals.Query},
			}
			embeddingChannel := make(chan models.VoyageEmbeddingResponse)
			defer close(embeddingChannel)
			go services.CallVoyageEmbedding(embeddingRequest, embeddingChannel)
			embeddingResponse := <-embeddingChannel
			if len(embeddingResponse.Data) > 0 {
				embeddingData := embeddingResponse.Data[0].Embedding
				searchChannel := make(chan []models.DocumentChunk)
				defer close(searchChannel)
				go services.SearchData(embeddingData, searchChannel)
				dbResults := <-searchChannel
				if session, exists := idMap.Load(clientSignals.Id); exists {
					session.(chan []models.DocumentChunk) <- dbResults
					dataSentToChannel = true
				}
			}
			if !dataSentToChannel {
				sse.PatchSignals([]byte("{searching: false}"))
			}
		}
	})
	router.Get("/assets/*", func(responseWriter http.ResponseWriter, request *http.Request) {
		fileServer := http.StripPrefix("/assets/", http.FileServer(http.Dir("assets")))
		fileServer.ServeHTTP(responseWriter, request)
	})
	http.ListenAndServe(":3000", router)
}
