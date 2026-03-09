package handler

import (
	"bulk-email-platform/internal/repository"
	"bulk-email-platform/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type EmailHandler struct {
	emailService *service.EmailService
	repo         *repository.PostGresRepo
}

func NewEmailHandler(repo *repository.PostGresRepo) *EmailHandler {
	return &EmailHandler{emailService: service.NewEmailService(), repo: repo}
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

	_, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

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

	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	userID, err := uuid.Parse(userID.(string))

	fileIDStr := ctx.Param("fileId")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "file_id is required",
		})
		return
	}

	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid file_id"})
		return
	}

	file, err := eh.repo.GetFileByID(ctx, fileID)
	if err != nil || file.UserID != userID {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	emails, err := eh.repo.GetPendingEmails(ctx, file.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get pending emails"})
		return
	}

	if len(emails) == 0 {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "no pending emails to send",
			"sent":    0,
			"failed":  0,
			"total":   0,
		})
		return
	}

	var successes, failures int

	for _, email := range emails {

		if err := eh.emailService.SendMail(service.EmailRequest{
			FromEmail: email.FromEmail,
			To:        email.ToEmail,
			Subject:   email.Subject,
			Message:   email.Message,
		}); err != nil {
			failures++
			eh.repo.UpdateEmailStatus(ctx.Request.Context(), email.ID, "failed", err.Error())
		} else {
			successes++
			eh.repo.UpdateEmailStatus(ctx.Request.Context(), email.ID, "sent", "")
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":   "Batch processing complete",
		"file_id":   fileID,
		"file_name": file.FileName,
		"sent":      successes,
		"failed":    failures,
		"total":     len(emails),
	})
}
