CREATE TABLE IF NOT EXISTS messages (
    id SERIAL PRIMARY KEY,
    content TEXT NOT NULL,
    recipient_phone VARCHAR(20) NOT NULL,
    sent BOOLEAN DEFAULT FALSE,
    sent_at TIMESTAMP,
    message_id VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_messages_sent ON messages(sent);
