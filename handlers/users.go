package handlers

import (
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"avenue/backend/db"
	"avenue/backend/email"
	"avenue/backend/shared"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

//go:embed templates/user_created.html
var userCreatedTmplFS embed.FS

var userCreatedTmpl = template.Must(
	template.New("user_created.html").ParseFS(userCreatedTmplFS, "templates/user_created.html"),
)

func userCreatedHTML(setPasswordURL string) string {
	var buf bytes.Buffer
	if err := userCreatedTmpl.Execute(&buf, setPasswordURL); err != nil {
		return setPasswordURL
	}
	return buf.String()
}

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

	sessID, err := c.Cookie(string(shared.SESSIONCOOKIENAME))
	if err != nil {
		c.Status(http.StatusOK)
		return
	}

	err = db.DeleteSession(sessID)
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

	go func(userEmail string) {
		if err := email.Send(email.Message{
			To:      userEmail,
			Subject: "Welcome to Avenue",
			Text:    "Your Avenue account has been created. You can now log in at any time.",
		}); err != nil && !errors.Is(err, email.NotConfigured) {
			log.Printf("email(register): %v", err)
		}
	}(u.Email)

	c.JSON(http.StatusCreated, u)
}

type CreateUserRequest struct {
	Email     string  `json:"email" binding:"required,min=4,max=512"`
	Password  *string `json:"password" binding:"omitempty,min=4,max=64"`
	FirstName string  `json:"firstName" binding:"min=1,max=64"`
	LastName  string  `json:"lastName" binding:"min=1,max=64"`
	IsAdmin   bool    `json:"isAdmin"`
	SendEmail bool    `json:"sendEmail"`
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
		c.AbortWithStatusJSON(http.StatusUnauthorized, Response{Error: "You are not an admin"})
		return
	}

	var req CreateUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Print(err)
		c.AbortWithStatusJSON(http.StatusUnauthorized, Response{Error: err.Error()})
		return
	}

	if !req.SendEmail && req.Password == nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, Response{
			Error: "password is required when not sending an invite email",
		})
		return
	}

	if !db.IsUniqueEmail(req.Email) {
		c.AbortWithStatusJSON(http.StatusConflict, Response{
			Error: "Email already exists",
		})
		return
	}

	// When sending an invite email the admin doesn't set a password — generate a
	// random placeholder that the user will replace via the set-password link.
	password := ""
	if req.Password != nil {
		password = *req.Password
	} else {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, Response{Error: "could not generate password"})
			return
		}
		password = hex.EncodeToString(b)
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
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

	if req.SendEmail {
		// Capture request fields before entering the goroutine — gin may recycle the context.
		scheme := "https"
		if c.Request.TLS == nil {
			scheme = "http"
		}
		host := c.Request.Host
		go func(userID int64, userEmail, scheme, host string) {
			token, err := db.CreatePasswordResetToken(userID)
			if err != nil {
				log.Printf("email(user created): create reset token: %v", err)
				return
			}
			setPasswordURL := scheme + "://" + host + "/reset-password?token=" + token
			if err := email.Send(email.Message{
				To:      userEmail,
				Subject: "Your Avenue account has been created",
				HTML:    userCreatedHTML(setPasswordURL),
				Text:    "An administrator has created an Avenue account for you.\n\nClick the link below to set your password:\n\n" + setPasswordURL + "\n\nThis link expires in 1 hour.",
			}); err != nil && !errors.Is(err, email.NotConfigured) {
				log.Printf("email(user created): %v", err)
			}
		}(nu.ID, nu.Email, scheme, host)
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
