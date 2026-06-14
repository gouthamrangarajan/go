package services

import (
	"htmx-calendar/components"
	"htmx-calendar/models"
	"htmx-calendar/services/db"
	"net/http"
	"os"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

var loginRedirectRoutes = map[string]func(w http.ResponseWriter, r *http.Request, to models.MonthYearDayWeekString, isOob bool){
	"/": MonthPageWithOob,
	// "/add": AddPageWithOob,
	"/wk": WeekPageWithOob,
}

func Login(responseWriter http.ResponseWriter, request *http.Request) {
	email := request.FormValue("email")
	password := request.FormValue("password")
	channel := make(chan db.LoginResponse)
	defer close(channel)
	go db.Login(db.LoginRequest{Email: email, Password: password}, channel)
	resp := <-channel
	if resp.ErrorCode != "" {
		sse := datastar.NewSSE(responseWriter, request)
		sse.PatchElementTempl(components.LoginError(), datastar.WithUseViewTransitions(true))
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
		sse := datastar.NewSSE(responseWriter, request)
		sse.PatchElementTempl(components.LoginSuccess(), datastar.WithUseViewTransitions(true))
		sse.ExecuteScript("window.location.reload()", datastar.WithExecuteScriptAutoRemove(true))
		// path := request.FormValue("path")
		// query := request.FormValue("query")
		// values, err := url.ParseQuery(query)
		// month := ""
		// year := ""
		// day := ""
		// week := ""
		// if err == nil {
		// 	month = values.Get("month")
		// 	year = values.Get("year")
		// 	day = values.Get("day")
		// 	week = values.Get("week")
		// }
		// if loginRedirectRoutes[path] != nil {
		// 	ctx := context.WithValue(request.Context(), TokenKey, resp.AccessToken)
		// 	request = request.WithContext(ctx)
		// 	to := models.MonthYearDayWeekString{Month: month, Year: year, Day: day, Week: week}
		// 	loginRedirectRoutes[path](responseWriter, request, to, true)
		// } else {
		// 	responseWriter.WriteHeader(404)
		// }
	}
}
