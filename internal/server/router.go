package server

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/crimsonn/media_pipeline/internal/api/transcoder"
	"github.com/crimsonn/media_pipeline/internal/config"
	"github.com/crimsonn/media_pipeline/internal/middleware"
	"github.com/crimsonn/media_pipeline/pkg/pb"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func SetupRouter(
	logger *slog.Logger,
	config *config.Config,
	transcoderService pb.TranscoderServiceClient,
	healthService healthpb.HealthClient,
	db *sql.DB,
	transcoderHandler *transcoder.Handler,
) *gin.Engine {
	router := gin.New()
	if config.Environment == "development" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router.Use(middleware.Logger())
	router.Use(gin.Recovery())
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	router.Use(cors.New(corsConfig))

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "OK"})
		})
		v1.GET("/transcoder/health", func(c *gin.Context) {
			response, err := healthService.Check(c.Request.Context(), &healthpb.HealthCheckRequest{
				Service: "",
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, response)
		})
	}

	return router
}
