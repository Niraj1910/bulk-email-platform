package handler

import (
	"bulk-email-platform/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type EmailHandler struct {
	emailService *service.EmailService
}

func NewEmailHandler() *EmailHandler {
	return &EmailHandler{emailService: service.NewEmailService()}
}

type SendEmailRequest struct {
	To        string `json:"to" binding:"required"`
	FromEmail string `json:"from_email" binding:"required"`
	Subject   string `json:"subject" binding:"required"`
	Message   string `json:"message" binding:"required"`
}

type SendEmailResponse struct {
	Message string `jon:"message"`
	To      string `json:"to"`
}

func (eh *EmailHandler) SendSingle(ctx *gin.Context) {
	var req SendEmailRequest

	err := ctx.ShouldBindJSON(&req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: to, from_email, subject, and message are required",
		})
		return
	}

	err = eh.emailService.SendMail(service.EmailRequest{
		FromEmail: req.FromEmail,
		To:        req.To,
		Subject:   req.Subject,
		Message:   req.Message,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to send email: " + err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, SendEmailResponse{
		Message: "Email sent successfully",
		To:      req.To,
	})
}

func (eh *EmailHandler) SendBatch(ctx *gin.Context) {
	var requests []SendEmailRequest

	err := ctx.ShouldBindJSON(&requests)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: to, from_email, subject, and message are required",
		})
		return
	}

	emailReqs := make([]service.EmailRequest, len(requests))
	for i, req := range requests {
		emailReqs[i] = service.EmailRequest{
			FromEmail: req.FromEmail,
			To:        req.To,
			Subject:   req.Subject,
			Message:   req.Message,
		}
	}

	successes, failures := eh.emailService.SendBulkMails(emailReqs)

	ctx.JSON(http.StatusOK, gin.H{
		"message":   "Batch processing complete",
		"successes": successes,
		"failures":  failures,
	})
}
