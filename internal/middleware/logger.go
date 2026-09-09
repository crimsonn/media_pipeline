package middleware

import (
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type LoggerConfig struct {
	Logger *slog.Logger
}

func DefaultLoggerConfig() *LoggerConfig {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	return &LoggerConfig{
		Logger: logger,
	}
}

func Logger() gin.HandlerFunc {
	configLogger := DefaultLoggerConfig()
	logger := configLogger.Logger
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set("request_id", requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()

		duration := time.Since(start)
		statusCode := c.Writer.Status()
		if raw != "" {
			path = path + "?" + raw
		}

		level := slog.LevelInfo
		if statusCode >= 500 {
			level = slog.LevelError
		} else if statusCode >= 400 {
			level = slog.LevelWarn
		}

		logger.Log(c.Request.Context(), level, "HTTP Request",
			slog.String("request_id", requestID),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", statusCode),
			slog.Duration("duration", duration),
			slog.String("duration_ms", duration.Round(time.Millisecond).String()),
			slog.String("client_ip", c.ClientIP()),
			slog.String("user_agent", c.Request.UserAgent()),
			slog.Int("response_size", c.Writer.Size()))

		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				logger.Error("Request error",
					slog.String("required_id", requestID),
					slog.String("error", e.Error()))
			}
		}
	}
}
