package mpostgres

import (
	"context"
	"insider/internal/model"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageService interface {
	GetUnsentMessages(ctx context.Context, limit int) ([]model.Message, error)
	UpdateMessageSent(ctx context.Context, id uint, messageID string) error
	GetSentMessages(ctx context.Context) ([]model.Message, error)
}

type message struct {
	pool *pgxpool.Pool
}

func NewMessageService(pool *pgxpool.Pool) MessageService {
	return &message{
		pool: pool,
	}
}

func (r *message) GetUnsentMessages(ctx context.Context, limit int) ([]model.Message, error) {
	var messages []model.Message

	query := `
		SELECT id, content, recipient_phone, sent, sent_at, message_id, created_at, updated_at 
		FROM messages 
		WHERE sent = $1 
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, query, false, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var msg model.Message
		var sentAt, createdAt, updatedAt *time.Time

		err := rows.Scan(
			&msg.ID,
			&msg.Content,
			&msg.RecipientPhone,
			&msg.Sent,
			&sentAt,
			&msg.MessageID,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, err
		}

		if sentAt != nil {
			msg.SentAt = *sentAt
		}
		if createdAt != nil {
			msg.CreatedAt = *createdAt
		}
		if updatedAt != nil {
			msg.UpdatedAt = *updatedAt
		}

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *message) UpdateMessageSent(ctx context.Context, id uint, messageID string) error {
	now := time.Now()
	query := `
		UPDATE messages 
		SET sent = $1, sent_at = $2, message_id = $3, updated_at = $4 
		WHERE id = $5
	`

	_, err := r.pool.Exec(ctx, query, true, now, messageID, now, id)
	return err
}

func (r *message) GetSentMessages(ctx context.Context) ([]model.Message, error) {
	var messages []model.Message

	query := `
		SELECT id, content, recipient_phone, sent, sent_at, message_id, created_at, updated_at 
		FROM messages 
		WHERE sent = $1
	`
	rows, err := r.pool.Query(ctx, query, true)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var msg model.Message
		var sentAt, createdAt, updatedAt *time.Time

		err := rows.Scan(
			&msg.ID,
			&msg.Content,
			&msg.RecipientPhone,
			&msg.Sent,
			&sentAt,
			&msg.MessageID,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, err
		}

		if sentAt != nil {
			msg.SentAt = *sentAt
		}
		if createdAt != nil {
			msg.CreatedAt = *createdAt
		}
		if updatedAt != nil {
			msg.UpdatedAt = *updatedAt
		}

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}
