package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/efixler/scrape/internal/auth"
)

// TODO: Write tests for the "real" cases
func TestCheckAuth(t *testing.T) {
	tests := []struct {
		name         string
		signingKey   auth.HMACBase64Key
		authRequest  checkAuthRequest
		expectStatus int
		expectCookie bool
	}{
		{
			name:         "no auth",
			signingKey:   auth.HMACBase64Key([]byte{}),
			expectStatus: http.StatusNoContent,
		},
		{
			name:         "valid token, expect cookie",
			signingKey:   auth.MustNewHS256SigningKey(),
			authRequest:  checkAuthRequest{Login: true},
			expectStatus: http.StatusOK,
			expectCookie: true,
		},
		{
			name:         "valid token, expect no cookie",
			signingKey:   auth.MustNewHS256SigningKey(),
			authRequest:  checkAuthRequest{Login: false},
			expectStatus: http.StatusOK,
			expectCookie: false,
		},
	}

	for _, tt := range tests {
		bodyJson, _ := json.Marshal(tt.authRequest)
		reader := bytes.NewReader(bodyJson)
		req := httptest.NewRequest("POST", "http://foo.bar/", reader)

		authzShim := authzShim(tt.signingKey)
		as := MustServer(nil, WithAuthz(authzShim))
		handler := as.checkAuthHandler()
		var token string
		if authzShim.AuthEnabled() {
			claims, _ := auth.NewClaims(
				auth.WithSubject("tester"),
				auth.WithAudience("testing"),
				auth.ExpiresIn(60*time.Second),
			)
			token, _ = claims.Sign(authzShim.SigningKey())
			req.Header.Set("Authorization", "Bearer "+token)
		}

		w := httptest.NewRecorder()
		handler(w, req)
		resp := w.Result()
		if resp.StatusCode != tt.expectStatus {
			t.Errorf("[%s] Expected %d, got %d", tt.name, tt.expectStatus, resp.StatusCode)
			continue
		}
		if resp.StatusCode != 200 {
			continue
		}
		if tt.expectCookie {
			foundCookie := false
			for _, cookie := range resp.Cookies() {
				if cookie.Name == "jwt" {
					foundCookie = true
					if !cookie.HttpOnly {
						t.Errorf("[%s] Expected HttpOnly cookie, got none", tt.name)
					}
					if cookie.SameSite != http.SameSiteStrictMode {
						t.Errorf("[%s] Expected SameSite=Strict cookie, got %v", tt.name, cookie.SameSite)
					}
					if cookie.Value != token {
						t.Errorf("[%s] Expected cookie value to match token, got %s", tt.name, cookie.Value)
					}
					break
				}
			}
			if !foundCookie {
				t.Errorf("[%s] Expected cookie, got none", tt.name)
			}
		}
		decoder := json.NewDecoder(resp.Body)
		decoder.DisallowUnknownFields()
		ar := new(checkAuthResponse)
		err := decoder.Decode(ar)
		if err != nil {
			t.Errorf("[%s] Error decoding response: %s", tt.name, err)
		}
		if ar.Subject == "" {
			t.Errorf("[%s] Expected non-empty subject, got %s", tt.name, ar.Subject)
		}
		if ar.Expires == 0 {
			t.Errorf("[%s] Expected non-zero expires, got %d", tt.name, ar.Expires)
		}
	}
}
