package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"avenue/backend/persist"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type LoginRequest struct {
	Username string `json:"username" validate:"required,min=4,max=64"`
	Password string `json:"password" validate:"required,min=4,max=64"`
}

var validate = validator.New()

const (
	COOKIENAME  = "user-id"
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

	u, err := s.authorize(req.Username, req.Password)
	if err != nil {
		// for now send the error in the response 🤔
		c.AbortWithStatusJSON(http.StatusUnauthorized, Response{
			Error: err.Error(),
		})
		return
	}

	c.SetCookie(COOKIENAME, fmt.Sprintf("%d", u.ID), 3600, "/", "localhost", false, true)
	c.JSON(200, gin.H{"message": "OK"})
}

func (s *Server) authorize(username, password string) (persist.User, error) {
	user, err := s.persist.GetUserByUsername(username)
	if err != nil {
		return user, err
	}

	if user.Password != password {
		return user, errors.New("Password incorrect")
	}

	return user, nil
}

func (s *Server) Logout(c *gin.Context) {
	c.SetCookie(COOKIENAME, "", -1, "/", "localhost", false, true)
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
