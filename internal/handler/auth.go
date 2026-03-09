package handler

import (
	"bulk-email-platform/internal/auth"
	"bulk-email-platform/internal/domain"
	"bulk-email-platform/internal/repository"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	repo *repository.PostGresRepo
}

func NewAuthHandler(repo *repository.PostGresRepo) *AuthHandler {
	return &AuthHandler{
		repo: repo,
	}
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required,min=6,max=20"`
	FullName string `json:"full_name" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required,min=6,max=20"`
}

type AuthResponse struct {
	Token    string    `json:"token"`
	UserID   uuid.UUID `json:"user_id"`
	Email    string    `json:"email"`
	FullName string    `json:"full_name"`
}

func (a AuthHandler) setAuthCookie(c *gin.Context, token string) {

	isProduction := os.Getenv("ENV") == "production"

	c.SetCookie("auth_token", token, 60*60*24*10, "/", "", isProduction, true)

}

func (h *AuthHandler) clearAuthCookie(c *gin.Context) {
	c.SetCookie("auth_token", "", -1, "/", "", false, true)
}

func (a *AuthHandler) Register(c *gin.Context) {

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existingUser, err := a.repo.GetUserByEmail(c.Request.Context(), req.Email)
	if err == nil && existingUser.ID != uuid.Nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user with this email already exists"})
		return
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process registration"})
		return
	}

	user := domain.User{
		Email:        req.Email,
		FullName:     req.FullName,
		PasswordHash: string(hashedBytes),
	}

	if err := a.repo.CreateNewUser(c.Request.Context(), &user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	a.setAuthCookie(c, token)

	c.JSON(http.StatusCreated, AuthResponse{
		Token:    token,
		UserID:   user.ID,
		Email:    user.Email,
		FullName: user.FullName,
	})
}

func (a *AuthHandler) Login(c *gin.Context) {

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := a.repo.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil || user.ID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Password is incorrect"})
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	a.setAuthCookie(c, token)

	c.JSON(http.StatusCreated, AuthResponse{
		Token:    token,
		UserID:   user.ID,
		Email:    user.Email,
		FullName: user.FullName,
	})
}

func (a *AuthHandler) Logout(c *gin.Context) {
	a.clearAuthCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":   userID,
		"email":     c.GetString("user_email"),
		"full_name": c.GetString("user_full_name"),
	})
}
