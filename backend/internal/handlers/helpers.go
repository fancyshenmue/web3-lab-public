package handlers

import "github.com/gin-gonic/gin"

// errorResponse returns a consistent error JSON payload.
func errorResponse(code, message string) gin.H {
	return gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	}
}
