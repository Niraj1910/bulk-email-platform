package handler

import (
	"bulk-email-platform/internal/domain"
	"bulk-email-platform/internal/repository"
	"bulk-email-platform/pkg/excel"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UploadHandler struct {
	repo *repository.PostGresRepo
}

// Constuctor
func NewUploadHandler(repo *repository.PostGresRepo) *UploadHandler {
	return &UploadHandler{
		repo: repo,
	}
}

type UploadResponse struct {
	Message     string              `json:"message"`
	FileName    string              `json:"fileName"`
	ValidRows   int                 `json:"validRows"`
	InvalidRows int                 `json:"invalidRows"`
	ValidData   []map[string]string `json:"validData"`
	InvalidData []InvalidRow        `json:"invalidData"`
}

type InvalidRow struct {
	RowNumber string `json:"rowNumber"`
	Errors    string `json:"errors"`
}

func (h *UploadHandler) Handle(c *gin.Context) {

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	if !strings.HasSuffix(strings.ToLower(file.Filename), ".xlsx") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only .xlsx files are supported for now"})
		return
	}

	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		fmt.Println("failed to create ./uploads directory")
	}
	tempPath := filepath.Join(uploadDir, fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename))

	if err := c.SaveUploadedFile(file, tempPath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	defer os.Remove(tempPath)

	parser := excel.NewParser()

	headers, validRows, invalidRows, err := parser.ParseFile(tempPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse file" + err.Error()})
		return
	}

	dbFile := domain.File{
		UserID:      userID,
		FileName:    file.Filename,
		FileSize:    string(file.Size),
		TotalRows:   len(validRows) + len(invalidRows),
		InvalidRows: len(invalidRows),
		ValidRows:   len(validRows),
		Status:      "uploaded",
	}

	err = h.repo.CreateFile(c, &dbFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create file record"})
		return
	}

	var columns []domain.FileColumn
	for i, header := range headers {
		columns = append(columns, domain.FileColumn{
			ColumnName:    header,
			ColumnIndex:   i,
			IsEmailColumn: header == "to" || header == "from",
			IsRequired:    header == "to" || header == "from",
		})
	}

	err = h.repo.CreateFileColoummns(c, dbFile.ID, columns)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save columns"})
		return
	}

	var emailRows []*domain.EmailRow

	for _, row := range validRows {
		rowNumber, _ := strconv.Atoi(row["_row_number"])

		emailRow := &domain.EmailRow{
			FileID:          dbFile.ID,
			ToEmail:         row["to"],
			FromEmail:       row["from"],
			Subject:         row["subject"],
			Message:         row["message"],
			Description:     row["description"],
			Context:         row["context"],
			RowNumber:       rowNumber,
			IsValid:         true,
			ValidationError: "",
			Status:          "pending",
		}
		emailRows = append(emailRows, emailRow)
	}

	for _, row := range invalidRows {
		rowNumber, _ := strconv.Atoi(row["_row_number"])

		emailRow := &domain.EmailRow{
			FileID:          dbFile.ID,
			ToEmail:         row["to"],
			FromEmail:       row["from"],
			Subject:         row["subject"],
			Message:         row["message"],
			Description:     row["description"],
			Context:         row["context"],
			RowNumber:       rowNumber,
			IsValid:         false,
			ValidationError: row["_errors"],
			GeneratedByLLM:  false,
			Status:          "failed",
		}
		emailRows = append(emailRows, emailRow)
	}

	if err := h.repo.CreateEmailRowsBatch(c.Request.Context(), emailRows); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save email rows"})
		return
	}

	if err := h.repo.MarkFileAsProcessed(c.Request.Context(), dbFile.ID, len(validRows), len(invalidRows)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update file stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "file processed successfully",
		"file_id":      dbFile.ID,
		"filename":     file.Filename,
		"total_rows":   len(validRows) + len(invalidRows),
		"valid_rows":   len(validRows),
		"invalid_rows": len(invalidRows),
		"headers":      headers,
	})
}
