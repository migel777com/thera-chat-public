package middleware

import (
	"chatgpt/models"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
)

// Can pass the logger
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		var err error
		for _, err = range c.Errors {
			// Adding stuck trace should be done without error handler middleware
			// so maybe error handler should be replaced to separate function
			// or stacks can be added to the error itself, so it has to be explored and tested
			fmt.Errorf("error: %v", err.Error())
		}

		// status -1 doesn't overwrite existing status code
		if err != nil {
			var errResponse models.ErrorResponse
			var errAdvanced models.AdvancedErrorResponse
			switch {
			case errors.As(err, &errResponse):
				c.JSON(-1, gin.H{"error": models.ErrorResponse{Code: errResponse.Code, Message: errResponse.Message}})
			case errors.As(err, &errAdvanced):
				c.JSON(-1, gin.H{errAdvanced.Key: models.AdvancedErrorResponse{Code: errAdvanced.Code, Message: errAdvanced.Message}})
			default:
				c.JSON(-1, gin.H{"error": models.ErrorResponse{Code: c.Writer.Status(), Message: err.Error()}})
			}
		}
	}
}
