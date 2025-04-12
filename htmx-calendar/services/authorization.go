package services

import (
	"htmx-calendar/components"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

func MiddlewareUI(next func(w http.ResponseWriter, r *http.Request, token string)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		query := r.URL.RawQuery
		cookie, err := r.Cookie("token")
		if err != nil {
			components.LoginPage(path, query).Render(r.Context(), w)
		} else {
			token := cookie.Value
			parsedToken, _, err := jwt.NewParser().ParseUnverified(token, jwt.MapClaims{})
			//to do validate signature
			if err != nil {
				components.LoginPage(path, query).Render(r.Context(), w)
				return
			} else {
				issuer, err := parsedToken.Claims.GetIssuer()
				if err != nil || os.Getenv("SUPABASE_AUTH_ISSUER") != issuer {
					components.LoginPage(path, query).Render(r.Context(), w)
					return
				}
			}
			next(w, r, token)
		}
	})
}

func MiddlewareJSON(next func(w http.ResponseWriter, r *http.Request, token string)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("token")
		if err != nil {
			w.WriteHeader(401)
		} else {
			token := cookie.Value
			parsedToken, _, err := jwt.NewParser().ParseUnverified(token, jwt.MapClaims{})
			//to do validate signature
			if err != nil {
				w.WriteHeader(401)
				return
			} else {
				issuer, err := parsedToken.Claims.GetIssuer()
				if err != nil || os.Getenv("SUPABASE_AUTH_ISSUER") != issuer {
					w.WriteHeader(401)
					return
				}
			}
			next(w, r, token)
		}
	})
}
