package storage

import (
	"database/sql"
	"fmt"
)

type AnyStorage struct {
	pool *sql.DB
}

func NewStorage(poll *sql.DB) (*AnyStorage, error) {
	if err := up(poll); err != nil {
		return nil, fmt.Errorf("migration up: %v", err)
	}

	return &AnyStorage{
		pool: poll,
	}, nil
}
