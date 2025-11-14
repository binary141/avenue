package handlers

import (
	"avenue/backend/persist"

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
func (s *Server) SetupRoutes() {
	s.router.GET("/ping", s.pingHandler)
	s.router.POST("/upload", s.Upload)
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
