package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"avenue/backend/persist"
	"avenue/backend/shared"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type LoginRequest struct {
	Email    string `json:"email" validate:"required,min=4,max=64"`
	Password string `json:"password" validate:"required,min=4,max=64"`
}

var Sessions map[string]SessionData

type SessionData struct {
	ExpiresAt time.Time
	IsValid   bool
	UserId    uint
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

	u, err := s.authorize(req.Email, req.Password)
	if err != nil {
		// for now send the error in the response 🤔
		c.AbortWithStatusJSON(http.StatusUnauthorized, Response{
			Error: err.Error(),
		})
		return
	}

	uuidStr := uuid.NewString()

	Sessions[uuidStr] = SessionData{
		ExpiresAt: time.Now().Add(12 * time.Hour),
		IsValid:   true,
		UserId:    u.ID,
	}

	c.SetCookie(shared.USERCOOKIENAME, fmt.Sprintf("%d", u.ID), 600, "/", "localhost", false, true)
	c.SetCookie(shared.SESSIONCOOKIENAME, uuidStr, 600, "/", "localhost", false, true)
	c.JSON(http.StatusOK, gin.H{"Message": "OK", "User-Id": u.ID, shared.SESSIONCOOKIENAME: uuidStr})
}

func (s *Server) authorize(email, password string) (persist.User, error) {
	user, err := s.persist.GetUserByEmail(email)
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

	// TODO delete the session from map
	ctx := c.Request.Context()

	sessID := ctx.Value(shared.SESSIONCOOKIENAME)

	sessIDStr, ok := sessID.(string)
	if !ok {
		c.Status(http.StatusBadRequest)
		return
	}

	v, ok := Sessions[sessIDStr]
	if !ok {
		c.Status(http.StatusBadRequest)
		return
	}

	v.IsValid = false
	v.ExpiresAt = time.Now()

	Sessions[sessIDStr] = v

	c.JSON(http.StatusOK, Response{Message: "OK"})
}

type RegisterRequest struct {
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

	if !s.persist.IsUniqueEmail(req.Email) {
		c.AbortWithStatusJSON(http.StatusConflict, Response{
			Error: "Email already exists",
		})
		return
	}

	u, err := s.persist.CreateUser(req.Email, req.Password)
	if err != nil {
		log.Print(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, u)
}

func (s *Server) GetProfile(c *gin.Context) {
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

	c.JSON(http.StatusOK, u)
}

type UpdateProfileRequest struct {
	Email string `json:"email" validate:"omitempty,email"`
}

func (s *Server) UpdateProfile(c *gin.Context) {
	ctx := c.Request.Context()
	userId, err := shared.GetUserIdFromContext(ctx)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, Response{
			Error: "User Id not found",
		})
		return
	}

	var req UpdateProfileRequest

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

	log.Print(req)

	u, err := s.persist.GetUserByIdStr(userId)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	if req.Email != "" && req.Email != u.Email {
		if !s.persist.IsUniqueEmail(req.Email) {
			c.AbortWithStatusJSON(http.StatusConflict, Response{
				Error: "Email already exists",
			})
			return
		}

		u.Email = req.Email
	}

	u, err = s.persist.UpdateUser(u)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, u)
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

	u, err := s.persist.GetUserByIdStr(userId)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	u.Password = req.Password

	u, err = s.persist.UpdateUser(u)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, u)
}
