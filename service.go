package priv

import (
	"database/sql"
	"github.com/pkg/errors"
	"strings"
	"encoding/base64"
)

var ErrNotAuthorized = errors.New("not authorized")
var ErrAccountNotFound = errors.New("account not found")
var ErrAccountDisabled = errors.New("account disabled")

type Service interface {
	GetAccountByApiKey(token string) (Account, error)
}

type ServiceImpl struct {
	db *sql.DB
}

func NewService(db *sql.DB) Service {
	return &ServiceImpl{db: db}
}

func (s *ServiceImpl) GetAccountByApiKey(token string) (Account, error) {
	row := s.db.QueryRow(`SELECT id, enabled FROM account WHERE api_key = $1`, token)
	var id int
	var enabled bool
	err := row.Scan(&id, &enabled)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrAccountNotFound
		} else {
			return nil, errors.WithMessage(err, "failed to read account")
		}
	}

	if !enabled {
		return nil, ErrAccountDisabled
	}

	account := &AccountImpl{db: s.db, id: id}
	return account, nil
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
	if errors.Cause(err) == ErrAccountNotFound {
		return nil, errors.WithMessage(ErrNotAuthorized, "invalid token")
	}

	return account, nil
}