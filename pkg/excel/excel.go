package excel

import (
	"bulk-email-platform/pkg/validator"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParseFile(fileName string) ([]string, []map[string]string, []map[string]string, error) {

	file, err := excelize.OpenFile(fileName)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to open excel file: %w", err)
	}

	defer file.Close()

	sheets := file.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil, nil, fmt.Errorf("no sheets found in the excel file")
	}

	firstSheet := sheets[0]
	rows, err := file.GetRows(firstSheet)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read rows: %w", err)
	}

	if len(rows) < 2 {
		return nil, nil, nil, nil
	}

	// process headers - skip empty ones and clean them
	headers := make([]string, 0)
	headerIndices := make([]int, 0)

	for i, h := range rows[0] {
		headerClean := strings.ToLower(strings.TrimSpace(h))

		if headerClean != "" && !strings.EqualFold(headerClean, "row_number") {
			headers = append(headers, headerClean)
			headerIndices = append(headerIndices, i)
		}
	}

	var validRows []map[string]string
	var invalidRows []map[string]string

	for i := 1; i < len(rows); i++ {
		row := rows[i]
		rowData := make(map[string]string)

		rowData["_row_number"] = fmt.Sprintf("%d", i+1)
		rowData["_is_valid"] = "true"
		rowData["_errors"] = ""

		for idx, headerIdx := range headerIndices {
			if headerIdx < len(row) {
				// clean up the value to be just 'text'
				value := strings.TrimSpace(row[headerIdx])

				if headers[idx] == "to" || headers[idx] == "from" {
					normalized, isValid := validator.ValidateEmail(value)
					value = normalized

					if !isValid || value == "" {
						rowData["_is_valid"] = "false"
						rowData["_errors"] += `invalid "` + headers[idx] + `" email; `
					}

				} else {
					value = strings.TrimSpace(value)
				}

				if value != "" {
					rowData[headers[idx]] = value
				}
			}
		}

		if rowData["_is_valid"] == "true" {
			delete(rowData, "_is_valid")
			delete(rowData, "_errors")
			validRows = append(validRows, rowData)
		} else {
			invalidRows = append(invalidRows, rowData)
		}
	}

	return headers, validRows, invalidRows, nil
}
