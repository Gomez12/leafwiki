package api

import (
	"net/http"

	"github.com/Gomez12/wiki/internal/wiki"
	"github.com/gin-gonic/gin"
)

func GetUsersHandler(wikiInstance *wiki.Wiki) gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := wikiInstance.GetUsers()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load users"})
			return
		}

		c.JSON(http.StatusOK, users)
	}
}
