package pending

import (
	"net/http"

	"github.com/crimsonn/media_pipeline/internal/domain"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetPendingFiles(ctx *gin.Context) {
	files, err := h.service.GetPendingFiles(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, files)
}

func (h *Handler) EnqueuePendingFile(ctx *gin.Context) {
	var req domain.PendingEnqueueRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := h.service.EnqueuePendingFile(ctx.Request.Context(), req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "File enqueued successfully"})
}
