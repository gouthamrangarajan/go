package middlewares

import (
	"context"
	"datastar-notes/components"
	"datastar-notes/services"
	"net/http"
	"strings"

	"github.com/starfederation/datastar-go/datastar"
)

func Authorization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		accessToken := services.GetAccessTokenFromRequest(request)
		path := request.URL.Path
		unauthorized := false
		skipCheck := false

		switch {
		case strings.HasPrefix(path, "/assets/"):
			skipCheck = true
		case path != "/login" && path != "/otp" && path != "/otp/verify" && accessToken == "":
			unauthorized = true
		case (path == "/login" || path == "/otp" || path == "/otp/verify") && accessToken == "":
			skipCheck = true
		}
		if skipCheck {
			next.ServeHTTP(responseWriter, request)
			return
		}

		if unauthorized {
			if request.Header.Get("datastar-request") == "true" {
				sse := datastar.NewSSE(responseWriter, request)
				sse.PatchElementTempl(components.LoginMainEl(), datastar.WithUseViewTransitions(true))
			} else {
				http.Redirect(responseWriter, request, "/login", http.StatusSeeOther)
			}
			return
		}
		ctx := context.WithValue(request.Context(), services.UserTokenKey, accessToken)
		request = request.WithContext(ctx)
		if request.URL.Path == "/login" || path == "/otp" || path == "/otp/verify" {
			if request.Header.Get("datastar-request") == "true" {
				sse := datastar.NewSSE(responseWriter, request)
				sse.PatchElementTempl(components.MainEl(), datastar.WithUseViewTransitions(true))
			} else {
				http.Redirect(responseWriter, request, "/", http.StatusSeeOther)
			}
			return
		}
		next.ServeHTTP(responseWriter, request)
	})
}
