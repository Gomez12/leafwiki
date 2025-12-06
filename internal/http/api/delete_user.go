package api

import (
	"net/http"

	"github.com/Gomez12/wiki/internal/wiki"
	"github.com/gin-gonic/gin"
)

func DeleteUserHandler(wikiInstance *wiki.Wiki) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := wikiInstance.DeleteUser(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}
