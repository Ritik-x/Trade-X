package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const userIDContextKey = "auth_user_id"

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {

		// 1. Read the Authorization header.
		authHeader := c.GetHeader("Authorization")

		// 2. Extract the Bearer token.
		parts := strings.Fields(authHeader)

		if len(parts) != 2 ||
			!strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing or invalid authorization header",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 3. Prepare a structure to hold JWT claims.
		claims := &jwt.RegisteredClaims{}

		// 4. Parse and validate the JWT.
		token, err := jwt.ParseWithClaims(
			tokenString,
			claims,
			func(token *jwt.Token) (any, error) {

				// Accept only the HS256 signing algorithm.
				if token.Method != jwt.SigningMethodHS256 {
					return nil, jwt.ErrSignatureInvalid
				}

				return []byte(jwtSecret), nil
			},
			jwt.WithValidMethods([]string{
				jwt.SigningMethodHS256.Alg(),
			}),
			jwt.WithIssuer("tradex"),
			jwt.WithExpirationRequired(),
		)

		if err != nil || token == nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired access token",
			})
			c.Abort()
			return
		}

		// 5. Extract and validate the user's UUID.
		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token subject",
			})
			c.Abort()
			return
		}

		// 6. Store the authenticated user's ID in Gin context.
		c.Set(userIDContextKey, userID)

		// 7. Continue to the next middleware or handler.
		c.Next()
	}
}

// GetUserID retrieves the authenticated user's ID from Gin context.
func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get(userIDContextKey)
	if !exists {
		return uuid.Nil, false
	}

	userID, ok := value.(uuid.UUID)
	return userID, ok
}
