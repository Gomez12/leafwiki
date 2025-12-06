package api

import (
	"net/http"

	"github.com/Gomez12/wiki/internal/wiki"
	"github.com/gin-gonic/gin"
)

func GetTreeHandler(w *wiki.Wiki) gin.HandlerFunc {
	return func(c *gin.Context) {
		tree := w.GetTree()
		c.JSON(http.StatusOK, ToAPINode(tree, ""))
	}
}
