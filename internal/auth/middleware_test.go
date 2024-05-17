package auth

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestJWTAuthMiddleWare(t *testing.T) {
	t.Parallel()
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
		extra        []ClaimsAuthorizer
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
			extra:        []ClaimsAuthorizer{func(c *Claims) error { return nil }},
			expectStatus: http.StatusOK,
		},
		{
			name:         "With extra authorizer, reject",
			key:          realKey,
			authHeader:   fmt.Sprintf("Bearer %s", token),
			extra:        []ClaimsAuthorizer{func(c *Claims) error { return fmt.Errorf("nope") }},
			expectStatus: http.StatusUnauthorized,
		},
	}
	type contextKey struct{}
	for _, tt := range tests {
		req := httptest.NewRequest("GET", "http://example.com", nil)
		recorder := httptest.NewRecorder()
		req.Header.Set("Authorization", tt.authHeader)
		m := JWTAuthMiddleware(tt.key, contextKey{}, tt.extra...)

		m(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(contextKey{}).(*Claims)
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
