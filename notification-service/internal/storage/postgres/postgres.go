package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/adapters"
	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/models"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(connStr string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Create table if not exists
	schema := `
	CREATE TABLE IF NOT EXISTS notifications (
		id UUID PRIMARY KEY,
		recipient_id VARCHAR(255) NOT NULL,
		sender_id VARCHAR(255),
		type VARCHAR(100) NOT NULL,
		title TEXT,
		body TEXT,
		image_url TEXT,
		action_url TEXT,
		priority VARCHAR(20) DEFAULT 'normal',
		channels TEXT[],
		data JSONB,
		read BOOLEAN DEFAULT FALSE,
		read_at TIMESTAMP,
		delivered_at JSONB,
		failed_channels TEXT[],
		template_id VARCHAR(255),
		template_data JSONB,
		expires_at TIMESTAMP,
		tenant_id VARCHAR(255),
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_recipient_created ON notifications(recipient_id, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_recipient_read ON notifications(recipient_id, read);
	CREATE INDEX IF NOT EXISTS idx_type ON notifications(type);
	CREATE INDEX IF NOT EXISTS idx_created_at ON notifications(created_at);
	CREATE INDEX IF NOT EXISTS idx_expires_at ON notifications(expires_at) WHERE expires_at IS NOT NULL;
	`

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &PostgresStorage{db: db}, nil
}

func (p *PostgresStorage) Create(ctx context.Context, notification *adapters.Notification) error {
	if notification.ID == "" {
		notification.ID = uuid.New().String()
	}

	now := time.Now()
	notification.CreatedAt = now.Format(time.RFC3339)
	notification.UpdatedAt = now.Format(time.RFC3339)

	query := `
		INSERT INTO notifications (
			id, recipient_id, sender_id, type, title, body,
			image_url, action_url, priority, read, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := p.db.ExecContext(ctx, query,
		notification.ID,
		notification.RecipientID,
		notification.SenderID,
		notification.Type,
		notification.Title,
		notification.Body,
		notification.ImageURL,
		notification.ActionURL,
		notification.Priority,
		notification.Read,
		now,
		now,
	)

	return err
}

func (p *PostgresStorage) Get(ctx context.Context, id string) (*adapters.Notification, error) {
	query := `
		SELECT id, recipient_id, sender_id, type, title, body,
			   priority, read, created_at, updated_at
		FROM notifications WHERE id = $1
	`

	var n adapters.Notification
	var createdAt, updatedAt time.Time

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&n.ID, &n.RecipientID, &n.SenderID, &n.Type, &n.Title, &n.Body,
		&n.Priority, &n.Read, &createdAt, &updatedAt,
	)

	if err != nil {
		return nil, err
	}

	n.CreatedAt = createdAt.Format(time.RFC3339)
	n.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &n, nil
}

func (p *PostgresStorage) List(ctx context.Context, query *adapters.ListQuery) (*adapters.ListResult, error) {
	// Build WHERE clause
	where := "WHERE recipient_id = $1"
	args := []interface{}{query.RecipientID}
	argCount := 1

	if query.Read != nil {
		argCount++
		where += fmt.Sprintf(" AND read = $%d", argCount)
		args = append(args, *query.Read)
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM notifications " + where
	var total int64
	err := p.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Get notifications
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	selectQuery := fmt.Sprintf(`
		SELECT id, recipient_id, sender_id, type, title, body,
			   priority, read, created_at, updated_at
		FROM notifications %s
		ORDER BY created_at DESC
		LIMIT %d
	`, where, limit)

	rows, err := p.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*adapters.Notification
	for rows.Next() {
		var n adapters.Notification
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&n.ID, &n.RecipientID, &n.SenderID, &n.Type, &n.Title, &n.Body,
			&n.Priority, &n.Read, &createdAt, &updatedAt,
		)
		if err != nil {
			continue
		}

		n.CreatedAt = createdAt.Format(time.RFC3339)
		n.UpdatedAt = updatedAt.Format(time.RFC3339)
		notifications = append(notifications, &n)
	}

	return &adapters.ListResult{
		Notifications: notifications,
		Total:         total,
		HasMore:       int64(len(notifications)) >= int64(limit),
	}, nil
}

func (p *PostgresStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	query := "UPDATE notifications SET updated_at = $1"
	args := []interface{}{time.Now()}
	argCount := 1

	if read, ok := updates["read"].(bool); ok {
		argCount++
		query += fmt.Sprintf(", read = $%d", argCount)
		args = append(args, read)

		if read {
			argCount++
			query += fmt.Sprintf(", read_at = $%d", argCount)
			args = append(args, time.Now())
		}
	}

	argCount++
	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, id)

	_, err := p.db.ExecContext(ctx, query, args...)
	return err
}

func (p *PostgresStorage) Delete(ctx context.Context, id string) error {
	_, err := p.db.ExecContext(ctx, "DELETE FROM notifications WHERE id = $1", id)
	return err
}

func (p *PostgresStorage) GetUnreadCount(ctx context.Context, recipientID string) (int64, error) {
	var count int64
	err := p.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM notifications WHERE recipient_id = $1 AND read = FALSE",
		recipientID,
	).Scan(&count)
	return count, err
}

func (p *PostgresStorage) BatchMarkAsRead(ctx context.Context, recipientID string, ids []string) (int64, error) {
	query := `
		UPDATE notifications
		SET read = TRUE, read_at = $1, updated_at = $2
		WHERE recipient_id = $3 AND id = ANY($4)
	`

	result, err := p.db.ExecContext(ctx, query, time.Now(), time.Now(), recipientID, ids)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (p *PostgresStorage) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := p.db.ExecContext(ctx,
		"DELETE FROM notifications WHERE expires_at IS NOT NULL AND expires_at < $1",
		time.Now(),
	)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (p *PostgresStorage) HealthCheck(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

func (p *PostgresStorage) SaveDeliveryState(ctx context.Context, state *models.NotificationDeliveryState) error {
	return nil // Stub
}

func (p *PostgresStorage) GetDeliveryState(ctx context.Context, notificationID string) (*models.NotificationDeliveryState, error) {
	return nil, nil // Stub
}

func (p *PostgresStorage) CreateWithOutbox(ctx context.Context, notification *adapters.Notification, event *adapters.NotificationEvent) error {
	// TODO: Implement transaction
	return p.Create(ctx, notification)
}

func (p *PostgresStorage) GetPendingOutboxEvents(ctx context.Context, limit int) ([]*adapters.NotificationEvent, error) {
	return nil, nil // Stub
}

func (p *PostgresStorage) DeleteOutboxEvent(ctx context.Context, eventID string) error {
	return nil // Stub
}

func (p *PostgresStorage) Close() error {
	return p.db.Close()
}
