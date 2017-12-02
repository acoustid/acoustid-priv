package priv

import (
	"encoding/json"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func makeTestRequest(t *testing.T) *http.Request {
	r, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)
	return r
}

func TestParseBasicAuth(t *testing.T) {
	testCases := []struct {
		Auth     string
		Username string
		Password string
	}{
		{"", "", ""},
		{"Bearer xxx", "", ""},
		{"Basic", "", ""},
		{"Basic xxx", "", ""},
		{"Basic Zm9vOmJhcg==", "foo", "bar"},
	}
	for _, testCase := range testCases {
		r := makeTestRequest(t)
		r.Header.Set("Authorization", testCase.Auth)
		username, password := ParseBasicAuth(r)
		assert.Equal(t, testCase.Username, username, "Incorrect username for %q", testCase.Auth)
		assert.Equal(t, testCase.Password, password, "Incorrect password for %q", testCase.Auth)
	}
}

func TestNoAuth_Authenticate(t *testing.T) {
	r := makeTestRequest(t)
	auth := &NoAuth{}
	account, err := auth.Authenticate(r)
	assert.NoError(t, err)
	assert.Equal(t, "default", account)
}

func TestPasswordAuth_Authenticate(t *testing.T) {
	r := makeTestRequest(t)
	r.Header.Set("Authorization", "Basic Zm9vOmJhcg==")
	auth := &PasswordAuth{Username: "foo", Password: "bar"}
	account, err := auth.Authenticate(r)
	assert.NoError(t, err)
	assert.Equal(t, "default", account)
}

func TestPasswordAuth_Authenticate_IncorrectUsername(t *testing.T) {
	r := makeTestRequest(t)
	r.Header.Set("Authorization", "Basic Zm9vOmJhcg==")
	auth := &PasswordAuth{Username: "foo2", Password: "bar"}
	account, err := auth.Authenticate(r)
	assert.Equal(t, ErrNotAuthorized, err)
	assert.Equal(t, "", account)
}

func TestPasswordAuth_Authenticate_IncorrectPassword(t *testing.T) {
	r := makeTestRequest(t)
	r.Header.Set("Authorization", "Basic Zm9vOmJhcg==")
	auth := &PasswordAuth{Username: "foo", Password: "bar2"}
	account, err := auth.Authenticate(r)
	assert.Equal(t, ErrNotAuthorized, err)
	assert.Equal(t, "", account)
}

func createFakeAcoustidBizServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var doc map[string]interface{}
		params := r.URL.Query()
		if params.Get("api_key") == "valid_api_key" && params.Get("tag") == "private" {
			doc = map[string]interface{}{
				"valid":      true,
				"account_id": 123,
			}
		} else {
			doc = map[string]interface{}{
				"valid": false,
			}
		}
		data, _ := json.Marshal(doc)
		w.Write(data)
	}))
}

func TestAcoustidBizAuth_Authenticate(t *testing.T) {
	server := createFakeAcoustidBizServer()
	defer server.Close()

	auth := NewAcoustidBizAuth()
	auth.Cache = cache.New(time.Minute, time.Minute)
	auth.Endpoint = server.URL

	{
		r := makeTestRequest(t)
		r.Header.Set("Authorization", "Basic eC1hY291c3RpZC1hcGkta2V5OnZhbGlkX2FwaV9rZXk=")
		account, err := auth.Authenticate(r)
		assert.NoError(t, err)
		assert.Equal(t, "acoustid-biz:123", account)
	}

	{
		r := makeTestRequest(t)
		r.Header.Set("Authorization", "Basic eC1hY291c3RpZC1hcGkta2V5OnZhbGlkX2FwaV9rZXk=")
		account, err := auth.Authenticate(r)
		assert.NoError(t, err)
		assert.Equal(t, "acoustid-biz:123", account)
	}
}

func TestAcoustidBizAuth_Authenticate_Invalid(t *testing.T) {
	server := createFakeAcoustidBizServer()
	defer server.Close()

	auth := NewAcoustidBizAuth()
	auth.Cache = cache.New(time.Minute, time.Minute)
	auth.Endpoint = server.URL

	{
		r := makeTestRequest(t)
		r.Header.Set("Authorization", "Basic eC1hY291c3RpZC1hcGkta2V5OmludmFsaWRfYXBpX2tleQ==")
		account, err := auth.Authenticate(r)
		assert.Equal(t, ErrNotAuthorized, err)
		assert.Equal(t, "", account)
	}

	{
		r := makeTestRequest(t)
		r.Header.Set("Authorization", "Basic eC1hY291c3RpZC1hcGkta2V5OmludmFsaWRfYXBpX2tleQ==")
		account, err := auth.Authenticate(r)
		assert.Equal(t, ErrNotAuthorized, err)
		assert.Equal(t, "", account)
	}
}
