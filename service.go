package priv

import (
	"database/sql"
)

type Service interface {
	GetAccount(externalID string) (Account, error)
}

type ServiceImpl struct {
	db *sql.DB
}

func NewService(db *sql.DB) Service {
	return &ServiceImpl{db: db}
}

func (s *ServiceImpl) GetAccount(externalID string) (Account, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `INSERT INTO account (external_id) VALUES ($1) ON CONFLICT (external_id) DO UPDATE SET external_id=EXCLUDED.external_id RETURNING id`
	row := tx.QueryRow(query, externalID)
	var id int
	err = row.Scan(&id)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &AccountImpl{db: s.db, id: id}, nil
}
