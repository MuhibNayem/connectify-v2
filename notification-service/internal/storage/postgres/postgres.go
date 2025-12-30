package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
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

	CREATE TABLE IF NOT EXISTS outbox (
		id UUID PRIMARY KEY,
		type VARCHAR(255) NOT NULL,
		payload JSONB NOT NULL,
		timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
		tenant_id VARCHAR(255),
		locked_until TIMESTAMP DEFAULT NULL
	);

	CREATE TABLE IF NOT EXISTS delivery_states (
		notification_id UUID PRIMARY KEY,
		channels JSONB NOT NULL,
		overall_status VARCHAR(50) NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_recipient_created ON notifications(recipient_id, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_recipient_read ON notifications(recipient_id, read);
	CREATE INDEX IF NOT EXISTS idx_type ON notifications(type);
	CREATE INDEX IF NOT EXISTS idx_created_at ON notifications(created_at);
	CREATE INDEX IF NOT EXISTS idx_expires_at ON notifications(expires_at) WHERE expires_at IS NOT NULL;
	CREATE INDEX IF NOT EXISTS idx_outbox_timestamp ON outbox(timestamp);
	CREATE INDEX IF NOT EXISTS idx_outbox_locked ON outbox(locked_until);
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
			image_url, action_url, priority, channels, data, read, 
			template_id, template_data, tenant_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`
	// Note: We adhere to standard driver capabilities. If pq.Array is required, ensure lib/pq is imported.
	// For this implementation, we rely on the driver handling string slices for TEXT[] columns or similar.

	// Fix: arguments count. ID..UpdatedAt.
	_, err := p.db.ExecContext(ctx, query,
		notification.ID, notification.RecipientID, notification.SenderID, notification.Type,
		notification.Title, notification.Body, notification.ImageURL, notification.ActionURL,
		notification.Priority, notification.Channels,
		notification.Data, notification.Read, notification.TemplateID, notification.TemplateData,
		notification.TenantID, now, now,
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
		if err == sql.ErrNoRows {
			return nil, adapters.ErrNotFound
		}
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
	query := `
		INSERT INTO delivery_states (notification_id, channels, overall_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (notification_id) DO UPDATE SET
			channels = EXCLUDED.channels,
			overall_status = EXCLUDED.overall_status,
			updated_at = EXCLUDED.updated_at
	`
	channelsJSON, err := toJSON(state.Channels)
	if err != nil {
		return err
	}

	_, err = p.db.ExecContext(ctx, query,
		state.NotificationID, channelsJSON, state.OverallStatus, state.CreatedAt, time.Now())
	return err
}

func (p *PostgresStorage) GetDeliveryState(ctx context.Context, notificationID string) (*models.NotificationDeliveryState, error) {
	query := `SELECT notification_id, channels, overall_status, created_at, updated_at FROM delivery_states WHERE notification_id = $1`

	var state models.NotificationDeliveryState
	var channelsData []byte

	err := p.db.QueryRowContext(ctx, query, notificationID).Scan(
		&state.NotificationID, &channelsData, &state.OverallStatus, &state.CreatedAt, &state.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if err := fromJSON(channelsData, &state.Channels); err != nil {
		return nil, err
	}
	return &state, nil
}

func (p *PostgresStorage) CreateWithOutbox(ctx context.Context, notification *adapters.Notification, event *adapters.NotificationEvent) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Insert Notification
	if notification.ID == "" {
		notification.ID = uuid.New().String()
	}
	now := time.Now()
	notification.CreatedAt = now.Format(time.RFC3339)
	notification.UpdatedAt = now.Format(time.RFC3339)

	_, err = tx.ExecContext(ctx, `
		INSERT INTO notifications (
			id, recipient_id, sender_id, type, title, body,
			image_url, action_url, priority, channels, data, read, 
			template_id, template_data, tenant_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`,
		notification.ID, notification.RecipientID, notification.SenderID, notification.Type,
		notification.Title, notification.Body, notification.ImageURL, notification.ActionURL,
		notification.Priority, notification.Channels, notification.Data, notification.Read,
		notification.TemplateID, notification.TemplateData, notification.TenantID, now, now,
	)
	if err != nil {
		return err
	}

	// 2. Insert Outbox Event
	payloadJSON, err := toJSON(event.Payload)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO outbox (id, type, payload, timestamp, tenant_id)
		VALUES ($1, $2, $3, $4, $5)
	`, event.ID, event.Type, payloadJSON, time.Now(), event.TenantID)

	if err != nil {
		return err
	}

	return tx.Commit()
}

func (p *PostgresStorage) CreateBatchWithOutbox(ctx context.Context, notifications []*adapters.Notification, events []*adapters.NotificationEvent) error {
	if len(notifications) == 0 {
		return nil
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Bulk Insert Notifications
	// Construct Multi-Value Insert
	query := `INSERT INTO notifications (
			id, recipient_id, sender_id, type, title, body,
			image_url, action_url, priority, channels, data, read, 
			template_id, template_data, tenant_id, created_at, updated_at
		) VALUES `

	vals := []interface{}{}
	now := time.Now().Format(time.RFC3339)

	for i, n := range notifications {
		if n.ID == "" {
			n.ID = uuid.New().String()
		}
		n.CreatedAt = now
		n.UpdatedAt = now

		// Parameter placeholders (e.g., $1, $2...)
		offset := i * 17
		query += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d),",
			offset+1, offset+2, offset+3, offset+4, offset+5, offset+6,
			offset+7, offset+8, offset+9, offset+10, offset+11, offset+12,
			offset+13, offset+14, offset+15, offset+16, offset+17)

		vals = append(vals,
			n.ID, n.RecipientID, n.SenderID, n.Type, n.Title, n.Body,
			n.ImageURL, n.ActionURL, n.Priority, n.Channels, n.Data, n.Read,
			n.TemplateID, n.TemplateData, n.TenantID, n.CreatedAt, n.UpdatedAt,
		)
	}
	// Remove trailing comma
	query = query[:len(query)-1]

	if _, err := tx.ExecContext(ctx, query, vals...); err != nil {
		return fmt.Errorf("failed to bulk insert notifications: %w", err)
	}

	// 2. Bulk Insert Outbox Events
	if len(events) > 0 {
		outboxQuery := `INSERT INTO outbox (id, type, payload, timestamp, tenant_id) VALUES `
		outboxVals := []interface{}{}

		for i, e := range events {
			// Ensure Event ID matches Notification ID if linked, or new UUID
			if e.ID == "" {
				e.ID = uuid.New().String()
			}
			e.Timestamp = now

			offset := i * 5
			outboxQuery += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d),",
				offset+1, offset+2, offset+3, offset+4, offset+5)

			payloadJson, _ := json.Marshal(e.Payload)

			outboxVals = append(outboxVals,
				e.ID, e.Type, payloadJson, e.Timestamp, e.TenantID,
			)
		}
		outboxQuery = outboxQuery[:len(outboxQuery)-1]

		if _, err := tx.ExecContext(ctx, outboxQuery, outboxVals...); err != nil {
			return fmt.Errorf("failed to bulk insert outbox: %w", err)
		}
	}

	return tx.Commit()
}

func (p *PostgresStorage) GetPendingOutboxEvents(ctx context.Context, limit int) ([]*adapters.NotificationEvent, error) {
	// Atomic Lease: Update locked_until for items that are not locked or lock expired
	// Skip locked items to avoid waiting
	query := `
		UPDATE outbox
		SET locked_until = NOW() + INTERVAL '2 minutes'
		WHERE id IN (
			SELECT id
			FROM outbox
			WHERE locked_until IS NULL OR locked_until < NOW()
			ORDER BY timestamp ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, type, payload, timestamp, tenant_id
	`

	rows, err := p.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*adapters.NotificationEvent
	for rows.Next() {
		var e adapters.NotificationEvent
		var payloadData []byte
		var ts time.Time

		if err := rows.Scan(&e.ID, &e.Type, &payloadData, &ts, &e.TenantID); err != nil {
			return nil, err
		}

		e.Timestamp = ts.Format(time.RFC3339)
		if err := fromJSON(payloadData, &e.Payload); err != nil {
			return nil, err
		}
		events = append(events, &e)
	}

	return events, nil
}

func (p *PostgresStorage) DeleteOutboxEvent(ctx context.Context, eventID string) error {
	_, err := p.db.ExecContext(ctx, "DELETE FROM outbox WHERE id = $1", eventID)
	return err
}

func (p *PostgresStorage) Close() error {
	return p.db.Close()
}

// Helpers

func toJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func fromJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
