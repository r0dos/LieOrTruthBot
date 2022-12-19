package storage

import (
	"database/sql"
	"fmt"
)

type anyStorage struct {
	pool *sql.DB
}

func NewStorage(poll *sql.DB) (*anyStorage, error) {
	if err := up(poll); err != nil {
		return nil, fmt.Errorf("migration up: %v", err)
	}

	return &anyStorage{
		pool: poll,
	}, nil
}
