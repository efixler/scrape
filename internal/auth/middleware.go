package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type ClaimsAuthorizer func(claims *Claims) error

// Checks the Authorization header for a JWT token and verifies it using the provided key.
// The token is always validated against the HMAC key, the issuer, and the Claims.Validate
// function.
//
// The ClaimsAuthorizer functions, if any, are called in order. If any of them return an
// error, the request is rejected with a 401 Unauthorized status and the error message
// is written to the response body.
//
// If the token is valid, the claims are added to the request context at the key value
// specified by contextKey.
func JWTAuthMiddleware(
	key HMACBase64Key,
	contextKey any,
	cc ...ClaimsAuthorizer,
) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			_, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
			if !found || (token == "") {
				http.Error(w, "No Authorization Passed", http.StatusUnauthorized)
				return
			}
			claims, err := VerifyToken(key, strings.TrimSpace(token))
			if err != nil {
				msg := fmt.Sprintf("Invalid Token - %v", err)
				http.Error(w, msg, http.StatusUnauthorized)
				return
			}
			for _, c := range cc {
				if err := c(claims); err != nil {
					msg := fmt.Sprintf("Not authorized for this request: %v", err)
					http.Error(w, msg, http.StatusUnauthorized)
					return
				}
			}
			r = r.WithContext(context.WithValue(r.Context(), contextKey, claims))
			next(w, r)
		}
	}
}
