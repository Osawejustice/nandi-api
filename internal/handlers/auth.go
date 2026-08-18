package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yourorg/nandi/internal/middleware"
	"github.com/yourorg/nandi/internal/services"
)

type AuthHandler struct {
	auth *services.AuthService
}

func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Register creates a tenant and the first owner user.
//
//	@Summary		Register organization
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		RegisterRequest	true	"Registration"
//	@Success		201		{object}	map[string]AuthData
//	@Failure		400		{object}	ErrorBody
//	@Failure		409		{object}	ErrorBody
//	@Router			/api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleBindError(c, err)
		return
	}
	result, err := h.auth.Register(c.Request.Context(), services.RegisterInput{
		Organization: req.Organization,
		Name:         req.Name,
		Email:        req.Email,
		Password:     req.Password,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}
	writeData(c, http.StatusCreated, authData(result))
}

// Login exchanges email/password for tokens.
//
//	@Summary		Login
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		LoginRequest	true	"Credentials"
//	@Success		200		{object}	map[string]AuthData
//	@Failure		401		{object}	ErrorBody
//	@Failure		409		{object}	ErrorBody
//	@Router			/api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleBindError(c, err)
		return
	}
	result, err := h.auth.Login(c.Request.Context(), services.LoginInput{
		Email:      req.Email,
		Password:   req.Password,
		TenantSlug: req.TenantSlug,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}
	writeData(c, http.StatusOK, authData(result))
}

// Refresh rotates the refresh token and returns a new pair.
//
//	@Summary		Refresh tokens
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		RefreshRequest	true	"Refresh token"
//	@Success		200		{object}	map[string]AuthData
//	@Failure		401		{object}	ErrorBody
//	@Router			/api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleBindError(c, err)
		return
	}
	result, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	writeData(c, http.StatusOK, authData(result))
}

// Logout revokes the supplied refresh token.
//
//	@Summary		Logout
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body	LogoutRequest	false	"Refresh token"
//	@Success		204
//	@Router			/api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req LogoutRequest
	_ = c.ShouldBindJSON(&req)
	if err := h.auth.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		handleServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Me returns the current principal and tenant.
//
//	@Summary		Current session
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]MeData
//	@Failure		401	{object}	ErrorBody
//	@Router			/api/v1/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	p := principalFromContext(c)
	user, tenant, err := h.auth.Me(c.Request.Context(), p)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	out := MeData{
		AuthType: middleware.AuthType(c),
		Role:     middleware.Role(c),
		Tenant:   tenantDTO(tenant),
	}
	if user != nil {
		dto := userDTO(user)
		out.User = &dto
	}
	writeData(c, http.StatusOK, out)
}

func authData(result *services.AuthResult) AuthData {
	return AuthData{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    result.ExpiresIn,
		Tenant:       tenantDTO(result.Tenant),
		User:         userDTO(result.User),
	}
}

// CreateUser adds an admin or agent to the current tenant.
//
//	@Summary		Create user
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateUserRequest	true	"User"
//	@Success		201		{object}	map[string]UserDTO
//	@Router			/api/v1/users [post]
func (h *AuthHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleBindError(c, err)
		return
	}
	user, err := h.auth.CreateUser(c.Request.Context(), middleware.TenantID(c), services.CreateUserInput{
		Name: req.Name, Email: req.Email, Password: req.Password, Role: req.Role,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}
	writeData(c, http.StatusCreated, userDTO(user))
}

func principalFromContext(c *gin.Context) services.Principal {
	return services.Principal{
		UserID:   middleware.UserID(c),
		TenantID: middleware.TenantID(c),
		Role:     middleware.Role(c),
		AuthType: middleware.AuthType(c),
		HasUser:  middleware.HasUser(c),
	}
}
