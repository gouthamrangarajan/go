package services

import (
	"context"
	"htmx-calendar/components"
	"htmx-calendar/services/db"
	"net/http"
	"net/url"
	"os"
	"time"
)

var loginRedirectRoutes = map[string]func(w http.ResponseWriter, r *http.Request, month string, year string, dayOrWeek string, isOob bool){
	"/":    MonthPageWithOob,
	"/add": AddPageWithOob,
	"/wk":  WeekPageWithOob,
}

func Login(responseWriter http.ResponseWriter, request *http.Request) {
	email := request.FormValue("email")
	password := request.FormValue("password")
	channel := make(chan db.LoginResponse)
	defer close(channel)
	go db.Login(db.LoginRequest{Email: email, Password: password}, channel)
	resp := <-channel
	if resp.ErrorCode != "" {
		components.LoginError().Render(request.Context(), responseWriter)
	} else {
		secure := true
		if os.Getenv("ENV") == "Development" {
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
		week := ""
		if err == nil {
			month = values.Get("month")
			year = values.Get("year")
			day = values.Get("day")
			week = values.Get("week")
		}
		if loginRedirectRoutes[path] != nil {
			ctx := context.WithValue(request.Context(), TokenKey, resp.AccessToken)
			request = request.WithContext(ctx)
			if day == "" {
				loginRedirectRoutes[path](responseWriter, request, month, year, week, true)
			} else {
				loginRedirectRoutes[path](responseWriter, request, month, year, day, true)
			}
		} else {
			responseWriter.WriteHeader(404)
		}
	}
}
