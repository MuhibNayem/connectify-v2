package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/adapters"
	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
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

	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(15 * time.Minute)

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
	CREATE INDEX IF NOT EXISTS idx_tenant ON notifications(tenant_id);
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

	dataJSON, err := toJSON(notification.Data)
	if err != nil {
		return err
	}

	templateJSON, err := toJSON(notification.TemplateData)
	if err != nil {
		return err
	}

	deliveredJSON, err := toJSON(notification.DeliveredAt)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO notifications (
			id, recipient_id, sender_id, type, title, body,
			image_url, action_url, priority, channels, data, read,
			delivered_at, failed_channels, template_id, template_data,
			expires_at, tenant_id, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,
			$13,$14,$15,$16,$17,$18,$19,$20
		)
	`

	_, err = p.db.ExecContext(ctx, query,
		notification.ID,
		notification.RecipientID,
		notification.SenderID,
		notification.Type,
		notification.Title,
		notification.Body,
		notification.ImageURL,
		notification.ActionURL,
		notification.Priority,
		pq.Array(notification.Channels),
		dataJSON,
		notification.Read,
		deliveredJSON,
		pq.Array(notification.FailedChannels),
		notification.TemplateID,
		templateJSON,
		toSQLTime(notification.ExpiresAt),
		notification.TenantID,
		now,
		now,
	)

	return err
}

func (p *PostgresStorage) Get(ctx context.Context, id string) (*adapters.Notification, error) {
	query := `
		SELECT id, recipient_id, sender_id, type, title, body,
			   image_url, action_url, priority, channels, data, read,
			   read_at, delivered_at, failed_channels, template_id, template_data,
			   expires_at, tenant_id, created_at, updated_at
		FROM notifications WHERE id = $1
	`

	var n adapters.Notification
	var createdAt, updatedAt time.Time
	var readAt sql.NullTime
	var expiresAt sql.NullTime
	var dataJSON, deliveredJSON, templateJSON []byte

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&n.ID,
		&n.RecipientID,
		&n.SenderID,
		&n.Type,
		&n.Title,
		&n.Body,
		&n.ImageURL,
		&n.ActionURL,
		&n.Priority,
		pq.Array(&n.Channels),
		&dataJSON,
		&n.Read,
		&readAt,
		&deliveredJSON,
		pq.Array(&n.FailedChannels),
		&n.TemplateID,
		&templateJSON,
		&expiresAt,
		&n.TenantID,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, adapters.ErrNotFound
		}
		return nil, err
	}

	if len(dataJSON) > 0 {
		_ = json.Unmarshal(dataJSON, &n.Data)
	}
	if len(deliveredJSON) > 0 {
		_ = json.Unmarshal(deliveredJSON, &n.DeliveredAt)
	}
	if len(templateJSON) > 0 {
		_ = json.Unmarshal(templateJSON, &n.TemplateData)
	}
	if readAt.Valid {
		readVal := readAt.Time.Format(time.RFC3339)
		n.ReadAt = &readVal
	}
	if expiresAt.Valid {
		expVal := expiresAt.Time.Format(time.RFC3339)
		n.ExpiresAt = &expVal
	}

	n.CreatedAt = createdAt.Format(time.RFC3339)
	n.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &n, nil
}

func (p *PostgresStorage) List(ctx context.Context, query *adapters.ListQuery) (*adapters.ListResult, error) {
	where := []string{"recipient_id = $1"}
	args := []interface{}{query.RecipientID}
	argPos := 1

	if query.Read != nil {
		argPos++
		where = append(where, fmt.Sprintf("read = $%d", argPos))
		args = append(args, *query.Read)
	}
	if len(query.Types) > 0 {
		argPos++
		where = append(where, fmt.Sprintf("type = ANY($%d)", argPos))
		args = append(args, pq.Array(query.Types))
	}
	if len(query.Channels) > 0 {
		argPos++
		where = append(where, fmt.Sprintf("channels && $%d", argPos))
		args = append(args, pq.Array(query.Channels))
	}
	if query.Priority != "" {
		argPos++
		where = append(where, fmt.Sprintf("priority = $%d", argPos))
		args = append(args, query.Priority)
	}
	if query.TenantID != "" {
		argPos++
		where = append(where, fmt.Sprintf("tenant_id = $%d", argPos))
		args = append(args, query.TenantID)
	}
	if query.Since != nil {
		if since, err := time.Parse(time.RFC3339, *query.Since); err == nil {
			argPos++
			where = append(where, fmt.Sprintf("created_at >= $%d", argPos))
			args = append(args, since)
		}
	}
	if query.Until != nil {
		if until, err := time.Parse(time.RFC3339, *query.Until); err == nil {
			argPos++
			where = append(where, fmt.Sprintf("created_at <= $%d", argPos))
			args = append(args, until)
		}
	}
	if query.Cursor != "" {
		if ts, cursorID, err := parseCursor(query.Cursor); err == nil {
			argPos++
			where = append(where, fmt.Sprintf("(created_at, id) < ($%d, $%d)", argPos, argPos+1))
			args = append(args, ts, cursorID)
			argPos++
		}
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 500 {
		limit = 500
	}

	orderDirection := "DESC"
	if query.Sort == "asc" {
		orderDirection = "ASC"
	}

	selectQuery := fmt.Sprintf(`
		SELECT id, recipient_id, sender_id, type, title, body,
			   image_url, action_url, priority, channels, data, read,
			   created_at, updated_at, delivered_at, failed_channels,
			   template_id, template_data, expires_at, tenant_id
		FROM notifications %s
		ORDER BY created_at %s, id %s
		LIMIT %d
	`, whereSQL, orderDirection, orderDirection, limit+1)

	rows, err := p.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*adapters.Notification
	for rows.Next() {
		var n adapters.Notification
		var createdAt, updatedAt time.Time
		var expiresAt sql.NullTime
		var dataJSON, deliveredJSON, templateJSON []byte

		if err := rows.Scan(
			&n.ID,
			&n.RecipientID,
			&n.SenderID,
			&n.Type,
			&n.Title,
			&n.Body,
			&n.ImageURL,
			&n.ActionURL,
			&n.Priority,
			pq.Array(&n.Channels),
			&dataJSON,
			&n.Read,
			&createdAt,
			&updatedAt,
			&deliveredJSON,
			pq.Array(&n.FailedChannels),
			&n.TemplateID,
			&templateJSON,
			&expiresAt,
			&n.TenantID,
		); err != nil {
			continue
		}

		if len(dataJSON) > 0 {
			_ = json.Unmarshal(dataJSON, &n.Data)
		}
		if len(deliveredJSON) > 0 {
			_ = json.Unmarshal(deliveredJSON, &n.DeliveredAt)
		}
		if len(templateJSON) > 0 {
			_ = json.Unmarshal(templateJSON, &n.TemplateData)
		}
		if expiresAt.Valid {
			expVal := expiresAt.Time.Format(time.RFC3339)
			n.ExpiresAt = &expVal
		}

		n.CreatedAt = createdAt.Format(time.RFC3339)
		n.UpdatedAt = updatedAt.Format(time.RFC3339)
		notifications = append(notifications, &n)
	}

	hasMore := len(notifications) > limit
	var nextCursor string
	if hasMore {
		last := notifications[len(notifications)-1]
		nextCursor = buildCursor(last.CreatedAt, last.ID)
		notifications = notifications[:len(notifications)-1]
	}

	return &adapters.ListResult{
		Notifications: notifications,
		Total:         int64(len(notifications)),
		NextCursor:    nextCursor,
		HasMore:       hasMore,
	}, nil
}

func (p *PostgresStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	query := "UPDATE notifications SET updated_at = $1"
	args := []interface{}{time.Now()}
	argPos := 1

	if read, ok := updates["read"].(bool); ok {
		argPos++
		query += fmt.Sprintf(", read = $%d", argPos)
		args = append(args, read)
		if read {
			argPos++
			query += fmt.Sprintf(", read_at = $%d", argPos)
			args = append(args, time.Now())
		}
	}

	argPos++
	query += fmt.Sprintf(" WHERE id = $%d", argPos)
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

	result, err := p.db.ExecContext(ctx, query, time.Now(), time.Now(), recipientID, pq.Array(ids))
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

	if notification.ID == "" {
		notification.ID = uuid.New().String()
	}
	now := time.Now()
	notification.CreatedAt = now.Format(time.RFC3339)
	notification.UpdatedAt = now.Format(time.RFC3339)

	dataJSON, err := toJSON(notification.Data)
	if err != nil {
		return err
	}
	templateJSON, err := toJSON(notification.TemplateData)
	if err != nil {
		return err
	}
	deliveredJSON, err := toJSON(notification.DeliveredAt)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO notifications (
			id, recipient_id, sender_id, type, title, body,
			image_url, action_url, priority, channels, data, read,
			delivered_at, failed_channels, template_id, template_data,
			expires_at, tenant_id, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,
			$13,$14,$15,$16,$17,$18,$19,$20
		)
	`,
		notification.ID, notification.RecipientID, notification.SenderID, notification.Type,
		notification.Title, notification.Body, notification.ImageURL, notification.ActionURL,
		notification.Priority, pq.Array(notification.Channels), dataJSON, notification.Read,
		deliveredJSON, pq.Array(notification.FailedChannels), notification.TemplateID, templateJSON,
		toSQLTime(notification.ExpiresAt), notification.TenantID, now, now,
	)
	if err != nil {
		return err
	}

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

	query := `INSERT INTO notifications (
			id, recipient_id, sender_id, type, title, body,
			image_url, action_url, priority, channels, data, read,
			delivered_at, failed_channels, template_id, template_data,
			expires_at, tenant_id, created_at, updated_at
		) VALUES `

	vals := []interface{}{}
	now := time.Now().Format(time.RFC3339)

	for i, n := range notifications {
		if n.ID == "" {
			n.ID = uuid.New().String()
		}
		n.CreatedAt = now
		n.UpdatedAt = now

		offset := i * 20
		query += fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d),",
			offset+1, offset+2, offset+3, offset+4, offset+5,
			offset+6, offset+7, offset+8, offset+9, offset+10,
			offset+11, offset+12, offset+13, offset+14, offset+15,
			offset+16, offset+17, offset+18, offset+19, offset+20,
		)

		dataJSON, _ := json.Marshal(n.Data)
		templateJSON, _ := json.Marshal(n.TemplateData)
		deliveredJSON, _ := json.Marshal(n.DeliveredAt)

		vals = append(vals,
			n.ID, n.RecipientID, n.SenderID, n.Type, n.Title,
			n.Body, n.ImageURL, n.ActionURL, n.Priority,
			pq.Array(n.Channels), dataJSON, n.Read,
			deliveredJSON, pq.Array(n.FailedChannels), n.TemplateID, templateJSON,
			toSQLTime(n.ExpiresAt), n.TenantID, n.CreatedAt, n.UpdatedAt,
		)
	}
	query = strings.TrimSuffix(query, ",")

	if _, err := tx.ExecContext(ctx, query, vals...); err != nil {
		return fmt.Errorf("failed to bulk insert notifications: %w", err)
	}

	if len(events) > 0 {
		outboxQuery := `INSERT INTO outbox (id, type, payload, timestamp, tenant_id) VALUES `
		outboxVals := []interface{}{}

		for i, e := range events {
			if e.ID == "" {
				e.ID = uuid.New().String()
			}
			e.Timestamp = now

			offset := i * 5
			outboxQuery += fmt.Sprintf("($%d,$%d,$%d,$%d,$%d),", offset+1, offset+2, offset+3, offset+4, offset+5)

			payloadJSON, _ := json.Marshal(e.Payload)
			outboxVals = append(outboxVals, e.ID, e.Type, payloadJSON, e.Timestamp, e.TenantID)
		}
		outboxQuery = strings.TrimSuffix(outboxQuery, ",")

		if _, err := tx.ExecContext(ctx, outboxQuery, outboxVals...); err != nil {
			return fmt.Errorf("failed to bulk insert outbox: %w", err)
		}
	}

	return tx.Commit()
}

func (p *PostgresStorage) GetPendingOutboxEvents(ctx context.Context, limit int) ([]*adapters.NotificationEvent, error) {
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

func toJSON(v interface{}) ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}
	return json.Marshal(v)
}

func fromJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func buildCursor(createdAt string, id string) string {
	return fmt.Sprintf("%s|%s", createdAt, id)
}

func parseCursor(cursor string) (time.Time, string, error) {
	parts := strings.Split(cursor, "|")
	if len(parts) != 2 {
		return time.Time{}, "", fmt.Errorf("invalid cursor")
	}
	ts, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return time.Time{}, "", err
	}
	return ts, parts[1], nil
}

func toSQLTime(ts *string) interface{} {
	if ts == nil || *ts == "" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339, *ts); err == nil {
		return parsed
	}
	return nil
}
