package priv

import (
	"database/sql"
)

type Repository interface {
	Account() Account
	Catalog(name string) Catalog
	ListCatalogs() ([]Catalog, error)
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

func (repo *RepositoryImpl) ListCatalogs() ([]Catalog, error) {
	rows, err := repo.db.Query(`SELECT id, name FROM catalog WHERE account_id = $1 ORDER BY name`, repo.account.id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var id int
	var name string
	var catalogs []Catalog
	for rows.Next() {
		err = rows.Scan(&id, &name)
		if err != nil {
			return nil, err
		}
		catalogs = append(catalogs, &CatalogImpl{db: repo.db, repo: repo, id: id, name: name})
	}
	return catalogs, nil
}
