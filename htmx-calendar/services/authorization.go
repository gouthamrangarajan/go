package services

import (
	"htmx-calendar/components"
	"net/http"
)

func MiddlewareUI(next func(w http.ResponseWriter, r *http.Request, token string)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// month := r.URL.Query().Get("month")
		// year := r.URL.Query().Get("year")
		path := r.URL.Path
		query := r.URL.RawQuery
		// cookie check
		cookie, err := r.Cookie("token")
		if err != nil {
			components.LoginPage(path, query).Render(r.Context(), w)
		} else {
			token := cookie.Value
			if token == "" {
				components.LoginPage(path, query).Render(r.Context(), w)
			} else {
				next(w, r, token)
			}
		}
	})
}

func MiddlewareJSON(next func(w http.ResponseWriter, r *http.Request, token string)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// cookie check
		cookie, err := r.Cookie("token")
		if err != nil {
			w.WriteHeader(401)
			w.Write([]byte("UnAuthorized"))
		} else {
			token := cookie.Value
			if token == "" {
				w.WriteHeader(401)
				w.Write([]byte("UnAuthorized"))
			} else {
				next(w, r, token)
			}
		}
	})
}
