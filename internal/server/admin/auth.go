package admin

import (
	"net/http"

	"github.com/efixler/scrape/internal/auth"
	"github.com/efixler/scrape/internal/server/middleware"
)

// _transitional_ handlers as we work towards implementing OIDC

//

type checkAuthResponse struct {
	Subject string `json:"subject"`
	Expires int    `json:"expires"`
}

type checkAuthRequest struct {
	EnableAdmin bool `json:"ea"`
}

func (as *adminServer) checkAuthHandler() http.HandlerFunc {
	ms := []middleware.Step{middleware.MaxBytes(4096)}

	//ms := ss.withAuthIfEnabled(MaxBytes(4096))
	return middleware.Chain(as.checkAuth, ms...)
}

func (as *adminServer) checkAuth(w http.ResponseWriter, r *http.Request) {
	if !as.authz.AuthEnabled() {
		w.WriteHeader(http.StatusNoContent)
	}
	claims, _ := r.Context().Value(auth.ClaimsContextKey{}).(*auth.Claims)
	ar := new(checkAuthResponse)
	ar.Subject = claims.Subject
	ar.Expires = int(claims.ExpiresAt.Time.Unix())
	middleware.WriteJSONOutput(w, ar, false, http.StatusOK)
}
