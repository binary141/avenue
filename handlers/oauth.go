package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"avenue/backend/db"
	"avenue/backend/sdk"
	"avenue/backend/shared"

	ksdk "github.com/binary141/keyring/sdk"
	"github.com/gin-gonic/gin"
)

// avenueClientName is the name keyring records this app under when it
// registers itself as an OAuth broker client.
const avenueClientName = "avenue"

// registerOAuthClient registers (or fetches the existing registration for)
// this app with keyring as an OAuth broker client, so its /auth/:provider
// endpoints can be called with a valid client_id/redirect_uri pair.
//
// CLIENT_UUID must be set to a stable value that's generated once and
// reused on every call — registering again with the same uuid returns the
// existing client instead of minting a duplicate. Without it there's no
// safe way to identify ourselves to keyring, so this fails loudly rather
// than silently skipping registration.
func (s *Server) registerOAuthClient(ctx context.Context) (*ksdk.OAuthClient, error) {
	clientUUID := shared.GetEnv("CLIENT_UUID", "")
	if clientUUID == "" {
		return nil, fmt.Errorf("CLIENT_UUID is not set")
	}

	registrationKey := shared.GetEnv("CLIENT_REGISTRATION_AUTH_KEY", "")
	redirectURI := strings.TrimRight(shared.GetEnv("AVENUE_PUBLIC_URL", "http://localhost:5173"), "/") + "/oauth/callback"

	return s.keyringClient("").CreateOAuthClient(ctx, registrationKey, clientUUID, avenueClientName, []string{redirectURI})
}

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
//
// keyring now requires every /auth/:provider/login request to carry a
// client_id/redirect_uri pair that matches a registered OAuth broker client,
// so this registers (or re-fetches) our client first.
func (s *Server) OAuthLogin(c *gin.Context) {
	provider := c.Param("provider")

	client, err := s.registerOAuthClient(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "", fmt.Errorf("register oauth client: %w", err))
		return
	}

	redirectURI := client.RedirectURIs[0]
	loginURL := fmt.Sprintf("%s/auth/%s/login?client_id=%s&redirect_uri=%s",
		shared.GetEnv("KEYRING_PUBLIC_URL", s.keyringURL),
		url.PathEscape(provider),
		url.QueryEscape(client.ClientID),
		url.QueryEscape(redirectURI),
	)
	c.Redirect(http.StatusFound, loginURL)
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
