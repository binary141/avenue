package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type LoginRequest struct {
	Username string `json:"username" validate:"required,min=4,max=64"`
	Password string `json:"password" validate:"required,min=4,max=64"`
}

var validate = validator.New()

const (
	COOKIENAME  = "my-cookie"
	COOKIEVALUE = "test"
)

func (s *Server) Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	if err := validate.Struct(req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, Response{
			Error: err.Error(),
		})
		return
	}

	err := authorize(req.Username, req.Password)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, Response{
			Error: err.Error(),
		})
		return
	}

	c.SetCookie(COOKIENAME, COOKIEVALUE, 3600, "/", "localhost", false, true)
	c.JSON(200, gin.H{"message": "OK"})
}

func authorize(username, password string) error {
	if username != "root" {
		return errors.New("Invalid Username")
	}

	if password != "password" {
		return errors.New("Invalid Password")
	}

	return nil
}

func (s *Server) Logout(c *gin.Context) {
	c.SetCookie(COOKIENAME, COOKIEVALUE, -1, "/", "localhost", false, true)
	c.JSON(200, gin.H{"message": "OK"})
}

func (s *Server) Register(c *gin.Context) {
	c.JSON(200, gin.H{"message": "OK"})
}

func (s *Server) GetProfile(c *gin.Context) {
	c.JSON(200, gin.H{"message": "OK-1"})
}

func (s *Server) UpdateProfile(c *gin.Context) {
	c.JSON(200, gin.H{"message": "OK"})
}

func (s *Server) UpdatePassword(c *gin.Context) {
	c.JSON(200, gin.H{"message": "OK"})
}
