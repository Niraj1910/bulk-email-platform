package repository

import (
	"bulk-email-platform/internal/domain"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostGresRepo struct {
	pool *pgxpool.Pool
}

func NewPostGresRepo(connString string) (*PostGresRepo, error) {

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour * 1
	config.MaxConnIdleTime = time.Minute * 30
	config.HealthCheckPeriod = time.Minute * 2

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	fmt.Println("Connected to database successfully")
	return &PostGresRepo{
		pool: pool,
	}, nil
}

func (pr *PostGresRepo) Close() {
	if pr.pool != nil {
		pr.pool.Close()
		fmt.Println(" Database connection pool closed")
	}
}

func (pr *PostGresRepo) CreateNewUser(ctx context.Context, user *domain.User) error {

	query := `INSERT INTO 
	users (id, email, password_hash, full_name, created_at, updated_at)
	values ($1, $2, $3, $4, NOW(),NOW())`

	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	_, err := pr.pool.Exec(ctx, query, user.ID, user.Email, user.PasswordHash, user.FullName)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (pr *PostGresRepo) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {

	query := `SELECT id, email, full_name, created_at, updated_at FROM users WHERE email = $1`

	var user domain.User
	err := pr.pool.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email, &user.FullName, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return domain.User{}, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (pr *PostGresRepo) CreateFile(ctx context.Context, file *domain.File) error {

	query := `INSERT INTO 
	files (id, file_name, file_size, total_rows, valid_rows, invalid_rows, status, created_at) 
	VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`

	if file.ID == uuid.Nil {
		file.ID = uuid.New()
	}
	if file.Status == "" {
		file.Status = "uploaded"
	}

	_, err := pr.pool.Exec(ctx, query, file.ID, file.FileName, file.FileSize, file.TotalRows, file.ValidRows, file.InvalidRows, file.Status)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	return nil
}

func (pr *PostGresRepo) GetFileByID(ctx context.Context, id uuid.UUID) (domain.File, error) {

	query := `SELECT id, user_id, file_name, file_size, total_rows, valid_rows, invalid_rows, status, created_at, processed_at FROM files where ID = $1`

	var file domain.File

	err := pr.pool.QueryRow(ctx, query, id).Scan(&file.ID, &file.UserID, &file.FileName, &file.FileSize, &file.TotalRows, &file.ValidRows, &file.InvalidRows, &file.Status, &file.CreatedAt, &file.ProcessedAt)

	if err != nil {
		return domain.File{}, fmt.Errorf("failed to get file: %w", err)
	}

	return file, nil
}

func (pr *PostGresRepo) MarkFileAsProcessed(ctx context.Context, fileID uuid.UUID, validRows, invalidRows int) error {

	query := `UPDATE files SET valid_rows = $1, invalid_rows = $2, status = 'completed', processed_at = NOW() WHERE id = $3`

	commandTag, err := pr.pool.Exec(ctx, query, validRows, invalidRows, fileID)
	if err != nil {
		return fmt.Errorf("failed to mark file as processed: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("no file found with ID %d ", fileID)
	}
	return nil
}

func (pr *PostGresRepo) CreateFileColoummns(ctx context.Context, fileID uuid.UUID, columns []domain.FileColumn) error {

	if len(columns) == 0 {
		return fmt.Errorf("column not found")
	}

	batch := pgx.Batch{}

	for _, column := range columns {
		if column.ID == uuid.Nil {
			column.ID = uuid.New()
		}
		column.FileID = fileID

		query := `INSERT INTO 
		file_columns (id, file_id, column_name, column_index, is_email_column, is_required, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6, NOW())`

		batch.Queue(query, column.ID, column.FileID, column.ColumnName, column.ColumnIndex, column.IsEmailColumn, column.IsRequired)
	}

	batchResults := pr.pool.SendBatch(ctx, &batch)
	defer batchResults.Close()

	for i := 0; i < len(columns); i++ {
		_, err := batchResults.Exec()
		if err != nil {
			return fmt.Errorf("failed to insert column at index %d: %w", i, err)
		}
	}
	return nil
}

func (pr *PostGresRepo) GetFileColumns(ctx context.Context, fileID uuid.UUID) ([]*domain.FileColumn, error) {

	query := `
		SELECT id, file_id, column_name, column_index, is_email_column, is_required, created_at
		FROM file_columns
		WHERE file_id = $1
		ORDER BY column_index
	`

	rows, err := pr.pool.Query(ctx, query, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to query file columns: %w", err)
	}
	defer rows.Close()

	var columns []*domain.FileColumn
	for rows.Next() {
		var col domain.FileColumn
		err := rows.Scan(
			&col.ID, &col.FileID, &col.ColumnName, &col.ColumnIndex,
			&col.IsEmailColumn, &col.IsRequired, &col.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}
		columns = append(columns, &col)
	}

	return columns, nil
}

func (pr *PostGresRepo) CreateEmailRowsBatch(ctx context.Context, emails []*domain.EmailRow) error {
	if len(emails) == 0 {
		return nil
	}

	batch := &pgx.Batch{}

	for _, email := range emails {
		if email.ID == uuid.Nil {
			email.ID = uuid.New()
		}
		if email.Status == "" {
			email.Status = "pending"
		}

		query := `
			INSERT INTO email_rows (
				id, file_id, to_email, from_email, subject, message,
				description, context, row_number, is_valid, validation_error,
				generated_by_llm, status, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW())
		`

		batch.Queue(query,
			email.ID, email.FileID, email.ToEmail, email.FromEmail,
			email.Subject, email.Message, email.Description, email.Context,
			email.RowNumber, email.IsValid, email.ValidationError,
			email.GeneratedByLLM, email.Status,
		)
	}

	br := pr.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(emails); i++ {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("failed to insert email at index %d: %w", i, err)
		}
	}

	return nil
}

func (pr *PostGresRepo) GetPendingEmails(ctx context.Context, fileID uuid.UUID) ([]*domain.EmailRow, error) {
	query := `
		SELECT id, file_id, to_email, from_email, subject, message, description, context, row_number, is_valid, validation_error, generated_by_llm, status, created_at
		FROM email_rows
		WHERE file_id = $1 AND status = 'pending' AND is_valid = true
		ORDER BY row_number
	`

	rows, err := pr.pool.Query(ctx, query, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending emails: %w", err)
	}
	defer rows.Close()

	var emails []*domain.EmailRow
	for rows.Next() {
		var email domain.EmailRow
		err := rows.Scan(
			&email.ID, &email.FileID, &email.ToEmail, &email.FromEmail,
			&email.Subject, &email.Message, &email.Description, &email.Context,
			&email.RowNumber, &email.IsValid, &email.ValidationError,
			&email.GeneratedByLLM, &email.Status, &email.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan email row: %w", err)
		}
		emails = append(emails, &email)
	}

	return emails, nil
}

func (pr *PostGresRepo) UpdateEmailStatus(ctx context.Context, emailID uuid.UUID, status string, deliveryError string) error {
	query := `
		UPDATE email_rows
		SET status = $1, delivery_error = $2, sent_at = NOW()
		WHERE id = $3
	`

	commandTag, err := pr.pool.Exec(ctx, query, status, deliveryError, emailID)
	if err != nil {
		return fmt.Errorf("failed to update email status: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("no email found with ID: %s", emailID)
	}

	return nil
}

func (pr *PostGresRepo) GetEmailStats(ctx context.Context, fileID uuid.UUID) (*domain.EmailStats, error) {
	query := `
		SELECT 	COUNT(*) as total,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
			COUNT(CASE WHEN status = 'sent' THEN 1 END) as sent,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
		FROM email_rows
		WHERE file_id = $1
	`

	var stats domain.EmailStats
	err := pr.pool.QueryRow(ctx, query, fileID).Scan(
		&stats.Total, &stats.Pending, &stats.Sent, &stats.Failed,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get email stats: %w", err)
	}

	return &stats, nil
}
