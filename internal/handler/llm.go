package handler

import (
	"bulk-email-platform/pkg/llm"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type RequestFromFrontend struct {
	Description string `json:"description" binding:"required"`
	Context     string `json:"context" binding:"required"`
}

type RespondToFrontend struct {
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type LLMHandler struct {
	client *llm.GroqClient
}

func NewLLMHandler(apikey string) *LLMHandler {
	return &LLMHandler{
		client: llm.NewGroqClient(apikey),
	}
}

func (h *LLMHandler) Generate(ctx *gin.Context) {
	var req RequestFromFrontend

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Both description and context are required",
		})
		return
	}

	subject, message, err := h.client.GenerateEmail(req.Description, req.Context)
	if err != nil {
		fmt.Println("could not generate email: error details -> ", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate email. Please try again.",
		})
		return
	}
	ctx.JSON(http.StatusOK, RespondToFrontend{
		Subject: subject,
		Message: message,
	})
}

func (h *LLMHandler) Health(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"status": "available",
		"model":  h.client.GetModel(),
	})
}
