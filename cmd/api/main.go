package main

import (
	"bulk-email-platform/internal/auth"
	"bulk-email-platform/internal/handler"
	"bulk-email-platform/internal/repository"
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

	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	repo, err := repository.NewPostGresRepo(dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer repo.Close()

	r := gin.Default()

	authHandler := handler.NewAuthHandler(repo)
	uploadhandler := handler.NewUploadHandler(repo)
	emailHandler := handler.NewEmailHandler(repo)
	llmHandler := handler.NewLLMHandler(groqKey)

	r.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/logout", authHandler.Logout)
	}

	protected := r.Group("/api")
	protected.Use(auth.AuthMiddleware())
	{
		protected.GET("/me", authHandler.Me)

		protected.POST("/upload", uploadhandler.Handle)

		emailRoutes := r.Group("/email")
		{
			emailRoutes.POST("/send", emailHandler.SendSingle)
			emailRoutes.POST("/send-batch/:fileId", emailHandler.SendBatch)
		}

		llmRoutes := r.Group("llm")
		{
			llmRoutes.POST("/generate", llmHandler.Generate)
			llmRoutes.GET("/health", llmHandler.Health)
		}
	}

	r.Run(":8080")
}
