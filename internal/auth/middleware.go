package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type Middleware func(http.HandlerFunc) http.HandlerFunc

//type AuthHandler func

func JWTAuthMiddleware(key HMACBase64Key, contextKey any) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			_, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
			if !found || (token == "") {
				http.Error(w, "No Authorization Passed", http.StatusUnauthorized)
				return
			}
			claims, err := VerifyToken(key, strings.TrimSpace(token))
			if err != nil {
				http.Error(
					w,
					fmt.Sprintf("Invalid Token %q: %v", token, err),
					http.StatusUnauthorized,
				)
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), contextKey, claims))
			next(w, r)
		}
	}
}
