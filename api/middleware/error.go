package middleware

import (
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
			c.JSON(-1, err.Error())
		}
	}
}
