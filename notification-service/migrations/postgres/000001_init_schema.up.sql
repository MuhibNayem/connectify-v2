CREATE TABLE IF NOT EXISTS schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL,
    PRIMARY KEY (version)
);

CREATE TABLE notifications (
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

CREATE INDEX idx_recipient_created ON notifications(recipient_id, created_at DESC);
CREATE INDEX idx_recipient_read ON notifications(recipient_id, read);
CREATE INDEX idx_type ON notifications(type);
