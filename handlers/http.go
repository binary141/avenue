package handlers

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"avenue/backend/db"
	"avenue/backend/logger"
	"avenue/backend/shared"

	ksdk "github.com/binary141/keyring/sdk"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/afero"
)

// Server holds dependencies for the HTTP server.
type Server struct {
	router     *gin.Engine
	fs         afero.Fs
	keyringURL string
}

// keyringClient returns a keyring SDK client for a single request, carrying
// token (which may be "" for anonymous calls like Login/Register). A fresh
// client is used per call rather than a shared one because the keyring SDK
// mutates its held token on Login/Logout/ExchangeOAuthCode, which would
// race across concurrent requests on a shared instance; NewClient does no
// network I/O, so this costs nothing.
func (s *Server) keyringClient(token string) *ksdk.Client {
	return ksdk.NewClient(s.keyringURL, ksdk.WithToken(token))
}

var (
	AUTHHEADER   = "Authorization"
	APPENV       = shared.GetEnv("APP_ENV", "production")
	COOKIEDOMAIN = shared.GetEnv("COOKIE_DOMAIN", "")
	COOKIESECURE = APPENV == "production"
)

// SessionRollingWindow is how long a session (and its cookies) stays valid
// after the most recent authenticated request.
const SessionRollingWindow = 15 * time.Minute

// FS returns the server's jailed filesystem, for callers outside the
// handlers package (e.g. the trash sweeper) that need to remove blobs.
func (s *Server) FS() afero.Fs {
	return s.fs
}

func SetupServer() Server {
	prod := APPENV == "production"
	if prod {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	if !prod {
		r.Use(gin.Logger())
	}

	fs := afero.NewOsFs()
	jailedFs := afero.NewBasePathFs(fs, shared.GetEnv("UPLOAD_DIR", "./avenuectl/temp/"))
	return Server{
		fs:         jailedFs,
		router:     r,
		keyringURL: shared.GetEnv("KEYRING_URL", "http://localhost:8081"),
	}
}

func (s *Server) ServeUI(uiFS embed.FS) {
	distFS, err := fs.Sub(uiFS, "dist")
	if err != nil {
		panic(err)
	}

	assetsFS, err := fs.Sub(distFS, "assets")
	if err != nil {
		panic(err)
	}

	// Serve only actual assets here
	s.router.StaticFS("/assets", http.FS(assetsFS))

	s.router.GET("/favicon.ico", func(c *gin.Context) {
		data, err := fs.ReadFile(distFS, "favicon.ico")
		if err != nil {
			c.String(http.StatusNotFound, "favicon not found")
			return
		}
		c.Data(http.StatusOK, "image/x-icon", data)
	})

	// SPA fallback
	s.router.NoRoute(func(c *gin.Context) {
		data, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "index.html not found")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})
}

// extractSessionToken pulls the raw session token out of an Authorization
// header (formatted "Token <token>") or, if that's empty, a "token" query
// param. Returns ("", false) if neither yields a well-formed token.
func extractSessionToken(authHeader, queryToken string) (string, bool) {
	h := authHeader
	if h == "" {
		if queryToken == "" {
			return "", false
		}
		h = fmt.Sprintf("Token %s", queryToken)
	}
	parts := strings.Split(h, "Token ")
	if len(parts) != 2 {
		return "", false
	}
	return parts[1], true
}

// requestSessionToken resolves the session token for a request: the
// Authorization header and (if allowQueryToken) the "token" query param take
// priority for non-browser API clients, falling back to the HttpOnly session
// cookie the browser attaches automatically. The cookie is what the SPA
// itself relies on — it never has JS-readable access to the raw token.
func requestSessionToken(c *gin.Context, allowQueryToken bool) (string, bool) {
	queryToken := ""
	if allowQueryToken {
		queryToken = c.Query("token")
	}

	if token, ok := extractSessionToken(c.GetHeader(AUTHHEADER), queryToken); ok {
		return token, true
	}

	if token, err := c.Cookie(string(shared.SESSIONCOOKIENAME)); err == nil && token != "" {
		return token, true
	}

	return "", false
}

// getAuthenticatedUserID extracts the local user ID from a token without
// requiring the session middleware. Returns (userID, true) if valid,
// (0, false) if not.
func (s *Server) getAuthenticatedUserID(c *gin.Context) (int64, bool) {
	token, ok := requestSessionToken(c, true)
	if !ok {
		return 0, false
	}
	me, err := s.keyringClient(token).MeWithToken(c.Request.Context(), token)
	if err != nil {
		return 0, false
	}
	localUser, err := db.GetOrCreateLocalUser(me.User.ID, me.User.UUID)
	if err != nil {
		return 0, false
	}
	return localUser.ID, true
}

func (s *Server) isAuthenticated(c *gin.Context) bool {
	token, ok := requestSessionToken(c, true)
	if !ok {
		return false
	}
	_, err := s.keyringClient(token).MeWithToken(c.Request.Context(), token)
	return err == nil
}

func (s *Server) sessionCheck(c *gin.Context) {
	s.sessionCheckImpl(c, false)
}

// sessionCheckAllowQueryToken behaves like sessionCheck but also accepts the
// session token via a "token" query param. This exists only for endpoints the
// UI reaches via direct browser navigation (file downloads/thumbnails), which
// can't attach an Authorization header. Query params can leak into logs,
// proxies, and Referer headers, so this must not be used as the default for
// the whole secured API.
func (s *Server) sessionCheckAllowQueryToken(c *gin.Context) {
	s.sessionCheckImpl(c, true)
}

func (s *Server) sessionCheckImpl(c *gin.Context, allowQueryToken bool) {
	token, ok := requestSessionToken(c, allowQueryToken)
	if !ok {
		if c.GetHeader(AUTHHEADER) != "" {
			logger.Warnf("sessionCheck: malformed Authorization header")
		}
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// Validate the token against keyring and fetch identity + roles. Keyring
	// is now the sole owner of session validity/expiry, so there's no local
	// session row to check or extend here.
	me, err := s.keyringClient(token).MeWithToken(c.Request.Context(), token)
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	localUser, err := db.GetOrCreateLocalUser(me.User.ID, me.User.UUID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", fmt.Errorf("resolve local user: %w", err))
		return
	}

	isAdmin := me.User.IsAdmin

	// keep the cookies' Max-Age in sync with the rolling session window,
	// otherwise the browser drops them well before keyring's session expires
	maxAge := int(SessionRollingWindow.Seconds())
	c.SetCookie(string(shared.USERCOOKIENAME), fmt.Sprint(localUser.ID), maxAge, "/", COOKIEDOMAIN, COOKIESECURE, true)
	c.SetCookie(string(shared.SESSIONCOOKIENAME), token, maxAge, "/", COOKIEDOMAIN, COOKIESECURE, true)

	rc := c.Request.Context()

	// Add a new value to the context
	newCtx := context.WithValue(rc, shared.USERCOOKIENAME, localUser.ID)
	// put the session token and admin flag into the context
	newCtx = context.WithValue(newCtx, shared.SESSIONCOOKIENAME, token)
	newCtx = context.WithValue(newCtx, shared.ISADMINCONTEXTKEY, isAdmin)

	// Update the request with the new context
	c.Request = c.Request.WithContext(newCtx)

	c.Next()
}

func (s *Server) fileSharingRequired(c *gin.Context) {
	if !shared.GetEnvBool("ENABLE_FILE_SHARING", false) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.Next()
}

func (s *Server) folderSharingRequired(c *gin.Context) {
	if !shared.GetEnvBool("ENABLE_FOLDER_SHARING", false) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.Next()
}

func (s *Server) SetupRoutes() {
	allowedOrigins := shared.GetEnv("ALLOWED_ORIGINS", "")

	logger.Errorf("%+v", allowedOrigins)

	originsList := strings.Split(allowedOrigins, ",")
	for i, o := range originsList {
		originsList[i] = strings.TrimSpace(o)
	}

	logger.Errorf("%+v", originsList)

	c := cors.Config{
		AllowOrigins: originsList,
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "content-type", "Accept", "Authorization", "authorization"},
		// The SPA now authenticates via its HttpOnly session cookie rather
		// than a JS-attached Authorization header, so cross-origin requests
		// (e.g. the Vite dev server on a different port) need the browser to
		// send that cookie, which requires AllowCredentials plus an explicit
		// origin allowlist (never "*") in ALLOWED_ORIGINS.
		AllowCredentials: true,
		ExposeHeaders:    []string{"Content-Length"},
		MaxAge:           12 * time.Hour,
	}

	s.router.Use(cors.New(c))

	unsecuredRouter := s.router.Group("")

	unsecuredRouter.GET("/ping", s.pingHandler)
	unsecuredRouter.POST("/login", s.Login)
	unsecuredRouter.GET("/loginMeta", s.LoginMeta)
	unsecuredRouter.POST("/register", s.Register)
	unsecuredRouter.POST("/verify-email", s.VerifyEmail)
	unsecuredRouter.POST("/forgot-password", s.ForgotPassword)
	unsecuredRouter.POST("/reset-password", s.ResetPassword)
	unsecuredRouter.GET("/auth/providers", s.ListOAuthProviders)
	unsecuredRouter.GET("/auth/:provider/login", s.OAuthLogin)
	unsecuredRouter.POST("/auth/exchange", s.OAuthExchange)
	publicFileShare := unsecuredRouter.Group("/api/share")
	publicFileShare.Use(s.fileSharingRequired)
	publicFileShare.GET("/:token", s.GetShareLinkMeta)
	publicFileShare.GET("/:token/download", s.DownloadSharedFile)
	publicFolderShare := unsecuredRouter.Group("/api/share/folder")
	publicFolderShare.Use(s.folderSharingRequired)
	publicFolderShare.GET("/:token", s.GetSharedFolderContents)
	publicFolderShare.GET("/:token/browse/:subFolderUUID", s.BrowseSharedSubFolder)
	publicFolderShare.GET("/:token/file/:fileUUID", s.DownloadSharedFolderFile)
	publicFolderShare.POST("/:token/upload", s.UploadToSharedFolder)

	securedRouterV1 := s.router.Group("/v1")
	securedRouterV1.Use(s.sessionCheck)

	securedRouterV1.GET("/ping", s.pingHandler)

	// -- meta routes -- //
	securedRouterV1.GET("/dashboard", s.DashboardInfo)

	// -- file routes -- //
	securedRouterV1.POST("/file", s.Upload)
	secureFileShare := securedRouterV1.Group("")
	secureFileShare.Use(s.fileSharingRequired)
	secureFileShare.POST("/file/:fileID/share", s.CreateShareLink)
	secureFileShare.GET("/file/:fileID/shares", s.ListFileShares)
	secureFileShare.GET("/shares", s.ListUserShares)
	secureFileShare.GET("/shares/expired", s.ListExpiredUserShares)
	secureFileShare.DELETE("/share/:token", s.RevokeShareLink)

	secureFolderShare := securedRouterV1.Group("")
	secureFolderShare.Use(s.folderSharingRequired)
	secureFolderShare.POST("/folder/:folderID/share", s.CreateFolderShareLink)
	secureFolderShare.GET("/folder/:folderID/shares", s.ListFolderShares)
	secureFolderShare.GET("/folder-shares", s.ListUserFolderShares)
	secureFolderShare.GET("/folder-shares/expired", s.ListExpiredUserFolderShares)
	secureFolderShare.DELETE("/share/folder/:token", s.RevokeShareFolderLink)

	securedRouterV1.GET("/file/list", s.ListFiles)
	securedRouterV1.GET("/folder/files/:fileName", s.SearchFiles) // search root folder
	securedRouterV1.GET("/folder/:folderID/files/:fileName", s.SearchFiles)

	// these two are reached via direct browser navigation (img src / <a> download
	// links) rather than fetch(), so they can't send an Authorization header and
	// need to accept the session token as a query param instead.
	downloadRoutesV1 := s.router.Group("/v1")
	downloadRoutesV1.Use(s.sessionCheckAllowQueryToken)
	downloadRoutesV1.GET("/file/:fileID", s.GetFile)
	downloadRoutesV1.GET("/files/zip", s.DownloadFilesZip)

	securedRouterV1.PATCH("/file/:fileID/move", s.MoveFile)
	securedRouterV1.PATCH("/file/:fileID/:fileName", s.UpdateFileName)
	securedRouterV1.DELETE("/file/:fileID", s.DeleteFile)
	securedRouterV1.PATCH("/file/:fileID/restore", s.RestoreFile)
	securedRouterV1.DELETE("/file/:fileID/purge", s.PurgeFile)
	securedRouterV1.DELETE("/files/bulk-delete", s.BulkDelete)
	securedRouterV1.PATCH("/files/bulk-restore", s.BulkRestore)
	securedRouterV1.PATCH("/files/bulk-move", s.BulkMove)

	// -- folder routes -- //
	securedRouterV1.POST("/folder", s.CreateFolder)
	securedRouterV1.DELETE("/folder/:folderID", s.DeleteFolder)
	securedRouterV1.PATCH("/folder/:folderID/:folderName", s.UpdateFolderName)
	securedRouterV1.GET("/folder/list/", s.ListFolderContents) // for use for getting the root folder
	securedRouterV1.GET("/folder/list/:folderID", s.ListFolderContents)
	securedRouterV1.PATCH("/folder/:folderID/restore", s.RestoreFolder)
	securedRouterV1.DELETE("/folder/:folderID/purge", s.PurgeFolder)

	// -- trash routes -- //
	securedRouterV1.GET("/trash", s.ListTrash)
	securedRouterV1.POST("/trash/empty", s.EmptyTrash)

	// --- users routes --- //
	securedRouterV1.POST("/logout", s.Logout)
	securedRouterV1.GET("/user/profile", s.GetProfile)
	securedRouterV1.GET("/users", s.GetUsers)
	securedRouterV1.POST("/user", s.CreateUser) // todo might be able to remove this route and have the ui do some work
	securedRouterV1.POST("/registration-tokens", s.CreateRegistrationToken)
	securedRouterV1.GET("/registration-tokens", s.ListRegistrationTokens)
	securedRouterV1.PUT("/user/profile", s.UpdateProfile)
	securedRouterV1.PATCH("/user/:userID", s.UpdateProfile)
	securedRouterV1.PATCH("/user/password", s.UpdatePassword)
	securedRouterV1.POST("/user/:userID/send-reset-email", s.AdminSendPasswordReset)
	securedRouterV1.GET("/user/sessions", s.ListSessions)
	securedRouterV1.DELETE("/user/sessions/:sessionID", s.RevokeSession)
	securedRouterV1.DELETE("/user/sessions", s.RevokeOtherSessions)

}

func (s *Server) Run(address string) error {
	return s.router.Run(address)
}

// pingHandler is a simple handler to check if the server is running.
func (s *Server) pingHandler(c *gin.Context) {
	ctx := c.Request.Context()
	logger.Debugf("ctx val: %s", ctx.Value(shared.USERCOOKIENAME))
	respond(c, 200, "pong", nil)
}
