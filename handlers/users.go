package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"avenue/backend/db"
	"avenue/backend/logger"
	"avenue/backend/sdk"
	"avenue/backend/shared"

	ksdk "github.com/binary141/keyring/sdk"
	"github.com/gin-gonic/gin"
)

func (s *Server) LoginMeta(c *gin.Context) {
	enabled := shared.GetEnv("REGISTRATION_ENABLED", "false")
	c.JSON(http.StatusOK, sdk.V1LoginMetaResponse{RegistrationEnabled: enabled})
}

// mapKeyringUser combines identity fields from keyring with quota/usage
// tracked locally into the sdk.User shape the frontend already expects.
func mapKeyringUser(u ksdk.User, local db.LocalUser) sdk.User {
	return sdk.User{
		ID:        local.ID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		CanLogin:  u.CanLogin,
		IsAdmin:   u.IsAdmin,
		Quota:     local.Quota,
		SpaceUsed: local.SpaceUsed,
		CreatedAt: u.CreatedAt,
	}
}

func (s *Server) Login(c *gin.Context) {
	var req sdk.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	emailKey := normalizeEmailKey(req.Email)
	if !loginIPLimiter.allow(c.ClientIP()) || !loginEmailLimiter.allow(emailKey) {
		c.Status(http.StatusTooManyRequests)
		return
	}

	resp, err := s.keyringClient("").Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	loginEmailLimiter.reset(emailKey)

	localUser, err := db.GetOrCreateLocalUser(resp.User.ID, resp.User.UUID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", fmt.Errorf("resolve local user: %w", err))
		return
	}

	maxAge := int(SessionRollingWindow.Seconds())
	c.SetCookie(string(shared.USERCOOKIENAME), fmt.Sprintf("%d", localUser.ID), maxAge, "/", COOKIEDOMAIN, COOKIESECURE, true)
	c.SetCookie(string(shared.SESSIONCOOKIENAME), resp.Token, maxAge, "/", COOKIEDOMAIN, COOKIESECURE, true)

	c.JSON(http.StatusOK, sdk.V1LoginResponse{
		Message:   "OK",
		UserID:    localUser.ID,
		SessionID: resp.Token,
		UserData:  mapKeyringUser(resp.User, localUser),
	})
}

func (s *Server) Logout(c *gin.Context) {
	c.SetCookie(string(shared.USERCOOKIENAME), "", -1, "/", COOKIEDOMAIN, COOKIESECURE, true)
	c.SetCookie(string(shared.SESSIONCOOKIENAME), "", -1, "/", COOKIEDOMAIN, COOKIESECURE, true)

	// sessionCheck (required by the secured route this handler is mounted on)
	// authenticates via the Authorization header, not the ambient cookie, so
	// logging out can't be triggered cross-site.
	token, err := shared.GetSessionIDFromContext(c.Request.Context())
	if err != nil {
		c.Status(http.StatusOK)
		return
	}

	if err := s.keyringClient(token).Logout(c.Request.Context()); err != nil {
		respond(c, http.StatusInternalServerError, "", fmt.Errorf("logout: %w", err))
		return
	}

	respond(c, http.StatusOK, "OK", nil)
}

func (s *Server) Register(c *gin.Context) {
	enabled := shared.GetEnv("REGISTRATION_ENABLED", "false")
	if enabled == "false" {
		respond(c, http.StatusBadRequest, "", errors.New("registration is not enabled"))
		return
	}

	var req sdk.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		respond(c, http.StatusBadRequest, "", err)
		return
	}

	if !shared.IsValidEmail(req.Email) {
		respond(c, http.StatusBadRequest, "", errors.New("email is not valid"))
		return
	}

	// Keyring's Register creates an unverified account and emails a
	// verification link; the account can't log in until VerifyEmail is
	// called. The response never reveals whether the email already existed.
	if err := s.keyringClient("").Register(c.Request.Context(), req.Email, req.Password, "", req.FirstName, req.LastName); err != nil {
		respond(c, http.StatusInternalServerError, "", fmt.Errorf("register: %w", err))
		return
	}

	c.JSON(http.StatusCreated, sdk.MessageResponse{
		Message: "if this email is available, an account has been created; check your email to verify it",
	})
}

// VerifyEmail activates a self-registered account, letting it log in.
func (s *Server) VerifyEmail(c *gin.Context) {
	var req sdk.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond(c, http.StatusBadRequest, "", err)
		return
	}

	if err := s.keyringClient("").VerifyEmail(c.Request.Context(), req.Token); err != nil {
		respond(c, http.StatusBadRequest, "", errors.New("invalid or expired verification token"))
		return
	}

	respond(c, http.StatusOK, "OK", nil)
}

// requireAdmin confirms the authenticated caller holds keyring's admin
// flag. On failure it writes the appropriate response and returns ok=false.
func requireAdmin(c *gin.Context) (ok bool) {
	if !shared.GetIsAdminFromContext(c.Request.Context()) {
		respond(c, http.StatusUnauthorized, "", errors.New("you are not an admin"))
		return false
	}
	return true
}

func (s *Server) CreateUser(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	token, err := shared.GetSessionIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusBadRequest, "", errors.New("session token not found"))
		return
	}

	var req sdk.CreateUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		respond(c, http.StatusBadRequest, "", err)
		return
	}

	if req.Password == nil {
		respond(c, http.StatusBadRequest, "", errors.New("password is required"))
		return
	}

	kc := s.keyringClient(token)

	if err := kc.CreateUser(c.Request.Context(), req.Email, "", req.FirstName, req.LastName, *req.Password); err != nil {
		respond(c, http.StatusInternalServerError, "", fmt.Errorf("create user: %w", err))
		return
	}

	created, err := findKeyringUserByEmail(c.Request.Context(), kc, req.Email)
	if err == nil {
		localUser, err := db.GetOrCreateLocalUser(created.ID, created.UUID)
		if err == nil {
			if req.Quota != nil {
				_ = db.UpdateQuota(localUser.ID, *req.Quota)
			}
			if req.IsAdmin {
				_ = kc.PromoteUser(c.Request.Context(), created.ID)
			}
		}
	}

	// Keyring doesn't yet send real invite email either (it logs the link
	// server-side), but this keeps the two systems consistent: get the new
	// user a reset link to set their own password.
	if req.SendEmail {
		_ = kc.ForgotPassword(c.Request.Context(), req.Email)
	}

	logger.Infof("new user created: email=%s", req.Email)

	us, err := s.listUsersWithQuota(c, token)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", fmt.Errorf("list users: %w", err))
		return
	}

	c.JSON(http.StatusOK, us)
}

// findKeyringUserByEmail scans keyring's user list for a matching email.
// Keyring's admin API only exposes bulk listing, not single-user lookup by
// email, so this trades an extra round trip for staying entirely on the
// public API surface.
func findKeyringUserByEmail(ctx context.Context, kc *ksdk.Client, email string) (ksdk.User, error) {
	users, err := kc.ListUsers(ctx)
	if err != nil {
		return ksdk.User{}, err
	}
	for _, u := range users {
		if u.Email == email {
			return u, nil
		}
	}
	return ksdk.User{}, fmt.Errorf("user with email %q not found", email)
}

// findKeyringUserByID scans keyring's user list for a matching ID, for the
// same reason as findKeyringUserByEmail.
func findKeyringUserByID(ctx context.Context, kc *ksdk.Client, id int64) (ksdk.User, error) {
	users, err := kc.ListUsers(ctx)
	if err != nil {
		return ksdk.User{}, err
	}
	for _, u := range users {
		if u.ID == id {
			return u, nil
		}
	}
	return ksdk.User{}, fmt.Errorf("user %d not found", id)
}

// listUsersWithQuota fetches every user from keyring and joins in the
// locally-tracked quota/space_used for each, creating a local mirror row
// for any keyring user seen for the first time.
func (s *Server) listUsersWithQuota(c *gin.Context, token string) ([]sdk.User, error) {
	kUsers, err := s.keyringClient(token).ListUsers(c.Request.Context())
	if err != nil {
		return nil, err
	}

	local, err := db.ListLocalUsersByKeyringID()
	if err != nil {
		return nil, err
	}

	out := make([]sdk.User, 0, len(kUsers))
	for _, ku := range kUsers {
		lu, ok := local[ku.ID]
		if !ok {
			lu, err = db.GetOrCreateLocalUser(ku.ID, ku.UUID)
			if err != nil {
				return nil, err
			}
		}
		out = append(out, mapKeyringUser(ku, lu))
	}
	return out, nil
}

func (s *Server) GetUsers(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	token, err := shared.GetSessionIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusBadRequest, "", errors.New("session token not found"))
		return
	}

	us, err := s.listUsersWithQuota(c, token)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", fmt.Errorf("list users: %w", err))
		return
	}

	c.JSON(http.StatusOK, us)
}

func (s *Server) GetProfile(c *gin.Context) {
	ctx := c.Request.Context()
	userID, err := shared.GetUserIDFromContext(ctx)
	if err != nil {
		respond(c, http.StatusBadRequest, "", fmt.Errorf("user id not found: %w", err))
		return
	}
	token, err := shared.GetSessionIDFromContext(ctx)
	if err != nil {
		respond(c, http.StatusBadRequest, "", errors.New("session token not found"))
		return
	}

	localUser, err := db.GetLocalUserByIDStr(userID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", fmt.Errorf("get local user: %w", err))
		return
	}

	me, err := s.keyringClient(token).Me(ctx)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", fmt.Errorf("get user: %w", err))
		return
	}

	c.JSON(http.StatusOK, mapKeyringUser(me.User, localUser))
}

func (s *Server) UpdateProfile(c *gin.Context) {
	ctx := c.Request.Context()
	callerLocalIDStr, err := shared.GetUserIDFromContext(ctx)
	if err != nil {
		respond(c, http.StatusBadRequest, "", errors.New("user id not found"))
		return
	}
	token, err := shared.GetSessionIDFromContext(ctx)
	if err != nil {
		respond(c, http.StatusBadRequest, "", errors.New("session token not found"))
		return
	}
	callerIsAdmin := shared.GetIsAdminFromContext(ctx)

	var req sdk.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond(c, http.StatusBadRequest, "", err)
		return
	}

	callerLocalID, err := strconv.ParseInt(callerLocalIDStr, 10, 64)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", errors.New("invalid caller id"))
		return
	}

	targetLocal, err := db.GetLocalUserByID(req.ID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", fmt.Errorf("get target user: %w", err))
		return
	}

	isSelf := req.ID == callerLocalID
	if !isSelf && !callerIsAdmin {
		respond(c, http.StatusBadRequest, "", errors.New("only admin users can edit another user's information"))
		return
	}

	kc := s.keyringClient(token)

	target, err := findKeyringUserByID(ctx, kc, targetLocal.KeyringUserID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", fmt.Errorf("get target user: %w", err))
		return
	}

	if req.Password != nil && isSelf {
		if req.CurrentPassword == nil || *req.CurrentPassword == "" {
			respond(c, http.StatusBadRequest, "", errors.New("current password is required"))
			return
		}
		if err := kc.ChangePassword(ctx, *req.CurrentPassword, *req.Password); err != nil {
			respond(c, http.StatusUnauthorized, "", errors.New("current password is incorrect"))
			return
		}
	}

	email := target.Email
	if req.Email != nil {
		email = *req.Email
	}
	firstName := target.FirstName
	if req.FirstName != nil {
		firstName = *req.FirstName
	}
	lastName := target.LastName
	if req.LastName != nil {
		lastName = *req.LastName
	}

	if req.Email != nil || req.FirstName != nil || req.LastName != nil {
		if err := kc.UpdateUser(ctx, target.ID, email, target.DisplayName, firstName, lastName); err != nil {
			status := http.StatusInternalServerError
			var apiErr *ksdk.APIError
			if errors.As(err, &apiErr) {
				status = apiErr.StatusCode
			}
			respond(c, status, "", fmt.Errorf("update user: %w", err))
			return
		}
	}

	if req.Quota != nil && callerIsAdmin {
		if err := db.UpdateQuota(targetLocal.ID, *req.Quota); err != nil {
			respond(c, http.StatusInternalServerError, "", fmt.Errorf("update quota: %w", err))
			return
		}
	}

	if req.IsAdmin != nil && callerIsAdmin {
		var roleErr error
		if *req.IsAdmin {
			roleErr = kc.PromoteUser(ctx, target.ID)
		} else {
			roleErr = kc.DemoteUser(ctx, target.ID)
		}
		if roleErr != nil {
			status := http.StatusInternalServerError
			var apiErr *ksdk.APIError
			if errors.As(roleErr, &apiErr) {
				status = apiErr.StatusCode
			}
			respond(c, status, "", fmt.Errorf("update admin status: %w", roleErr))
			return
		}
	}

	updated, err := findKeyringUserByID(ctx, kc, target.ID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", fmt.Errorf("get updated user: %w", err))
		return
	}
	targetLocal, err = db.GetLocalUserByID(targetLocal.ID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", fmt.Errorf("get updated local user: %w", err))
		return
	}

	c.JSON(http.StatusOK, mapKeyringUser(updated, targetLocal))
}

func (s *Server) UpdatePassword(c *gin.Context) {
	ctx := c.Request.Context()
	token, err := shared.GetSessionIDFromContext(ctx)
	if err != nil {
		respond(c, http.StatusBadRequest, "", errors.New("session token not found"))
		return
	}

	var req sdk.UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond(c, http.StatusBadRequest, "", err)
		return
	}

	if err := s.keyringClient(token).ChangePassword(ctx, req.CurrentPassword, req.Password); err != nil {
		respond(c, http.StatusUnauthorized, "", errors.New("current password is incorrect"))
		return
	}

	userID, err := shared.GetUserIDFromContext(ctx)
	if err != nil {
		respond(c, http.StatusBadRequest, "", errors.New("user id not found"))
		return
	}
	localUser, err := db.GetLocalUserByIDStr(userID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", fmt.Errorf("get local user: %w", err))
		return
	}
	me, err := s.keyringClient(token).Me(ctx)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", fmt.Errorf("get user: %w", err))
		return
	}

	c.JSON(http.StatusOK, mapKeyringUser(me.User, localUser))
}
