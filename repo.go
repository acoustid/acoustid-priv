package priv

import "database/sql"

type Repository interface {
	Account() Account
	Catalog(name string) Catalog
}

type RepositoryImpl struct {
	db      *sql.DB
	account *AccountImpl
}

func (repo *RepositoryImpl) Account() Account {
	return repo.account
}

func (repo *RepositoryImpl) Catalog(name string) Catalog {
	return &CatalogImpl{db: repo.db, repo: repo, name: name}
}
