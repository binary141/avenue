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
	Email    string `json:"email" validate:"required,min=4,max=64"`
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

	u, err := s.authorize(req.Email, req.Password)
	if err != nil {
		// for now send the error in the response 🤔
		c.AbortWithStatusJSON(http.StatusUnauthorized, Response{
			Error: err.Error(),
		})
		return
	}

	session, err := s.persist.CreateSession(int(u.ID))
	if err != nil {
		// for now send the error in the response 🤔
		c.AbortWithStatusJSON(http.StatusUnauthorized, Response{
			Error: err.Error(),
		})
		return
	}

	c.SetCookie(string(shared.USERCOOKIENAME), fmt.Sprintf("%d", u.ID), 600, "/", "localhost", false, true)
	c.SetCookie(string(shared.SESSIONCOOKIENAME), session.SessionID, 600, "/", "localhost", false, true)

	c.JSON(http.StatusOK, gin.H{
		"Message":                        "OK",
		"User-Id":                        u.ID,
		string(shared.SESSIONCOOKIENAME): session.SessionID,
		"user_data":                      u,
	})
}

func (s *Server) authorize(email, password string) (persist.User, error) {
	user, err := s.persist.GetUserByEmail(email)
	if err != nil {
		return user, err
	}

	// todo make this more smart
	if user.Password != password {
		return user, errors.New("password incorrect")
	}

	return user, nil
}

func (s *Server) Logout(c *gin.Context) {
	// expire the cookie
	c.SetCookie(string(shared.USERCOOKIENAME), "", -1, "/", "localhost", false, true)

	ctx := c.Request.Context()

	sessID := ctx.Value(shared.SESSIONCOOKIENAME)

	sessIDStr, ok := sessID.(string)
	if !ok {
		c.Status(http.StatusBadRequest)
		return
	}

	err := s.persist.DeleteSession(sessIDStr)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, Response{Message: "OK"})
}

type RegisterRequest struct {
	Password  string `json:"password" validate:"required,min=4,max=64"`
	FirstName string `json:"firstName" validate:"min=1,max=64"`
	LastName  string `json:"lastName" validate:"min=1,max=64"`
	Email     string `json:"email" validate:"required,min=4,max=512"`
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

	isAdmin := false

	u, err := s.persist.CreateUser(req.Email, req.Password, req.FirstName, req.LastName, isAdmin)
	if err != nil {
		log.Print(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, u)
}

type CreateUserRequest struct {
	Email     string `json:"email" validate:"required,min=4,max=512"`
	Password  string `json:"password" validate:"required,min=4,max=64"`
	FirstName string `json:"firstName" validate:"min=1,max=64"`
	LastName  string `json:"lastName" validate:"min=1,max=64"`
	IsAdmin   bool   `json:"isAdmin"`
}

func (s *Server) CreateUser(c *gin.Context) {
	ctx := c.Request.Context()
	userID, err := shared.GetUserIDFromContext(ctx)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, Response{
			Error: fmt.Sprintf("User Id not found: %s", err.Error()),
		})
		return
	}

	u, err := s.persist.GetUserByIDStr(userID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	if !u.IsAdmin {
		c.AbortWithStatusJSON(http.StatusUnauthorized, Response{})
		return
	}

	var req CreateUserRequest

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

	log.Printf("Req: %+v\n", req)

	if !s.persist.IsUniqueEmail(req.Email) {
		c.AbortWithStatusJSON(http.StatusConflict, Response{
			Error: "Email already exists",
		})
		return
	}

	nu, err := s.persist.CreateUser(req.Email, req.Password, req.FirstName, req.LastName, req.IsAdmin)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	log.Printf("New User: %+v\n", nu)

	// todo allow pagination
	us, err := s.persist.GetUsers()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, us)
}

func (s *Server) GetUsers(c *gin.Context) {
	ctx := c.Request.Context()
	userID, err := shared.GetUserIDFromContext(ctx)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, Response{
			Error: fmt.Sprintf("User Id not found: %s", err.Error()),
		})
		return
	}

	u, err := s.persist.GetUserByIDStr(userID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	if !u.IsAdmin {
		c.AbortWithStatusJSON(http.StatusUnauthorized, Response{})
		return
	}

	// todo allow pagination
	us, err := s.persist.GetUsers()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, us)
}

func (s *Server) GetProfile(c *gin.Context) {
	ctx := c.Request.Context()
	userID, err := shared.GetUserIDFromContext(ctx)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, Response{
			Error: fmt.Sprintf("User Id not found: %s", err.Error()),
		})
		return
	}

	u, err := s.persist.GetUserByIDStr(userID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, u)
}

type UpdateProfileRequest struct {
	ID        string `json:"id" validate:"required"`
	Email     string `json:"email" validate:"omitempty,email,min=4,max=512"`
	IsAdmin   bool   `json:"isAdmin"`
	Password  string `json:"password" validate:"omitempty,min=4,max=64"`
	FirstName string `json:"firstName" validate:"omitempty,min=1,max=64"`
	LastName  string `json:"lastName" validate:"omitempty,min=1,max=64"`
}

func (s *Server) UpdateProfile(c *gin.Context) {
	ctx := c.Request.Context()
	userID, err := shared.GetUserIDFromContext(ctx)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, Response{
			Error: "User Id not found",
		})
		return
	}

	var req UpdateProfileRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, Response{
			Error: err.Error(),
		})
		return
	}

	if err := validate.Struct(req); err != nil {
		log.Print(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, Response{
			Error: err.Error(),
		})
		return
	}

	u, err := s.persist.GetUserByIDStr(userID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	if req.ID != userID && !u.IsAdmin {
		c.AbortWithStatusJSON(http.StatusBadRequest, Response{
			Error: "Only admin users can edit another users information",
		})
		return
	}

	updatingUser, err := s.persist.GetUserByIDStr(req.ID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	if req.Email != "" && req.Email != updatingUser.Email {
		if !s.persist.IsUniqueEmail(req.Email) {
			c.AbortWithStatusJSON(http.StatusConflict, Response{
				Error: "Email already exists",
			})
			return
		}

		updatingUser.Email = req.Email
	}

	if req.FirstName != "" {
		updatingUser.FirstName = req.FirstName
	}

	if req.LastName != "" {
		updatingUser.LastName = req.LastName
	}

	if req.Password != "" {
		updatingUser.Password = req.Password
	}

	updatingUser.IsAdmin = req.IsAdmin

	if !req.IsAdmin {
		otherAdmins, _ := s.persist.HasOtherAdmins(updatingUser)
		if !otherAdmins {
			c.AbortWithStatusJSON(http.StatusBadRequest, Response{
				Error: "Application requires at least one Admin user",
			})
			return
		}
	}

	log.Printf("%+v", req)
	log.Printf("%+v", updatingUser)

	updatingUser, err = s.persist.UpdateUser(updatingUser)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, updatingUser)
}

type UpdatePasswordRequest struct {
	Password string `json:"password" validate:"required,min=8,max=128"`
}

func (s *Server) UpdatePassword(c *gin.Context) {
	ctx := c.Request.Context()
	userID, err := shared.GetUserIDFromContext(ctx)
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

	u, err := s.persist.GetUserByIDStr(userID)
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
