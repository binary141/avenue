package handlers

import (
	"log"
	"net/http"
	"strconv"

	"avenue/backend/persist"
	"avenue/backend/shared"

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

// setupRouter creates and configures the Gin router.
func SetupServer(p *persist.Persist) Server {
	r := gin.Default()

	return Server{
		fs:      afero.NewOsFs(),
		router:  r,
		persist: p,
	}
}

var (
	AUTHHEADER   = shared.GetEnv("AUTH_HEADER", "my-auth-header")
	AUTHKEY      = shared.GetEnv("AUTH_KEY", "MY-AUTH-VAL")
	USERIDHEADER = shared.GetEnv("USER_HEADER", "user-id")
)

func (s *Server) UserIDExists(userID string) bool {
	// todo do a lookup in the db and see if the user exists
	i, err := strconv.Atoi(userID)
	if err != nil {
		log.Print(err)
		return false
	}

	u, err := s.persist.GetUserById(i)
	if err != nil {
		log.Print(err)
		return false
	}

	log.Print(u)
	return true
}

func (s *Server) sessionCheck(c *gin.Context) {
	// if the auth header is present with the needed fields, we can allow them to bypass the cookie check :)
	if h := c.GetHeader(AUTHHEADER); h != "" {
		if u := c.GetHeader(USERIDHEADER); u != "" {
			if h == AUTHKEY && s.UserIDExists(u) {
				c.Next()
			}
		}
	}

	cookie, err := c.Cookie(COOKIENAME)
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	if cookie == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	c.Next()
}

func (s *Server) SetupRoutes() {
	unsecuredRouter := s.router.Group("")

	unsecuredRouter.GET("/ping", s.pingHandler)
	unsecuredRouter.POST("/login", s.Login)
	s.router.POST("/register", s.Register)

	securedRouterV1 := s.router.Group("/v1")
	securedRouterV1.Use(s.sessionCheck)

	securedRouterV1.POST("/upload", s.Upload)
	securedRouterV1.GET("/file/list", s.ListFiles)
	securedRouterV1.GET("/ping", s.pingHandler)

	// --- users routes --- //
	securedRouterV1.POST("/logout", s.Logout)
	securedRouterV1.GET("/user/profile", s.GetProfile)
	securedRouterV1.PUT("/user/profile", s.UpdateProfile)
	securedRouterV1.PATCH("/user/password", s.UpdatePassword)
	s.router.GET("/ping", s.pingHandler)
	// s.router.POST("/upload", s.Upload)
	s.router.GET("/file/list", s.ListFiles)
	s.router.GET("/file", s.GetFile)
}

func (s *Server) Run(address string) error {
	return s.router.Run(address)
}

// pingHandler is a simple handler to check if the server is running.
func (s *Server) pingHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}
