package handler

import (
	"bulk-email-platform/pkg/excel"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	// we will add dependencies
}

// Constuctor
func NewUploadHandler() *UploadHandler {
	return &UploadHandler{}
}

func (h *UploadHandler) Handle(c *gin.Context) {

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

	parser := &excel.Parser{}

	validRows, invalidRows, err := parser.ParseFile(tempPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse file" + err.Error()})
		return
	}

	fmt.Println("====== Excel Parsing Result =======")
	fmt.Println("File: ", file.Filename)
	fmt.Println("Total rows found: ", len(validRows)+len(invalidRows))
	fmt.Println("Valid Rows: ", len(validRows))
	fmt.Println("Invalid Rows: ", len(invalidRows))

	fmt.Printf("============================\n\n")

	for i, row := range validRows {

		fmt.Printf("=== Row %d (Excel Row %s) ===\n", i+1, row["_row_number"])

		for key, value := range row {

			if key == "_row_number" {
				fmt.Printf("  📍 Row %s\n", value)
			} else {
				fmt.Printf("  %s: %q\n", key, value)

			}

		}
		fmt.Printf("============================\n\n")

	}

	fmt.Printf("============================\n\n")

	if len(invalidRows) > 0 {
		fmt.Printf("❌ Invalid rows:\n")
		for _, invalid := range invalidRows {
			fmt.Printf("  📍 Excel Row %s: %s\n",
				invalid["_row_number"],
				invalid["_errors"])
		}
		fmt.Println()
	}

	fmt.Printf("============================\n\n")

	c.JSON(http.StatusOK, gin.H{
		"message":         "file processed successfully",
		"filename":        file.Filename,
		"valid_rows":      len(validRows),
		"invalid_rows":    len(invalidRows),
		"valid_preview":   validRows[:min(3, len(validRows))],
		"invalid_preview": invalidRows[:min(3, len(invalidRows))],
	})
}
