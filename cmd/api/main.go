package main

import (
	"bulk-email-platform/internal/handler"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		fmt.Println("failed to load enviornment vairables")
	}

	groqKey := os.Getenv("GROQ_API_KEY")
	if groqKey == "" {
		log.Fatal("GROQ_API_KEY environment variable is required")
	}

	r := gin.Default()

	uploadhandler := handler.NewUploadHandler()
	llmHandler := handler.NewLLMHandler(groqKey)
	emailHandler := handler.NewEmailHandler()

	r.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/upload", uploadhandler.Handle)

	llmRoutes := r.Group("/api/llm")
	{
		llmRoutes.POST("/generate", llmHandler.Generate)
		llmRoutes.GET("/health", llmHandler.Health)
	}
	emailRoutes := r.Group("/api/email")
	{
		emailRoutes.POST("/send", emailHandler.SendSingle)
		emailRoutes.POST("/send-batch", emailHandler.SendBatch)
	}

	r.Run(":8080")
}
