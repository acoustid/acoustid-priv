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
	db, err := sql.Open("postgres", "postgresql://acoustid:acoustid@localhost:15432/acoustid_test?sslmode=disable")
	if err != nil {
		t.Skip("Couldn't connect to the database: %v", err)
	}
	testDB = db
	return db
}

func TestService_GetAccountByApiKey(t *testing.T) {
	db := connectToDB(t)
	service := NewService(db)
	account, err := service.GetAccountByApiKey("api_key")
	assert.NoError(t, err)
	assert.NotNil(t, account)
}

func TestService_GetAccountByApiKey_Disabled(t *testing.T) {
	db := connectToDB(t)
	service := NewService(db)
	account, err := service.GetAccountByApiKey("disabled_api_key")
	if assert.Error(t, err) {
		assert.Equal(t, ErrAccountDisabled, err)
	}
	assert.Nil(t, account)
}

func TestService_GetAccountByApiKey_NotFound(t *testing.T) {
	db := connectToDB(t)
	service := NewService(db)
	account, err := service.GetAccountByApiKey("api_key_that_does_not_exist")
	if assert.Error(t, err) {
		assert.Equal(t, ErrAccountNotFound, err)
	}
	assert.Nil(t, account)
}

func TestAuthenticate_Token(t *testing.T) {
	db := connectToDB(t)
	service := NewService(db)
	account, err := Authenticate(service, "Token api_key")
	assert.NoError(t, err)
	assert.NotNil(t, account)
}

func TestAuthenticate_Basic(t *testing.T) {
	db := connectToDB(t)
	service := NewService(db)
	account, err := Authenticate(service, "Basic eC1hY291c3RpZC1hcGkta2V5OmFwaV9rZXk=")
	assert.NoError(t, err)
	assert.NotNil(t, account)
}

func TestAuthenticate_InvalidHeader1(t *testing.T) {
	db := connectToDB(t)
	service := NewService(db)
	account, err := Authenticate(service, "Token")
	assert.Error(t, err)
	assert.Nil(t, account)
}

func TestAuthenticate_InvalidHeader2(t *testing.T) {
	db := connectToDB(t)
	service := NewService(db)
	account, err := Authenticate(service, "Basic")
	assert.Error(t, err)
	assert.Nil(t, account)
}

func TestAuthenticate_EmptyHeader(t *testing.T) {
	db := connectToDB(t)
	service := NewService(db)
	account, err := Authenticate(service, "")
	assert.Error(t, err)
	assert.Nil(t, account)
}
