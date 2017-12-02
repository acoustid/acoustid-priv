package priv

import (
	"database/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func getTestRepository(t *testing.T, db *sql.DB) Repository {
	account := getTestAccount(t, connectToDB(t))
	return account.Repository()
}

func TestRepository_Account(t *testing.T) {
	account := getTestAccount(t, connectToDB(t))
	repo := account.Repository()
	assert.Equal(t, account, repo.Account())
}

func TestRepository_Catalog(t *testing.T) {
	account := getTestAccount(t, connectToDB(t))
	repo := account.Repository()
	catalog := repo.Catalog("cat1")
	assert.NotNil(t, catalog)
}

func TestRepository_ListCatalogs(t *testing.T) {
	account := getTestAccount(t, connectToDB(t))
	repo := account.Repository()
	repo.Catalog("test1").CreateCatalog()
	repo.Catalog("test2").CreateCatalog()
	catalogs, err := repo.ListCatalogs()
	require.NoError(t, err)
	require.Equal(t, 2, len(catalogs))
	assert.Equal(t, "test1", catalogs[0].Name())
	assert.Equal(t, "test2", catalogs[1].Name())
}
