package api

import (
	"net/http"

	"github.com/Gomez12/wiki/internal/wiki"
	"github.com/gin-gonic/gin"
)

func SortPagesHandler(w *wiki.Wiki) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req struct {
			OrderedIds []string `json:"orderedIds"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		if err := w.SortPages(id, req.OrderedIds); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sort pages"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Pages sorted successfully"})
	}
}
