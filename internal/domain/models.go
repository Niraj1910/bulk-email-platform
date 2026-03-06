package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type File struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	FileName    string     `json:"file_name"`
	FileSize    string     `json:"file_size"`
	TotalRows   int        `json:"total_rows"`
	ValidRows   int        `json:"valid_rows"`
	InvalidRows int        `json:"invalid_rows"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	ProcessedAt *time.Time `json:"processed_at"`
}

type FileColumn struct {
	ID            uuid.UUID `json:"id"`
	FileID        uuid.UUID `json:"file_id"`
	ColumnName    string    `json:"column_name"`
	ColumnIndex   int       `json:"column_index"`
	IsEmailColumn bool      `json:"is_email_coloumn"`
	IsRequired    bool      `json:"is_required"`
	CreatedAt     time.Time `json:"created_at"`
}

type EmailRow struct {
	ID     uuid.UUID `json:"id"`
	FileID uuid.UUID `json:"file_id"`

	ToEmail   string `json:"to_email"`
	FromEmail string `json:"from_email"`
	Subject   string `json:"subject"`
	Message   string `json:"message"`

	Description string `json:"description"`
	Context     string `json:"context"`

	RowNumber       int    `json:"row_number"`
	IsValid         bool   `json:"is_valid"`
	ValidationError string `json:"validation_error,omitempty"`

	GeneratedByLLM bool `json:"generated_by_llm"`

	Status        string     `json:"status"`
	SentAt        *time.Time `json:"sent_at,omitempty"`
	DeliveryError string     `json:"delivery_error,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type EmailStats struct {
	Total   int `json:"total"`
	Pending int `json:"pending"`
	Sent    int `json:"sent"`
	Failed  int `json:"failed"`
}
