package server

import (
	"log/slog"

	"github.com/crimsonn/media_pipeline/internal/api/jobs"
	"github.com/crimsonn/media_pipeline/internal/api/pending"
	"github.com/crimsonn/media_pipeline/internal/api/playback"
	"github.com/crimsonn/media_pipeline/internal/api/transcoder"
	"github.com/crimsonn/media_pipeline/internal/config"
	"github.com/crimsonn/media_pipeline/internal/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter(
	logger *slog.Logger,
	config *config.Config,
	transcoderHandler *transcoder.Handler,
	pendingHandler *pending.Handler,
	jobsHandler *jobs.Handler,
	playbackHandler *playback.Handler,
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
	{
		pendingGroup := v1.Group("/pending")
		pendingGroup.GET("/files", pendingHandler.GetPendingFiles)
		pendingGroup.POST("/files", pendingHandler.EnqueuePendingFile)
	}
	{
		jobsGroup := v1.Group("/jobs")
		jobsGroup.GET("/latest", jobsHandler.GetLatestJobs)
	}
	{
		playbackGroup := v1.Group("/playback")
		playbackGroup.GET("/outputs", playbackHandler.ListOutputs)
		playbackGroup.GET("/hls/*filepath", playbackHandler.ServeFile)
	}

	return router
}
