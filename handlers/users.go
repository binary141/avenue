package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"avenue/backend/db"
	"avenue/backend/shared"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Email    string `json:"email" binding:"required,min=4,max=64"`
	Password string `json:"password" binding:"required,min=4,max=64"`
}

func (s *Server) LoginMeta(c *gin.Context) {

	enabled := shared.GetEnv("REGISTRATION_ENABLED", "false")

	c.JSON(http.StatusOK, gin.H{"registration_enabled": enabled})
}

func (s *Server) Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
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

	session, err := db.CreateSession(u.ID)
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

func (s *Server) authorize(email, password string) (db.User, error) {
	user, err := db.GetUserByEmail(email)
	if err != nil {
		return user, err
	}

	log.Println(user.Password)

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return db.User{}, err
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

	err := db.DeleteSession(sessIDStr)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, Response{Message: "OK"})
}

type RegisterRequest struct {
	Password  string `json:"password" binding:"required,min=4,max=64"`
	FirstName string `json:"firstName" binding:"max=64"`
	LastName  string `json:"lastName" binding:"max=64"`
	Email     string `json:"email" binding:"required,min=4,max=512"`
}

func (s *Server) Register(c *gin.Context) {
	enabled := strings.ToLower(shared.GetEnv("REGISTRATION_ENABLED", "false"))

	if enabled == "false" {
		c.AbortWithStatusJSON(http.StatusBadRequest, Response{
			Error: "Registration is not enabled",
		})
		return
	}

	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
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

	if !db.IsUniqueEmail(req.Email) {
		c.AbortWithStatusJSON(http.StatusConflict, Response{
			Error: "Email already exists",
		})
		return
	}

	isAdmin := false

	hashedPass, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println(err.Error())
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, Response{
			Error: err.Error(),
		})
		return
	}

	u, err := db.CreateUser(req.Email, string(hashedPass), req.FirstName, req.LastName, isAdmin)
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
	Email     string `json:"email" binding:"required,min=4,max=512"`
	Password  string `json:"password" binding:"required,min=4,max=64"`
	FirstName string `json:"firstName" binding:"min=1,max=64"`
	LastName  string `json:"lastName" binding:"min=1,max=64"`
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

	u, err := db.GetUserByIDStr(userID)
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

	if !db.IsUniqueEmail(req.Email) {
		c.AbortWithStatusJSON(http.StatusConflict, Response{
			Error: "Email already exists",
		})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println(err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	nu, err := db.CreateUser(req.Email, string(hashed), req.FirstName, req.LastName, req.IsAdmin)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	log.Printf("New User: %+v\n", nu)

	// todo allow pagination
	us, err := db.GetUsers()
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

	u, err := db.GetUserByIDStr(userID)
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
	us, err := db.GetUsers()
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

	u, err := db.GetUserByIDStr(userID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, u)
}

type UpdateProfileRequest struct {
	ID        int64   `json:"id" binding:"required,min=1"`
	Email     *string `json:"email" binding:"omitempty,email,min=4,max=512"`
	IsAdmin   *bool   `json:"isAdmin"`
	Password  *string `json:"password" binding:"omitempty,min=4,max=64"`
	FirstName *string `json:"firstName" binding:"min=0,max=64"`
	LastName  *string `json:"lastName" binding:"min=0,max=64"`
	Quota     *int64  `json:"quota" binding:"omitempty,min=0"`
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

	u, err := db.GetUserByIDStr(userID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	if fmt.Sprintf("%d", req.ID) != userID && !u.IsAdmin {
		c.AbortWithStatusJSON(http.StatusBadRequest, Response{
			Error: "Only admin users can edit another users information",
		})
		return
	}

	updatingUser, err := db.GetUserByID(req.ID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	if req.Email != nil && *req.Email != updatingUser.Email {
		if !db.IsUniqueEmail(*req.Email) {
			c.AbortWithStatusJSON(http.StatusConflict, Response{
				Error: "Email already exists",
			})
			return
		}

		updatingUser.Email = *req.Email
	}

	if req.FirstName != nil {
		updatingUser.FirstName = *req.FirstName
	}

	if req.LastName != nil {
		updatingUser.LastName = *req.LastName
	}

	log.Println(req)
	log.Println(updatingUser)

	if req.Password != nil {
		hashed, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Println(err.Error())
			c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
				Error: err.Error(),
			})
			return
		}
		updatingUser.Password = string(hashed)
	}

	if req.IsAdmin != nil && u.IsAdmin {
		otherAdmins, _ := db.HasOtherAdmins(updatingUser)
		if !otherAdmins && !*req.IsAdmin {
			c.AbortWithStatusJSON(http.StatusBadRequest, Response{
				Error: "Application requires at least one Admin user",
			})
			return
		}

		updatingUser.IsAdmin = *req.IsAdmin
	}

	if req.Quota != nil {
		updatingUser.Quota = *req.Quota
	}

	updatingUser, err = db.UpdateUser(updatingUser)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, updatingUser)
}

type UpdatePasswordRequest struct {
	Password string `json:"password" binding:"required,min=8,max=128"`
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

	u, err := db.GetUserByIDStr(userID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println(err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}
	u.Password = string(hashed)

	u, err = db.UpdateUser(u)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, u)
}
