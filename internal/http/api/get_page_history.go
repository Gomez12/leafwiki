package api

import (
	"net/http"

	"github.com/Gomez12/wiki/internal/wiki"
	"github.com/gin-gonic/gin"
)

func GetPageHistoryHandler(w *wiki.Wiki) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Query("path")
		if path == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing path"})
			return
		}

		history, currentHash, err := w.GetPageHistory(path)
		if err != nil {
			respondWithError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"history":     history,
			"currentHash": currentHash,
		})
	}
}
