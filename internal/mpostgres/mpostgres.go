package mpostgres

import (
	"context"
	"message-service/internal/model"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/useinsider/go-pkg/inslogger"
)

type MessageService interface {
	GetUnsentMessages(ctx context.Context, limit int) ([]model.Message, error)
	UpdateMessageSent(ctx context.Context, id uint) error
	GetSentMessages(ctx context.Context) ([]model.Message, error)
}

type message struct {
	pool   *pgxpool.Pool
	logger inslogger.Interface
}

func NewMessageService(pool *pgxpool.Pool, logger inslogger.Interface) MessageService {
	return &message{
		pool:   pool,
		logger: logger,
	}
}

func (r *message) GetUnsentMessages(ctx context.Context, limit int) ([]model.Message, error) {
	var messages []model.Message

	query := `
		SELECT id, content, recipient_phone, sent, sent_at, created_at, updated_at 
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

func (r *message) UpdateMessageSent(ctx context.Context, id uint) error {
	now := time.Now()
	query := `
        UPDATE messages 
        SET sent = $1, sent_at = $2, updated_at = $3 
        WHERE id = $4
    `

	_, err := r.pool.Exec(ctx, query, true, now, now, id)
	if err != nil {
		r.logger.Errorf("Failed to update message with ID %d: %v", id, err)
		return err
	}

	r.logger.Logf("Message with ID %d updated successfully", id)
	return nil
}

func (r *message) GetSentMessages(ctx context.Context) ([]model.Message, error) {
	var messages []model.Message

	query := `
		SELECT id, content, recipient_phone, sent, sent_at, created_at, updated_at 
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
