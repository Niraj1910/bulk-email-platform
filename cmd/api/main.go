package main

import (
	"bulk-email-platform/internal/handler"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {

	r := gin.Default()

	uploadhandler := handler.NewUploadHandler()

	r.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/upload", uploadhandler.Handle)

	r.Run(":8080")
}
