package api

import (
	"net/http"

	"github.com/Gomez12/wiki/internal/wiki"
	"github.com/gin-gonic/gin"
)

func SearchStatusHandler(wikiInstance *wiki.Wiki) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := wikiInstance.GetIndexingStatus()
		c.JSON(http.StatusOK, status)
	}
}
