package api

import (
	"net/http"

	"github.com/Gomez12/wiki/internal/wiki"
	"github.com/gin-gonic/gin"
)

func UpdatePageHandler(w *wiki.Wiki) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req struct {
			Title   string `json:"title" binding:"required"`
			Slug    string `json:"slug" binding:"required"`
			Content string `json:"content" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}

		page, err := w.UpdatePage(id, req.Title, req.Slug, req.Content)
		if err != nil {
			respondWithError(c, err)
			return
		}

		c.JSON(http.StatusOK, ToAPIPage(page))
	}
}
