package playback

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListOutputs(c *gin.Context) {
	outputs, err := h.service.ListOutputs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, outputs)
}

func (h *Handler) ServeFile(c *gin.Context) {
	rel := strings.TrimPrefix(c.Param("filepath"), "/")
	if rel == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file path"})
		return
	}

	full, err := h.service.ResolveFile(rel)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	switch strings.ToLower(filepath.Ext(full)) {
	case ".m3u8":
		c.Header("Content-Type", "application/vnd.apple.mpegurl")
		c.Header("Cache-Control", "no-cache")
	case ".ts":
		c.Header("Content-Type", "video/mp2t")
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	case ".m4s":
		c.Header("Content-Type", "video/iso.segment")
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	default:
		if ct := mime.TypeByExtension(filepath.Ext(full)); ct != "" {
			c.Header("Content-Type", ct)
		}
	}

	c.File(full)
}
