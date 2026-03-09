package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {

		var tokenString string

		authHeader := ctx.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 || parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		} else {
			tokenString, _ = ctx.Cookie("auth_token")
		}

		if tokenString == "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"message": "Authentication required (no token in cookie or header)",
			})
			ctx.Abort()
			return
		}

		claims, err := ValidateJWT(tokenString)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired  token", "details": err.Error()})
			ctx.Abort()
			return
		}

		ctx.Set("user_id", claims.UserID)
		ctx.Set("user_email", claims.Email)

		ctx.Next()
	}
}
