package server

import (
	"datastar-openrouter/internal/handlers"
	"datastar-openrouter/services"
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
	promptRouter := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Compress(5))
	router.Use(services.AuthorizationMiddleware)

	promptRouter.Use(middleware.ClientIPFromXFFTrustedProxies(1))
	promptRouter.Use(httprate.LimitBy(
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
	var uiSidMap sync.Map
	helperService := services.NewHelperService()
	mainHandler := handlers.NewMainHandler(&uiSidMap, helperService)
	fileHandler := handlers.NewFileHandler(&uiSidMap, helperService)
	sessionActionHandler := handlers.NewSessionActionHandler(&uiSidMap, helperService)
	sseHandler := handlers.NewSSEHandler(&uiSidMap, helperService)
	promptHandler := handlers.NewPromptHandler(&uiSidMap, helperService)

	router.Get("/", mainHandler.HandleMainPage)
	router.Get("/{sessionId}", mainHandler.HandleMainPage)
	router.Post("/sse", sseHandler.HandleLongSSE)
	router.Post("/new", sessionActionHandler.HandleNewChat)
	promptRouter.Post("/chat", promptHandler.HandlePrompt)
	router.Post("/session/delete", sessionActionHandler.HandleDeleteSession)
	router.Post("/sessions/search", sessionActionHandler.HandleSearchSessions)
	router.Post("/fileupload", fileHandler.HandleFileUpload)
	router.Post("/fileupload/remove", fileHandler.HandleRemoveUploadedFile)
	router.Post("/retry", promptHandler.HandleRetry)
	router.Post("/image", fileHandler.HandleGetImage)

	router.Get("/assets/*", func(responseWriter http.ResponseWriter, request *http.Request) {
		http.StripPrefix("/assets/", http.FileServer(http.Dir("assets/"))).ServeHTTP(responseWriter, request)
	})
	router.Mount("/", promptRouter)
	return router
}
