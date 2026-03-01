package handler

import (
	"bulk-email-platform/pkg/excel"
	"fmt"
	"maps"
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

	validRows, invalidRows, err := parser.ParseFile(tempPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse file" + err.Error()})
		return
	}

	// Log results
	fmt.Println("====== Excel Parsing Result =======")
	fmt.Println("File: ", file.Filename)
	fmt.Println("Total rows found: ", len(validRows)+len(invalidRows))
	fmt.Println("Valid Rows: ", len(validRows))
	fmt.Println("Invalid Rows: ", len(invalidRows))
	fmt.Printf("============================\n\n")

	// prepare for invalid rows
	invalidData := make([]InvalidRow, 0, len(invalidRows))
	for _, invalid := range invalidRows {

		cleanFormat := make(map[string]string)
		for k, v := range invalid {
			switch k {
			case "_row_number":
				cleanFormat[k] = v

			case "_errors":
				cleanFormat[k] = v
			}
			invalidData = append(invalidData, InvalidRow{RowNumber: cleanFormat["_row_number"], Errors: cleanFormat["_errors"]})
		}
		fmt.Printf("  📍 Excel Row %s: %s\n", invalid["_row_number"], invalid["_errors"])
	}

	// prepare for valid rows
	validData := make([]map[string]string, 0, len(validRows))
	for _, valid := range validRows {

		cleanFormat := maps.Clone(valid)
		cleanFormat["_row_number"] = valid["_row_number"]
		validData = append(validData, cleanFormat)
	}

	response := UploadResponse{
		Message:     "file processed successfully",
		FileName:    file.Filename,
		ValidRows:   len(validRows),
		InvalidRows: len(invalidRows),
		ValidData:   validData,
		InvalidData: invalidData,
	}

	// Dont forget to add "file Attachments" in the mail

	c.JSON(http.StatusOK, response)
}
