package priv

import (
	"database/sql"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"testing"
)

var testDB *sql.DB

func connectToDB(t *testing.T) *sql.DB {
	if testDB != nil {
		return testDB
	}
	url, err := ParseDatabaseEnv(true)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatalf("Couldn't connect to the database: %v", err)
	}
	err = db.Ping()
	if err != nil {
		t.Fatalf("Couldn't connect to the database: %v", err)
	}
	testDB = db
	return db
}

func TestService_GetAccount(t *testing.T) {
	db := connectToDB(t)
	service := NewService(db)
	account, err := service.GetAccount("test1")
	assert.NoError(t, err)
	assert.NotNil(t, account)
}
