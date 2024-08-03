package admin

import (
	"log/slog"
	"net/http"

	"github.com/efixler/scrape/internal/auth"
	"github.com/efixler/scrape/internal/server/middleware"
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

//

type payloadKey struct{}

type checkAuthResponse struct {
	Subject string `json:"subject"`
	Expires int    `json:"expires"`
}

type checkAuthRequest struct {
	Login bool `json:"login"`
}

func (as *adminServer) checkAuthHandler() http.HandlerFunc {
	if !as.authz.AuthEnabled() {
		return noContent
	}
	return middleware.Chain(
		as.checkAuth,
		auth.JWTAuthMiddleware(as.authz.SigningKey()),
		middleware.MaxBytes(4096),
		middleware.DecodeJSONBody[checkAuthRequest](payloadKey{}))
}

// This handler is a bridge login stub as/until OICD pieces come into place.
// It'll take the token that was used to authorize the request (in a prior middleware step)
// and put it in a cookie. The auth middleware (tbd) will respect the cookie on future requests
// as it respects the Authorization header now.
func (as *adminServer) checkAuth(w http.ResponseWriter, r *http.Request) {
	claims, _ := r.Context().Value(auth.ClaimsContextKey{}).(*auth.Claims)
	ar := new(checkAuthResponse)
	ar.Subject = claims.Subject
	ar.Expires = int(claims.ExpiresAt.Time.Unix())
	req, _ := r.Context().Value(payloadKey{}).(*checkAuthRequest)
	if req.Login {
		// Re-creating the token from claims. The token, in the auothorization header,
		// was already used to authorize this request, but we're not going to pull it from there
		// to be safe, especially as things are in rapid flux.
		// Ideally, we don't want to be re-making the token, as long as we can guarantee
		// that the input token is valid and not tamperable by the client.
		token, err := claims.Sign(as.authz.SigningKey())
		if err != nil {
			http.Error(w, "Login failed", http.StatusInternalServerError)
			slog.Warn("Failed to sign token in login", "error", err, "claims", claims)
			return
		}

		cookie := http.Cookie{
			Name:     "jwt",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Expires:  claims.ExpiresAt.Time,
		}
		http.SetCookie(w, &cookie)
	}
	middleware.WriteJSONOutput(w, ar, false, http.StatusOK)
}

func noContent(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
