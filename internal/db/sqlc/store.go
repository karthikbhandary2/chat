package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SQLStore struct {
	*Queries
	connPool *pgxpool.Pool
}

type Store interface {
	Querier
	CreateConversationWithParticipants(ctx context.Context, convType string, participantIDs []uuid.UUID) (Conversation, error)
}

func NewStore(connPool *pgxpool.Pool) Store {
	return &SQLStore{
		connPool: connPool,
		Queries:  New(connPool),
	}
}

// execTx runs fn inside a database transaction, committing if fn returns nil
// and rolling back if it returns an error.
func (s *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := s.connPool.Begin(ctx)
	if err != nil {
		return err
	}

	q := New(tx)
	if err := fn(q); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx error: %v, rollback error: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit(ctx)
}

func (s *SQLStore) CreateConversationWithParticipants(ctx context.Context, convType string, participantIDs []uuid.UUID) (Conversation, error) {
	var conversation Conversation

	err := s.execTx(ctx, func(q *Queries) error {
		var err error
		conversation, err = q.CreateConversation(ctx, convType)
		if err != nil {
			return err
		}

		for _, participantID := range participantIDs {
			_, err := q.AddParticipant(ctx, AddParticipantParams{
				ConversationID: conversation.ID,
				UserID:         participantID,
			})
			if err != nil {
				return err
			}
		}

		return nil
	})

	return conversation, err
}
