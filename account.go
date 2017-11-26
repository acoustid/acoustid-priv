package priv

import "database/sql"

type Account interface {
	Repository() Repository
}

type AccountImpl struct {
	db *sql.DB
	id int
}

func (account *AccountImpl) Repository() Repository {
	repo := &RepositoryImpl{db: account.db, account: account}
	return repo
}
