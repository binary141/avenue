package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"avenue/backend/persist"
	"avenue/backend/shared"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type LoginRequest struct {
	Username string `json:"username" validate:"required,min=4,max=64"`
	Password string `json:"password" validate:"required,min=4,max=64"`
}

var validate = validator.New()

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

	c.SetCookie(shared.USERCOOKIENAME, fmt.Sprintf("%d", u.ID), 3600, "/", "localhost", false, true)
	c.JSON(http.StatusOK, Response{Message: "OK"})
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
	// expire the cookie
	c.SetCookie(shared.USERCOOKIENAME, "", -1, "/", "localhost", false, true)

	c.JSON(http.StatusOK, Response{Message: "OK"})
}

type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=4,max=64"`
	Password string `json:"password" validate:"required,min=4,max=64"`
	Email    string `json:"email" validate:"required,min=4,max=512"`
}

func (s *Server) Register(c *gin.Context) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Print(err)
		c.Status(http.StatusBadRequest)
		return
	}

	if err := validate.Struct(req); err != nil {
		log.Print(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, Response{
			Error: err.Error(),
		})
		return
	}

	if !shared.IsValidEmail(req.Email) {
		c.AbortWithStatusJSON(http.StatusBadRequest, Response{
			Error: "Email is not valid",
		})
		return
	}

	if !s.persist.IsUniqueUsername(req.Username) {
		c.AbortWithStatusJSON(http.StatusConflict, Response{
			Error: "Username already exists",
		})
		return
	}

	if !s.persist.IsUniqueEmail(req.Email) {
		c.AbortWithStatusJSON(http.StatusConflict, Response{
			Error: "Email already exists",
		})
		return
	}

	err := s.persist.CreateUser(req.Username, req.Email, req.Password)
	if err != nil {
		log.Print(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, Response{Message: "OK"})
}

func (s *Server) GetProfile(c *gin.Context) {
	c.JSON(http.StatusOK, Response{Message: "OK"})
}

func (s *Server) UpdateProfile(c *gin.Context) {
	c.JSON(http.StatusOK, Response{Message: "OK"})
}

type UpdatePasswordRequest struct {
	Password string `json:"password" validate:"required,min=8,max=128"`
}

func (s *Server) UpdatePassword(c *gin.Context) {
	ctx := c.Request.Context()
	userId, err := shared.GetUserIdFromContext(ctx)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, Response{
			Error: "User Id not found",
		})
		return
	}

	u, err := s.persist.GetUserByIdStr(userId)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	var req UpdatePasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Print(err)
		c.Status(http.StatusBadRequest)
		return
	}

	if err := validate.Struct(req); err != nil {
		log.Print(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, Response{
			Error: err.Error(),
		})
		return
	}

	u.Password = req.Password
	log.Printf("user: %+v", u)

	_, err = s.persist.UpdateUser(u)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{Message: "OK"})
}
