package priv

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrNotAuthorized = errors.New("not authorized")
var ErrAccountNotFound = errors.New("account not found")

type ApiKeyProvider interface {
	CheckApiKey(apiKey string, accountID *int) (bool, error)
}

type Service interface {
	GetAccountByApiKey(token string) (Account, error)
	SetApiKeyProvider(provider ApiKeyProvider)
}

type AcoustidBizApiKeyProvider struct{}

func (p *AcoustidBizApiKeyProvider) CheckApiKey(apiKey string, accountID *int) (bool, error) {
	u, err := url.Parse("https://acoustid.biz/internal/validate-api-key")
	if err != nil {
		return false, err
	}
	u.RawQuery = url.Values{"api_key": {apiKey}, "tag": {"private"}}.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, errors.New("")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	type Doc struct {
		Valid     bool `json:"valid"`
		AccountID int  `json:"account_id"`
	}

	var doc Doc
	err = json.Unmarshal(body, &doc)
	if err != nil {
		return false, err
	}

	if accountID != nil {
		*accountID = doc.AccountID
	}
	return doc.Valid, err
}

type ServiceImpl struct {
	db             *sql.DB
	apiKeyProvider ApiKeyProvider
}

func NewService(db *sql.DB) Service {
	return &ServiceImpl{db: db}
}

func (s *ServiceImpl) SetApiKeyProvider(provider ApiKeyProvider) {
	s.apiKeyProvider = provider
}

func (s *ServiceImpl) GetAccountByApiKey(apiKey string) (Account, error) {
	row := s.db.QueryRow(`SELECT account_id, expires_at FROM api_key WHERE api_key = $1`, apiKey)
	var cacheAccountID *int
	var cacheExpiresAt *time.Time
	err := row.Scan(&cacheAccountID, &cacheExpiresAt)
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, errors.WithMessage(err, "failed to read account")
		}
	}

	now := time.Now().Add(-time.Millisecond * time.Duration(rand.Int()%500))

	if cacheAccountID != nil {
		if cacheExpiresAt == nil || cacheExpiresAt.After(now) {
			return &AccountImpl{db: s.db, id: *cacheAccountID}, nil
		}
	}

	if cacheExpiresAt != nil && cacheExpiresAt.After(now) {
		return nil, ErrAccountNotFound
	}

	if s.apiKeyProvider == nil {
		return nil, ErrAccountNotFound
	}

	var accountID int
	found, err := s.apiKeyProvider.CheckApiKey(apiKey, &accountID)
	if err != nil {
		if cacheAccountID != nil {
			log.Printf("Failed to check API key, using expired cache: %v", err)
			return &AccountImpl{db: s.db, id: *cacheAccountID}, nil
		}
		return nil, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if !found {
		expiresAt := time.Now().Add(time.Minute)
		query := `INSERT INTO api_key (api_key, expires_at, account_id) VALUES ($1, $2, NULL) ON CONFLICT (api_key) DO UPDATE SET expires_at = EXCLUDED.expires_at, account_id = EXCLUDED.account_id`
		_, err = tx.Exec(query, apiKey, expiresAt)
		if err != nil {
			return nil, err
		}
	} else {
		_, err := tx.Exec(`INSERT INTO account (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, accountID)
		if err != nil {
			return nil, err
		}
		expiresAt := time.Now().Add(time.Hour)
		query := `INSERT INTO api_key (api_key, expires_at, account_id) VALUES ($1, $2, $3) ON CONFLICT (api_key) DO UPDATE SET expires_at = EXCLUDED.expires_at, account_id = EXCLUDED.account_id`
		_, err = tx.Exec(query, apiKey, expiresAt, accountID)
		if err != nil {
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	if !found {
		return nil, ErrAccountNotFound
	}

	return &AccountImpl{db: s.db, id: accountID}, nil
}

func Authenticate(service Service, auth string) (Account, error) {
	var token string
	authParts := strings.SplitN(auth, " ", 2)
	if authParts[0] == "Token" && len(authParts) == 2 {
		token = authParts[1]
	} else if authParts[0] == "Basic" && len(authParts) == 2 {
		decodedAuth, err := base64.StdEncoding.DecodeString(authParts[1])
		if err != nil {
			return nil, errors.WithMessage(ErrNotAuthorized, "invalid authentication header")
		}
		decodedAuthParts := strings.SplitN(string(decodedAuth), ":", 2)
		if decodedAuthParts[0] == "x-acoustid-api-key" && len(decodedAuthParts) == 2 {
			token = decodedAuthParts[1]
		} else {
			return nil, errors.WithMessage(ErrNotAuthorized, "invalid authentication header")
		}
	} else {
		return nil, errors.WithMessage(ErrNotAuthorized, "invalid authentication header")
	}

	account, err := service.GetAccountByApiKey(token)
	if err != nil {
		if errors.Cause(err) == ErrAccountNotFound {
			return nil, errors.WithMessage(ErrNotAuthorized, "invalid token")
		}
		return nil, err
	}

	return account, nil
}
