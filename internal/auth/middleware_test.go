package auth

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestJWTAuthMiddleWare(t *testing.T) {
	realKey := MustNewHS256SigningKey()
	c, _ := NewClaims(
		ExpiresAt(time.Now().Add(24*time.Hour)),
		WithSubject("subject"),
		WithAudience("audience"),
	)
	token, _ := c.Sign(realKey)
	tests := []struct {
		name         string
		key          HMACBase64Key
		authHeader   string
		options      []middlewareOption
		cookies      []http.Cookie
		expectStatus int
	}{
		{
			name:         "no auth",
			key:          realKey,
			authHeader:   "",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "valid auth",
			key:          realKey,
			authHeader:   fmt.Sprintf("Bearer %s", token),
			expectStatus: http.StatusOK,
		},
		{
			name:         "No token in header",
			key:          realKey,
			authHeader:   "Bearer",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Garbage token in header",
			key:          realKey,
			authHeader:   "Bearer llkllKjLKDLD.kkajhdakjsdhakdjh.ajkshdakjshd",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Only token in header",
			key:          realKey,
			authHeader:   token,
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Key mismatch",
			key:          MustNewHS256SigningKey(),
			authHeader:   fmt.Sprintf("Bearer %s", token),
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "With extra authorizer, passthru",
			key:          realKey,
			authHeader:   fmt.Sprintf("Bearer %s", token),
			options:      []middlewareOption{WithClaimsAuthorizer(func(c *Claims) error { return nil })},
			expectStatus: http.StatusOK,
		},
		{
			name:         "With extra authorizer, reject",
			key:          realKey,
			authHeader:   fmt.Sprintf("Bearer %s", token),
			options:      []middlewareOption{WithClaimsAuthorizer(func(c *Claims) error { return errors.New("reject") })},
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "With valid cookie and no header",
			key:          realKey,
			options:      []middlewareOption{WithCookie("jwt")},
			cookies:      []http.Cookie{{Name: "jwt", Value: token}},
			expectStatus: http.StatusOK,
		},
		{
			name:         "With cookie config but no cookie",
			key:          realKey,
			options:      []middlewareOption{WithCookie("jwt")},
			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", "http://example.com", nil)
		for _, cookie := range tt.cookies {
			req.AddCookie(&cookie)
		}
		recorder := httptest.NewRecorder()
		req.Header.Set("Authorization", tt.authHeader)
		m := JWTAuthzMiddleware(tt.key, tt.options...)

		m(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(ClaimsContextKey{}).(*Claims)
			if !ok {
				t.Fatalf("[%s] JWTAuthMiddleware, expected claims, got %v", tt.name, claims)
			}
		})(recorder, req)

		response := recorder.Result()
		if response.StatusCode != tt.expectStatus {
			body, _ := io.ReadAll(response.Body)
			t.Fatalf("[%s] JWTAuthMiddleware, expected status %d, got %d (%s)", tt.name, tt.expectStatus, response.StatusCode, body)
		}
	}
}
