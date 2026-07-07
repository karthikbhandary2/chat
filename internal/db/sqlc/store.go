package db

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// SQLStore provides all functions to execute SQL queries and transactions
type SQLStore struct {
	*Queries
	connPool *pgxpool.Pool
}

type Store interface {
	Querier
}

// NewStore creates a new Store
func NewStore(connPool *pgxpool.Pool) Store {
	return &SQLStore{
		connPool: connPool,
		Queries:  New(connPool),
	}
}
