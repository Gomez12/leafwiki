package api

import (
	"net/http"

	"github.com/Gomez12/wiki/internal/wiki"
	"github.com/gin-gonic/gin"
)

func DeleteAssetHandler(w *wiki.Wiki) gin.HandlerFunc {
	return func(c *gin.Context) {
		pageID := c.Param("id")
		filename := c.Param("name")

		if filename == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing filename"})
			return
		}

		if err := w.DeleteAsset(pageID, filename); err != nil {
			respondWithError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "asset deleted"})
	}
}
