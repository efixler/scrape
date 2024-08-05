package admin

import (
	"log/slog"
	"net/http"

	"github.com/efixler/scrape/internal/auth"
	"github.com/efixler/scrape/internal/server/middleware"
)

const (
	TokenCookieName = "token"
)

type AuthzProvider interface {
	AuthEnabled() bool
	SigningKey() auth.HMACBase64Key
}

type authzShim auth.HMACBase64Key

func (a authzShim) AuthEnabled() bool {
	return len(a) > 0
}

func (a authzShim) SigningKey() auth.HMACBase64Key {
	return auth.HMACBase64Key(a)
}

// _transitional_ handlers as we work towards implementing OIDC

type checkAuthResponse struct {
	Subject string `json:"subject"`
	Expires int    `json:"expires"`
}

func (as *adminServer) tokenToCookieHandler() http.HandlerFunc {
	if !as.authz.AuthEnabled() {
		return noContent
	}
	return middleware.Chain(
		as.tokenToCookie,
		auth.JWTAuthzMiddleware(as.authz.SigningKey()),
	)
}

// This handler is a bridge login stub as/until OICD pieces come into place.
// It'll take the token that was used to authorize the request (in a prior middleware step)
// and put it in a cookie. The auth middleware (tbd) will respect the cookie on future requests
// as it respects the Authorization header now.
func (as *adminServer) tokenToCookie(w http.ResponseWriter, r *http.Request) {
	claims, _ := r.Context().Value(auth.ClaimsContextKey{}).(*auth.Claims)
	ar := new(checkAuthResponse)
	ar.Subject = claims.Subject
	ar.Expires = int(claims.ExpiresAt.Time.Unix())

	// Re-creating the token from claims. The token, in the authorization header,
	// was already used to authorize this request, but we're not going to pull it from there
	// to be safe, especially as things are in rapid flux.
	// Also, in the future, we'll probably want to tag the cookie tokens somehow to distinguish
	// them from the Authorization header tokens (for CSRF requirements, etc)
	token, err := claims.Sign(as.authz.SigningKey())
	if err != nil {
		http.Error(w, "Login failed", http.StatusInternalServerError)
		slog.Warn("Failed to sign token in login", "error", err, "claims", claims)
		return
	}

	cookie := http.Cookie{
		Name:     TokenCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  claims.ExpiresAt.Time,
	}
	http.SetCookie(w, &cookie)
	middleware.WriteJSONOutput(w, ar, false, http.StatusOK)
}

func noContent(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
