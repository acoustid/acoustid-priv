package priv

import (
	"testing"
	"database/sql"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
)

func getTestAccount(t *testing.T, db *sql.DB) Account {
	service := NewService(db)
	account, err := service.GetAccountByApiKey("api_key")
	require.NoError(t, err)
	return account
}

func TestAccount_Repository(t *testing.T) {
	account := getTestAccount(t, connectToDB(t))
	repo := account.Repository()
	assert.NotNil(t, repo)
}
