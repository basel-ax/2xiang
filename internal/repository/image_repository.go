package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/swenro11/2xiang/internal/domain"
)

// ImageRepository defines the interface for image data access
type ImageRepository interface {
	GetReadyToGenerate(ctx context.Context) (*domain.Image, error)
	GetReadyToCheck(ctx context.Context) (*domain.Image, error)
	UpdateStatus(ctx context.Context, id int, status string) error
	UpdateUUID(ctx context.Context, id int, uuid string) error
	UpdateBase64(ctx context.Context, id int, base64 string) error
}

// PostgresImageRepository implements ImageRepository for PostgreSQL
type PostgresImageRepository struct {
	db *sql.DB
}

// NewPostgresImageRepository creates a new PostgreSQL image repository
func NewPostgresImageRepository(db *sql.DB) *PostgresImageRepository {
	return &PostgresImageRepository{db: db}
}

// GetReadyToGenerate retrieves an image ready for generation
func (r *PostgresImageRepository) GetReadyToGenerate(ctx context.Context) (*domain.Image, error) {
	query := `
		SELECT id, prompt
		FROM images
		WHERE status = 'ReadyToGenerate'
		AND prompt IS NOT NULL
		AND prompt != ''
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`

	var img domain.Image
	err := r.db.QueryRowContext(ctx, query).Scan(
		&img.ID,
		&img.Prompt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &img, nil
}

// GetReadyToCheck retrieves an image ready for status check
func (r *PostgresImageRepository) GetReadyToCheck(ctx context.Context) (*domain.Image, error) {
	query := `
		SELECT id, uuid
		FROM images
		WHERE status = 'Generate'
		AND uuid IS NOT NULL
		AND uuid != ''
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`

	var img domain.Image
	err := r.db.QueryRowContext(ctx, query).Scan(&img.ID, &img.UUID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &img, nil
}

// UpdateStatus updates the status of an image
func (r *PostgresImageRepository) UpdateStatus(ctx context.Context, id int, status string) error {
	query := `
		UPDATE images
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.ExecContext(ctx, query, status, time.Now(), id)
	return err
}

// UpdateUUID updates the UUID of an image
func (r *PostgresImageRepository) UpdateUUID(ctx context.Context, id int, uuid string) error {
	query := `
		UPDATE images
		SET uuid = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.ExecContext(ctx, query, uuid, time.Now(), id)
	return err
}

// UpdateBase64 updates the base64 data of an image
func (r *PostgresImageRepository) UpdateBase64(ctx context.Context, id int, base64 string) error {
	query := `
		UPDATE images
		SET base64 = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.ExecContext(ctx, query, base64, time.Now(), id)
	return err
}
