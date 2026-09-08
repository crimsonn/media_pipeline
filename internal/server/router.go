package server

import (
	"log/slog"

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
		transcoderGroup := v1.Group("/transcoder")
		transcoderGroup.GET("/renditions", transcoderHandler.GetRenditions)
		transcoderGroup.POST("/renditions", transcoderHandler.CreateRendition)
		transcoderGroup.DELETE("/renditions/:id", transcoderHandler.DeleteRendition)
		transcoderGroup.POST("/profiles", transcoderHandler.CreateProfile)
		transcoderGroup.GET("/profiles/:id", transcoderHandler.GetProfile)
		transcoderGroup.GET("/profiles", transcoderHandler.GetProfiles)
	}

	return router
}
