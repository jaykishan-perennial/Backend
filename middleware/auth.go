package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"license-management-backend/config"
	"license-management-backend/database"
	"license-management-backend/models"
)

func JWTAuth(cfg *config.Config, requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid authorization format"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid or expired token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid token claims"})
			c.Abort()
			return
		}

		role, _ := claims["role"].(string)
		if requiredRole != "" && role != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Insufficient permissions"})
			c.Abort()
			return
		}

		userID, _ := claims["user_id"].(float64)
		c.Set("user_id", uint(userID))
		c.Set("role", role)

		if role == "customer" {
			if custID, ok := claims["customer_id"].(float64); ok {
				c.Set("customer_id", uint(custID))
			}
		}

		c.Next()
	}
}

func APIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "X-API-Key header required"})
			c.Abort()
			return
		}

		var customer models.Customer
		if err := database.DB.Where("api_key = ?", apiKey).First(&customer).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid API key"})
			c.Abort()
			return
		}

		c.Set("customer_id", customer.ID)
		c.Set("user_id", customer.UserID)
		c.Next()
	}
}
