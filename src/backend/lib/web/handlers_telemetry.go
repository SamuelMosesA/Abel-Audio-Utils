package web

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type FrontendErrorRequest struct {
	Message string `json:"message" binding:"required"`
	Stack   string `json:"stack"`
	URL     string `json:"url"`
}

// ErrorLogHandler receives frontend unhandled errors and logs them via slog.
func ErrorLogHandler() gin.HandlerFunc {
	logger := slog.With("app.component", "frontend")
	return func(c *gin.Context) {
		var req FrontendErrorRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		logger.Error("Frontend unhandled error",
			slog.String("error.message", req.Message),
			slog.String("error.stack", req.Stack),
			slog.String("error.url", req.URL),
		)

		c.Status(http.StatusOK)
	}
}
