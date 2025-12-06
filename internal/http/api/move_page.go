package api

import (
	"net/http"

	"github.com/Gomez12/wiki/internal/wiki"
	"github.com/gin-gonic/gin"
)

func MovePageHandler(w *wiki.Wiki) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req struct {
			NewParentID string `json:"parentId"`
		}

		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
			return
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}

		if err := w.MovePage(id, req.NewParentID); err != nil {
			respondWithError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Page moved"})
	}
}
