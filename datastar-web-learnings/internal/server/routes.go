package server

import (
	"datastar-web-learnings/internal/handlers"
	"datastar-web-learnings/services"
	openrouter "datastar-web-learnings/services/open-router"
	"datastar-web-learnings/services/pinecone"
	"datastar-web-learnings/services/storage"
	"datastar-web-learnings/services/voyage"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

type Router struct {
	rateLimitSeconds  int
	rateLimitRequests int
}

func NewRouter() *Router {
	rateLimitSecondsStr := os.Getenv("RATE_LIMIT_SECONDS")
	rateLimitRequestsStr := os.Getenv("RATE_LIMIT_REQUESTS")

	rateLimitSeconds, err := strconv.Atoi(rateLimitSecondsStr)
	if err != nil {
		rateLimitSeconds = 5
	}
	rateLimitRequests, err := strconv.Atoi(rateLimitRequestsStr)
	if err != nil {
		rateLimitRequests = 10
	}

	return &Router{
		rateLimitSeconds:  rateLimitSeconds,
		rateLimitRequests: rateLimitRequests,
	}
}

func (r *Router) HttpHandler() http.Handler {
	router := chi.NewRouter()
	dataRouter := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Compress(5))
	router.Use(middleware.Recoverer)

	dataRouter.Use(middleware.ClientIPFromXFFTrustedProxies(1))
	dataRouter.Use(httprate.LimitBy(
		r.rateLimitRequests,
		time.Duration(r.rateLimitSeconds)*time.Second,
		func(request *http.Request) (string, error) {
			// Get the IP that middleware.RealIP has already verified
			ip := middleware.GetClientIP(request.Context())
			// Canonicalize handles IPv6 /64 subnets naturally
			// Fallback: If for some reason the middleware failed (local dev),
			// use the direct network address.
			if ip == "" {
				ip = request.RemoteAddr
			}
			return httprate.CanonicalizeIP(ip), nil
		},
		httprate.WithLimitHandler(func(responseWriter http.ResponseWriter, request *http.Request) {
			realIP := middleware.GetClientIP(request.Context())
			fmt.Printf("Blocked request from IP: %s\n", realIP)
			http.Error(responseWriter, "Too many requests.", http.StatusTooManyRequests)
		}),
	)) // 10 request in 5 seconds

	router.Get("/assets/*", func(responseWriter http.ResponseWriter, request *http.Request) {
		http.StripPrefix("/assets/", http.FileServer(http.Dir("assets/"))).ServeHTTP(responseWriter, request)
	})

	var sidMap = sync.Map{}
	var quizMap = sync.Map{}
	dbService := storage.NewDbService()
	yTService := services.NewYTService()
	helperService := services.NewHelperService(dbService)
	openRouterClient := openrouter.NewClient(helperService)
	voyageClient := voyage.NewClient()
	pineconeClient := pinecone.NewClient()

	mainHandler := handlers.NewMainHandler(&sidMap, &quizMap, helperService, openRouterClient, voyageClient,
		pineconeClient, dbService, yTService)

	router.Get("/", mainHandler.HandleLandingPage)
	router.Get("/add", mainHandler.HandleAddPage)
	router.Post("/add", mainHandler.HandleAddVideo)
	router.Post("/tags/ui", mainHandler.HandleTags)
	router.Post("/delete", mainHandler.HandleDeleteVideo)
	router.Get("/quiz", mainHandler.HandleLoadQuiz)

	dataRouter.Get("/sse", mainHandler.HandleSSE)
	dataRouter.Get("/data", mainHandler.HandleLoadMore)
	dataRouter.Get("/search", mainHandler.HandleSearch)
	dataRouter.Post("/quiz", mainHandler.HandleQuizGenerationVerifyAnswerAndPrevNext)
	router.Mount("/", dataRouter)
	return router
}
