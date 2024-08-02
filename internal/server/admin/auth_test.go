package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/efixler/scrape/internal/auth"
)

// TODO: Write tests for the "real" cases
func TestCheckAuth(t *testing.T) {
	tests := []struct {
		name           string
		claims         *auth.Claims
		signingKey     auth.HMACBase64Key
		authRequest    checkAuthRequest
		expectStatus   int
		expectCookie   bool
		expectResponse checkAuthResponse
	}{
		{
			name:         "no auth",
			signingKey:   auth.HMACBase64Key([]byte{}),
			expectStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		bodyJson, _ := json.Marshal(tt.authRequest)
		reader := bytes.NewReader(bodyJson)
		req := httptest.NewRequest("POST", "http://foo.bar/", reader)

		authzShim := authzShim(tt.signingKey)
		as := MustServer(nil, WithAuthz(authzShim))
		handler := as.checkAuthHandler()

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
	}
}
