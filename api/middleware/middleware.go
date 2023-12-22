package middleware

import (
	"chatgpt/auth"
	"chatgpt/models"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

// Authenticate Authentication middleware.
func Authenticate(cache models.CacheClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		// read Header.
		// standard Header format: Authorization: Bearer tokenValue.
		authorizationHeader := c.Request.Header.Get("Authorization")

		// check, if Header empty, then clear context and return.
		headerParts := strings.Split(authorizationHeader, " ")

		// header format validation.
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			// log: invalid authentication token response(c)
			c.AbortWithError(http.StatusUnauthorized, errors.New("no Authorization token"))
			return
		}
		token := headerParts[1]

		// token format validation.

		var user models.User
		err := auth.GetUserByToken(ctx, cache, token, &user)
		if err != nil {
			// log: error. Specific case: Not Found response.
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.Set("user", user)
		c.Set("token", token)
		c.Next()
	}
}
