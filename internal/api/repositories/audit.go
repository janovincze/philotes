// Package repositories provides data access layer for API resources.
package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
)

// AuditRepository handles database operations for audit logs.
type AuditRepository struct {
	db *sql.DB
}

// NewAuditRepository creates a new AuditRepository.
func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Create creates a new audit log entry.
func (r *AuditRepository) Create(ctx context.Context, log *models.AuditLog) error {
	query := `
		INSERT INTO philotes.audit_logs (user_id, api_key_id, action, resource_type, resource_id, ip_address, user_agent, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	var detailsJSON []byte
	var err error
	if log.Details != nil {
		detailsJSON, err = json.Marshal(log.Details)
		if err != nil {
			return fmt.Errorf("failed to marshal details: %w", err)
		}
	}

	_, err = r.db.ExecContext(ctx, query,
		log.UserID,
		log.APIKeyID,
		log.Action,
		nullString(log.ResourceType),
		log.ResourceID,
		nullString(log.IPAddress),
		nullString(log.UserAgent),
		detailsJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// AuditListOptions contains options for listing audit logs.
type AuditListOptions struct {
	UserID       *uuid.UUID
	APIKeyID     *uuid.UUID
	Action       string
	ResourceType string
	ResourceID   *uuid.UUID
	StartTime    *time.Time
	EndTime      *time.Time
	Limit        int
	Offset       int
}

// List retrieves audit logs with optional filters.
func (r *AuditRepository) List(ctx context.Context, opts AuditListOptions) ([]models.AuditLog, int, error) {
	// Build query with filters
	baseQuery := `FROM philotes.audit_logs WHERE 1=1`
	args := []any{}
	argIdx := 1

	if opts.UserID != nil {
		baseQuery += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, *opts.UserID)
		argIdx++
	}
	if opts.APIKeyID != nil {
		baseQuery += fmt.Sprintf(" AND api_key_id = $%d", argIdx)
		args = append(args, *opts.APIKeyID)
		argIdx++
	}
	if opts.Action != "" {
		baseQuery += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, opts.Action)
		argIdx++
	}
	if opts.ResourceType != "" {
		baseQuery += fmt.Sprintf(" AND resource_type = $%d", argIdx)
		args = append(args, opts.ResourceType)
		argIdx++
	}
	if opts.ResourceID != nil {
		baseQuery += fmt.Sprintf(" AND resource_id = $%d", argIdx)
		args = append(args, *opts.ResourceID)
		argIdx++
	}
	if opts.StartTime != nil {
		baseQuery += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *opts.StartTime)
		argIdx++
	}
	if opts.EndTime != nil {
		baseQuery += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *opts.EndTime)
		argIdx++
	}

	// Get total count
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Get logs with pagination
	selectQuery := `SELECT id, user_id, api_key_id, action, resource_type, resource_id, ip_address, user_agent, details, created_at ` +
		baseQuery + " ORDER BY created_at DESC"

	if opts.Limit > 0 {
		selectQuery += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, opts.Limit)
		argIdx++
	}
	if opts.Offset > 0 {
		selectQuery += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, opts.Offset)
	}

	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var log models.AuditLog
		var userID, apiKeyID, resourceID sql.NullString
		var resourceType, ipAddress, userAgent sql.NullString
		var detailsJSON []byte

		err := rows.Scan(
			&log.ID,
			&userID,
			&apiKeyID,
			&log.Action,
			&resourceType,
			&resourceID,
			&ipAddress,
			&userAgent,
			&detailsJSON,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log row: %w", err)
		}

		if userID.Valid {
			id, _ := uuid.Parse(userID.String)
			log.UserID = &id
		}
		if apiKeyID.Valid {
			id, _ := uuid.Parse(apiKeyID.String)
			log.APIKeyID = &id
		}
		if resourceID.Valid {
			id, _ := uuid.Parse(resourceID.String)
			log.ResourceID = &id
		}
		log.ResourceType = resourceType.String
		log.IPAddress = ipAddress.String
		log.UserAgent = userAgent.String

		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &log.Details); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal details: %w", err)
			}
		}

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate audit logs: %w", err)
	}

	return logs, total, nil
}

// DeleteOlderThan deletes audit logs older than the specified time.
func (r *AuditRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	query := `DELETE FROM philotes.audit_logs WHERE created_at < $1`

	result, err := r.db.ExecContext(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old audit logs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}
