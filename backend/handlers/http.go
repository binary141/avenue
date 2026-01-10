package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"avenue/backend/persist"
	"avenue/backend/shared"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/afero"
)

// Server holds dependencies for the HTTP server.
type Server struct {
	// Add dependencies here, e.g., a database connection
	router  *gin.Engine
	persist *persist.Persist
	fs      afero.Fs
}

func SetupServer(p *persist.Persist) Server {
	r := gin.Default()
	fs := afero.NewOsFs()
	jailedFs := afero.NewBasePathFs(fs, "./avenuectl/temp/")
	return Server{
		fs:      jailedFs,
		router:  r,
		persist: p,
	}
}

var (
	MASTERAUTHHEADER = shared.GetEnv("AUTH_HEADER", "my-auth-header")
	AUTHHEADER       = "Authorization"
	AUTHKEY          = shared.GetEnv("AUTH_KEY", "MY-AUTH-VAL")
	USERIDHEADER     = shared.GetEnv("USER_HEADER", "user-id")
)

func (s *Server) UserIDExists(userID string) bool {
	// todo do a lookup in the db and see if the user exists
	i, err := strconv.Atoi(userID)
	if err != nil {
		log.Print(err)
		return false
	}

	_, err = s.persist.GetUserById(i)
	if err != nil {
		log.Print(err)
		return false
	}

	return true
}

func (s *Server) sessionCheck(c *gin.Context) {
	// if the auth header is present with the needed fields, we can allow them to bypass the cookie check :)
	if h := c.GetHeader(MASTERAUTHHEADER); h != "" {
		if u := c.GetHeader(USERIDHEADER); u != "" {
			if h == AUTHKEY && s.UserIDExists(u) {

				rc := c.Request.Context()

				// Add a new value to the context
				newCtx := context.WithValue(rc, shared.USERCOOKIENAME, u)

				// Update the request with the new context
				c.Request = c.Request.WithContext(newCtx)
				c.Next()
				return
			}
		}
	}

	h := c.GetHeader(AUTHHEADER)
	if h == "" {
		// use a query param as a default
		// this is used in places where we can't send a cookie value (such as browser downloads)
		q := c.Query("token")
		if q == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		h = fmt.Sprintf("Token %s", q)
	}

	parts := strings.Split(h, "Token ")

	if len(parts) != 2 {
		log.Print("Not enough parts")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// see if the session is valid
	session, valid := s.persist.IsValidSession(parts[1])
	if !valid {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	if !s.UserIDExists(fmt.Sprint(session.UserId)) {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	session.ExpiresAt = time.Now().Add(15 * time.Minute).Unix()

	// update the session to be a rolling timeout
	_, err := s.persist.UpdateSession(session)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rc := c.Request.Context()

	// Add a new value to the context
	newCtx := context.WithValue(rc, shared.USERCOOKIENAME, session.UserId)
	// put the session data into the context
	newCtx = context.WithValue(newCtx, shared.SESSIONCOOKIENAME, parts[1])

	// Update the request with the new context
	c.Request = c.Request.WithContext(newCtx)

	c.Next()
}

func (s *Server) SetupRoutes() {
	c := cors.Config{
		AllowOrigins:     []string{shared.GetEnv("ALLOW_ORIGIN", "http://localhost:5173"), "http://localhost:8080"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "content-type", "Accept", "Authorization", "authorization"},
		AllowCredentials: false,
		ExposeHeaders:    []string{"Content-Length"},
		MaxAge:           12 * time.Hour,
	}

	s.router.Use(cors.New(c))

	unsecuredRouter := s.router.Group("")

	unsecuredRouter.GET("/ping", s.pingHandler)
	unsecuredRouter.POST("/login", s.Login)
	unsecuredRouter.POST("/register", s.Register)

	securedRouterV1 := s.router.Group("/v1")
	securedRouterV1.Use(s.sessionCheck)

	securedRouterV1.GET("/ping", s.pingHandler)

	// -- file routes -- //
	securedRouterV1.POST("/file", s.Upload)
	securedRouterV1.GET("/file/list", s.ListFiles)
	securedRouterV1.GET("/file/:fileID", s.GetFile)
	securedRouterV1.PATCH("/file/:fileID/:fileName", s.UpdateFileName)
	securedRouterV1.DELETE("/file/:fileID", s.DeleteFile)

	// -- folder routes -- //
	securedRouterV1.POST("/folder", s.CreateFolder)
	securedRouterV1.DELETE("/folder/:folderID", s.DeleteFolder)
	securedRouterV1.GET("/folder/list/", s.ListFolderContents) // for use for getting the root folder
	securedRouterV1.GET("/folder/list/:folderID", s.ListFolderContents)

	// --- users routes --- //
	securedRouterV1.POST("/logout", s.Logout)
	securedRouterV1.GET("/user/profile", s.GetProfile)
	securedRouterV1.GET("/users", s.GetUsers)
	securedRouterV1.POST("/user", s.CreateUser) // todo might be able to remove this route and have the ui do some work
	securedRouterV1.PUT("/user/profile", s.UpdateProfile)
	securedRouterV1.PATCH("/user/password", s.UpdatePassword)
}

func (s *Server) Run(address string) error {
	return s.router.Run(address)
}

// pingHandler is a simple handler to check if the server is running.
func (s *Server) pingHandler(c *gin.Context) {
	ctx := c.Request.Context()
	log.Printf("ctx val: %s", ctx.Value(shared.USERCOOKIENAME))
	c.JSON(200, gin.H{
		"message": "pong",
	})
}
