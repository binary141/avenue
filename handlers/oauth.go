package handlers

import (
	"fmt"
	"net/http"
	"net/url"

	"avenue/backend/db"
	"avenue/backend/sdk"
	"avenue/backend/shared"

	"github.com/gin-gonic/gin"
)

// ListOAuthProviders reports the OAuth providers keyring is currently
// configured to accept logins from, so the login page can render only
// buttons that actually work.
func (s *Server) ListOAuthProviders(c *gin.Context) {
	providers, err := s.keyringClient("").ListProviders(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "", fmt.Errorf("list oauth providers: %w", err))
		return
	}
	c.JSON(http.StatusOK, sdk.V1OAuthProvidersResponse{Providers: providers})
}

// OAuthLogin sends the browser on to keyring to start the named provider's
// authorization flow. This has to be a top-level browser navigation (not a
// fetch()) since it ends in a redirect to the provider's consent screen.
func (s *Server) OAuthLogin(c *gin.Context) {
	provider := c.Param("provider")
	c.Redirect(http.StatusFound, shared.GetEnv("KEYRING_PUBLIC_URL", s.keyringURL)+"/auth/"+url.PathEscape(provider)+"/login")
}

// OAuthExchange redeems the one-time code keyring's callback redirected the
// browser back with, for a real session — mirroring Login's behavior.
func (s *Server) OAuthExchange(c *gin.Context) {
	var req sdk.OAuthExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	resp, err := s.keyringClient("").ExchangeOAuthCode(c.Request.Context(), req.Code)
	if err != nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	localUser, err := db.GetOrCreateLocalUser(resp.User.ID, resp.User.UUID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", fmt.Errorf("resolve local user: %w", err))
		return
	}

	maxAge := int(SessionRollingWindow.Seconds())
	c.SetCookie(string(shared.USERCOOKIENAME), fmt.Sprintf("%d", localUser.ID), maxAge, "/", COOKIEDOMAIN, COOKIESECURE, true)
	c.SetCookie(string(shared.SESSIONCOOKIENAME), resp.Token, maxAge, "/", COOKIEDOMAIN, COOKIESECURE, true)

	c.JSON(http.StatusOK, sdk.V1LoginResponse{
		Message:   "OK",
		UserID:    localUser.ID,
		SessionID: resp.Token,
		UserData:  mapKeyringUser(resp.User, localUser),
	})
}
