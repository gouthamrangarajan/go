package main

import (
	"bytes"
	"context"
	"datastar-portfolio/components"
	"datastar-portfolio/models"
	"datastar-portfolio/services"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/resend/resend-go/v3"
	"github.com/starfederation/datastar-go/datastar"
)

var emailRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
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

	router.Get("/", func(responseWriter http.ResponseWriter, request *http.Request) {
		featureChannel := make(chan []models.DemoItem)
		go services.GetFeaturedDemos(featureChannel)
		features := <-featureChannel
		close(featureChannel)
		component := components.Landing(features)
		component.Render(request.Context(), responseWriter)
	})
	router.Post("/contact", func(responseWriter http.ResponseWriter, request *http.Request) {
		var clientSignal models.ClientSignals
		err := datastar.ReadSignals(request, &clientSignal)
		if err != nil {
			fmt.Printf("Error reading client signals: %v\n", err.Error())
		}
		contactEmail := strings.ToLower(strings.TrimSpace(clientSignal.Email))
		contactMessage := strings.TrimSpace(clientSignal.Message)
		if contactEmail == "" || contactMessage == "" || len(contactMessage) < 5 ||
			!emailRegex.MatchString(contactEmail) {
			responseWriter.WriteHeader(http.StatusBadRequest)
			return
		}

		sse := datastar.NewSSE(responseWriter, request)
		emailStrBuffer := new(bytes.Buffer)
		err = components.Email(models.ClientSignals{Email: contactEmail, Message: contactMessage}).Render(context.Background(), emailStrBuffer)
		if err == nil {
			client := resend.NewClient(os.Getenv("RESEND_API_KEY"))

			params := &resend.SendEmailRequest{
				From:    os.Getenv("EMAIL_FROM"),
				To:      []string{os.Getenv("EMAIL_TO")},
				Html:    emailStrBuffer.String(),
				Subject: fmt.Sprintf("Contact Form: Message from %s", contactEmail),
			}

			_, err = client.Emails.Send(params)
		}
		if err != nil {
			// sse.
			fmt.Printf("Error in rendering email component or sending email: %v\n", err.Error())
			sse.PatchElementTempl(components.ContactSubmittedMessage("Sorry, there was an error processing your request. Please try again later.", true), datastar.WithUseViewTransitions(true))
			return
		}
		sse.PatchSignals([]byte("{contactFormDisabled:true}"))
		sse.PatchElementTempl(components.ContactSubmittedMessage("Thanks for reaching out! I will get back to you soon.", false), datastar.WithUseViewTransitions(true))
	})

	router.Get("/projects", func(responseWriter http.ResponseWriter, request *http.Request) {
		channel := make(chan []models.DemoItem)
		go services.GetAllDemos(channel, true)
		allProjects := <-channel
		close(channel)
		id := uuid.NewString()
		component := components.Projects(allProjects, id)
		component.Render(request.Context(), responseWriter)
	})
	router.Get("/sse", func(responseWriter http.ResponseWriter, request *http.Request) {
		var clientSignal models.ClientSignals
		err := datastar.ReadSignals(request, &clientSignal)
		if err != nil {
			fmt.Printf("Error reading client signals in sse: %v\n", err.Error())
		}
		sse := datastar.NewSSE(responseWriter, request)

		session := make(chan []models.DemoItem)
		idMap.Store(clientSignal.Id, session)

		for {
			select {
			case <-request.Context().Done():
				idMap.Delete(clientSignal.Id)
				return
			case searchResults := <-session:
				if sessionInMap, ok := idMap.Load(clientSignal.Id); !ok || sessionInMap != session {
					return
				}
				sse.PatchElementTempl(components.ProjectCardCollection(searchResults), datastar.WithUseViewTransitions(true))
				sse.PatchSignals([]byte("{filtering:false}"))
			}
		}

	})
	router.Post("/search", func(responseWriter http.ResponseWriter, request *http.Request) {
		var clientSignal models.ClientSignals
		err := datastar.ReadSignals(request, &clientSignal)
		if err != nil {
			fmt.Printf("Error reading client signals in search: %v\n", err.Error())
		}
		// fmt.Printf("Received search request with text: '%s' and service filter: %v\n", clientSignal.SrchTxt, clientSignal.ServiceFilter)
		sse := datastar.NewSSE(responseWriter, request)
		sse.PatchSignals([]byte("{filtering:true}"))
		dataSentToChannel := false
		clientSignal.SrchTxt = strings.TrimSpace(clientSignal.SrchTxt)
		if clientSignal.SrchTxt != "" {
			embeddingRequest := models.VoyageEmbeddingRequest{
				Input: []string{clientSignal.SrchTxt},
			}
			embeddingChannel := make(chan models.VoyageEmbeddingResponse)
			go services.CallVoyageEmbedding(embeddingRequest, embeddingChannel)
			embeddingResponse := <-embeddingChannel
			close(embeddingChannel)
			if len(embeddingResponse.Data) > 0 {
				searchChannel := make(chan []models.DemoItem)
				go services.SearchDemos(searchChannel, embeddingResponse.Data[0].Embedding, clientSignal.ServiceFilter)
				searchResults := <-searchChannel
				close(searchChannel)
				if session, exists := idMap.Load(clientSignal.Id); exists {
					session.(chan []models.DemoItem) <- searchResults
					dataSentToChannel = true
				}
				return
			}
		}
		getAllDataChannel := make(chan []models.DemoItem)
		go services.GetAllDemosWithServiceFilter(getAllDataChannel, clientSignal.ServiceFilter)
		allDemos := <-getAllDataChannel
		close(getAllDataChannel)
		if session, exists := idMap.Load(clientSignal.Id); exists {
			session.(chan []models.DemoItem) <- allDemos
			dataSentToChannel = true
		}
		if !dataSentToChannel {
			sse.PatchSignals([]byte("{filtering:false}"))
		}
	})

	router.Get("/assets/*", func(responseWriter http.ResponseWriter, request *http.Request) {
		http.StripPrefix("/assets/", http.FileServer(http.Dir("assets/"))).ServeHTTP(responseWriter, request)
	})
	http.ListenAndServe(":3000", router)
}
