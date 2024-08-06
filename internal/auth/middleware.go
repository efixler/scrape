package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// The JWTAuthMiddleware will store claims in the request context at this key.
type ClaimsContextKey struct{}

// Use this interface to add extra authorization checks to the middleware.
type ClaimsAuthorizer func(claims *Claims) error

type config struct {
	tokenF           []func(*http.Request) string
	extraAuthorizers []ClaimsAuthorizer
}

type middlewareOption func(*config) error

// Add additional claims authorizers to the middleware
func WithClaimsAuthorizer(ca ...ClaimsAuthorizer) middlewareOption {
	return func(c *config) error {
		c.extraAuthorizers = append(c.extraAuthorizers, ca...)
		return nil
	}
}

// Accept a token passed via a cookie of the specified name.
func WithCookie(cookieName string) middlewareOption {
	return func(c *config) error {
		c.tokenF = append(c.tokenF, tokenFromCookie(cookieName))
		return nil
	}
}

// Checks the Authorization header for a JWT token and verifies it using
// the provided key.
// The token is always validated against the HMAC key, the issuer, and
// the Claims.Validate function.
//
// ClaimsAuthorizer functions, if any, are called in order. If any of them
// return an error, the request is rejected with a 401 Unauthorized status
// and the error message is written to the response body.
//
// If the token is valid, the claims are added to the request context at
// the key value of ClaimsContextKey{}.
func JWTAuthzMiddleware(
	key HMACBase64Key,
	options ...middlewareOption,
) func(http.HandlerFunc) http.HandlerFunc {
	cfg := &config{
		tokenF: []func(*http.Request) string{tokenFromHeader},
	}
	for _, opt := range options {
		if err := opt(cfg); err != nil {
			panic(err)
		}
	}
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var token string
			for _, f := range cfg.tokenF {
				token = f(r)
				if token != "" {
					break
				}
			}
			if token == "" {
				http.Error(w, "No Authorization Passed", http.StatusUnauthorized)
				return
			}
			claims, err := VerifyToken(key, strings.TrimSpace(token))
			if err != nil {
				msg := fmt.Sprintf("Invalid Token - %v", err)
				http.Error(w, msg, http.StatusUnauthorized)
				return
			}
			for _, c := range cfg.extraAuthorizers {
				if err := c(claims); err != nil {
					msg := fmt.Sprintf("Not authorized for this request: %v", err)
					http.Error(w, msg, http.StatusUnauthorized)
					return
				}
			}
			r = r.WithContext(context.WithValue(r.Context(), ClaimsContextKey{}, claims))
			next(w, r)
		}
	}
}

func tokenFromHeader(r *http.Request) string {
	_, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
	if !found {
		return ""
	}
	return token
}

func tokenFromCookie(cookieName string) func(*http.Request) string {
	return func(r *http.Request) string {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			return ""
		}
		return cookie.Value
	}
}
