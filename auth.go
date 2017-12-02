package priv

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrNotAuthorized = errors.New("not authorized")

type Authenticator interface {
	Authenticate(r *http.Request) (account string, err error)
}

type NoAuth struct {
}

func (a *NoAuth) Authenticate(r *http.Request) (account string, err error) {
	return "default", nil
}

type PasswordAuth struct {
	Username string
	Password string
}

func (a *PasswordAuth) Authenticate(r *http.Request) (account string, err error) {
	username, password := ParseBasicAuth(r)
	if username == a.Username && password == a.Password {
		return "default", nil
	}
	return "", ErrNotAuthorized
}

type AcoustidBizAuth struct {
	Cache    Cache
	Endpoint string
	Username string
}

func NewAcoustidBizAuth() *AcoustidBizAuth {
	auth := &AcoustidBizAuth{}
	auth.Endpoint = "https://acoustid.biz/internal/validate-api-key"
	auth.Username = "x-acoustid-api-key"
	return auth
}

func (a *AcoustidBizAuth) Authenticate(r *http.Request) (account string, err error) {
	username, password := ParseBasicAuth(r)
	if strings.ToLower(username) == a.Username && password != "" {
		return a.check(password)
	}
	return "", ErrNotAuthorized
}

func (a *AcoustidBizAuth) check(apiKey string) (account string, err error) {
	cacheKey := fmt.Sprintf("acoustid-biz-api-key:%s", apiKey)
	if a.Cache != nil {
		result, found := a.Cache.Get(cacheKey)
		if found {
			account = result.(string)
			if account == "" {
				return "", ErrNotAuthorized
			}
			return account, nil
		}
	}

	account, err = a.validateApiKey(apiKey)
	if err != nil {
		return "", errors.WithMessage(err, "failed to check remote API key")
	}

	if a.Cache != nil {
		var expiration time.Duration
		if account == "" {
			expiration = time.Minute
		} else {
			expiration = time.Hour
		}
		a.Cache.Set(cacheKey, account, expiration)
	}

	if account == "" {
		return "", ErrNotAuthorized
	}
	return account, nil
}

func (a *AcoustidBizAuth) validateApiKey(apiKey string) (account string, err error) {
	params := url.Values{"api_key": {apiKey}, "tag": {"private"}}
	resp, err := http.Get(a.Endpoint + "?" + params.Encode())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error %v", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var doc struct {
		Valid     bool `json:"valid"`
		AccountID int  `json:"account_id"`
	}
	err = json.Unmarshal(body, &doc)
	if err != nil {
		return "", err
	}
	if !doc.Valid {
		return "", nil
	}
	return fmt.Sprintf("acoustid-biz:%v", doc.AccountID), nil
}

func ParseBasicAuth(r *http.Request) (username string, password string) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", ""
	}
	headerParts := strings.SplitN(header, " ", 2)
	if strings.ToLower(headerParts[0]) != "basic" || len(headerParts) != 2 {
		return "", ""
	}
	auth, err := base64.StdEncoding.DecodeString(headerParts[1])
	if err != nil {
		return "", ""
	}
	decodedAuthParts := strings.SplitN(string(auth), ":", 2)
	if len(decodedAuthParts) != 2 {
		return "", ""
	}
	return decodedAuthParts[0], decodedAuthParts[1]
}
