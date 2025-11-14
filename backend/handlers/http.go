package handlers

import "github.com/gin-gonic/gin"

// Server holds dependencies for the HTTP server.
type Server struct {
	// Add dependencies here, e.g., a database connection
	router *gin.Engine
}

// setupRouter creates and configures the Gin router.
func SetupServer() Server {
	r := gin.Default()

	return Server{
		router: r,
	}
}
func (s *Server) SetupRoutes() {
	s.router.GET("/ping", s.pingHandler)
	s.router.POST("/upload", s.Upload)
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
