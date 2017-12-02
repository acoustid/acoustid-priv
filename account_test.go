package priv

import (
	"database/sql"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func getTestAccount(t *testing.T, db *sql.DB) Account {
	service := NewService(db)
	account, err := service.GetAccount(fmt.Sprintf("test:%s", t.Name()))
	require.NoError(t, err)
	return account
}

func TestAccount_Repository(t *testing.T) {
	account := getTestAccount(t, connectToDB(t))
	repo := account.Repository()
	assert.NotNil(t, repo)
}
