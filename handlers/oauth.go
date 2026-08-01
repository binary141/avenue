package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"avenue/backend/db"
	"avenue/backend/sdk"
	"avenue/backend/shared"

	"github.com/gin-gonic/gin"
)

// avenueClientName is the name this app presents to keyring's OAuth broker,
// matching the "name" field in keyring's client_apps/avenue.json.
const avenueClientName = "avenue"

// avenueRedirectURI is the redirect_uri this app presents to keyring's OAuth
// broker. It must exactly match an entry in keyring's client_apps/avenue.json
// redirectUris allowlist — keyring no longer supports registering a client
// over the network, so there's nothing to reconcile here at request time.
func avenueRedirectURI() string {
	return strings.TrimRight(shared.GetEnv("AVENUE_PUBLIC_URL", "http://localhost:5173"), "/") + "/oauth/callback"
}

// avenueClientUUID is this app's stable identity, generated once and baked
// into keyring's client_apps/avenue.json alongside avenueClientName.
func avenueClientUUID() (string, error) {
	clientUUID := shared.GetEnv("CLIENT_UUID", "")
	if clientUUID == "" {
		return "", fmt.Errorf("CLIENT_UUID is not set")
	}
	return clientUUID, nil
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
// keyring identifies clients by a client_name/client_uuid pair provisioned
// out-of-band in its client_apps/ directory, so this app presents its own
// stable identity rather than registering itself over the network.
func (s *Server) OAuthLogin(c *gin.Context) {
	provider := c.Param("provider")

	clientUUID, err := avenueClientUUID()
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}

	loginURL := fmt.Sprintf("%s/auth/%s/login?client_name=%s&client_uuid=%s&redirect_uri=%s",
		shared.GetEnv("KEYRING_PUBLIC_URL", s.keyringURL),
		url.PathEscape(provider),
		url.QueryEscape(avenueClientName),
		url.QueryEscape(clientUUID),
		url.QueryEscape(avenueRedirectURI()),
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
