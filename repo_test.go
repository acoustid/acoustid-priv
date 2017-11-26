package priv

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"database/sql"
)

func getTestRepository(t *testing.T, db *sql.DB) Repository {
	account := getTestAccount(t, connectToDB(t))
	return account.Repository()
}

func TestRepo_Account(t *testing.T) {
	account := getTestAccount(t, connectToDB(t))
	repo := account.Repository()
	assert.Equal(t, account, repo.Account())
}

func TestRepo_Catalog(t *testing.T) {
	account := getTestAccount(t, connectToDB(t))
	repo := account.Repository()
	catalog := repo.Catalog("cat1")
	assert.NotNil(t, catalog)
}
