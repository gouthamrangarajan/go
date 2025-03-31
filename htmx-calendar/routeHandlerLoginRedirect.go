package main

import (
	"htmx-calendar/components"
	"htmx-calendar/services"
	"net/http"
	"net/url"
	"os"
	"time"
)

type httpHandler func(w http.ResponseWriter, r *http.Request, token string, month string, year string, day string, isOob bool)

type routeConfig map[string]httpHandler

var loginRedirectRoutes = routeConfig{
	"/":    MainPageWithOob,
	"/add": AddPageWithOob,
}

func Login(responseWriter http.ResponseWriter, request *http.Request) {
	email := request.FormValue("email")
	password := request.FormValue("password")
	channel := make(chan services.LoginResponse)
	defer close(channel)
	go services.Login(services.LoginRequest{Email: email, Password: password}, channel)
	resp := <-channel
	if resp.ErrorCode != "" {
		components.LoginError().Render(request.Context(), responseWriter)
	} else {
		secure := true
		if os.Getenv("Env") == "Development" {
			secure = false
		}
		cookie := http.Cookie{
			Name:     "token",
			Value:    resp.AccessToken,
			Expires:  time.Now().Add(time.Duration(resp.ExpiresIn-120) * time.Second), //RG add expiry 2 mins lesser , expiresin is seconds
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(responseWriter, &cookie)
		path := request.FormValue("path")
		query := request.FormValue("query")
		values, err := url.ParseQuery(query)
		month := ""
		year := ""
		day := ""
		if err == nil {
			month = values.Get("month")
			year = values.Get("year")
			day = values.Get("day")
		}
		if loginRedirectRoutes[path] != nil {
			loginRedirectRoutes[path](responseWriter, request, resp.AccessToken, month, year, day, true)
		} else {
			responseWriter.WriteHeader(404)
			responseWriter.Write([]byte("Not Found"))
		}
	}
}
