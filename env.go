package priv

import (
	"github.com/pkg/errors"
	"io/ioutil"
	"net"
	"net/url"
	"os"
)

func ParseDatabaseEnv(test bool) (string, error) {
	prefix := "ACOUSTID_PRIV"
	if test {
		prefix += "_TEST"
	}

	dbURL := os.Getenv(prefix + "_DB_URL")
	if dbURL != "" {
		return dbURL, nil
	}

	u := url.URL{Scheme: "postgresql"}

	host := os.Getenv(prefix + "_DB_HOST")
	port := os.Getenv(prefix + "_DB_PORT")
	if host != "" {
		if port != "" {
			u.Host = net.JoinHostPort(host, port)
		} else {
			u.Host = host
		}
	} else {
		if test {
			u.Host = "localhost:15432"
		} else {
			u.Host = "localhost"
		}
	}

	user := os.Getenv(prefix + "_DB_USER")
	if user == "" {
		userFile := os.Getenv(prefix + "_DB_USER_FILE")
		if userFile != "" {
			userData, err := ioutil.ReadFile(userFile)
			if err != nil {
				return "", errors.WithMessage(err, "Unable to read user file")
			}
			user = string(userData)
		} else {
			user = "acoustid"
		}
	}
	password := os.Getenv(prefix + "_DB_PASSWORD")
	if password != "" {
		u.User = url.UserPassword(user, password)
	} else {
		passwordFile := os.Getenv(prefix + "_DB_PASSWORD_FILE")
		if passwordFile != "" {
			passwordData, err := ioutil.ReadFile(passwordFile)
			if err != nil {
				return "", errors.WithMessage(err, "Unable to read password file")
			}
			password = string(passwordData)
			u.User = url.UserPassword(user, password)
		} else {
			u.User = url.User(user)
		}
	}

	u.Path = os.Getenv(prefix + "_DB_NAME")
	if u.Path == "" {
		if test {
			u.Path = "acoustid_priv_test"
		} else {
			u.Path = "acoustid_priv"
		}
	}

	sslMode := os.Getenv(prefix + "_DB_SSL")
	if sslMode == "" {
		sslMode = "disable"
	}
	v := url.Values{}
	v.Set("sslmode", sslMode)
	u.RawQuery = v.Encode()

	return u.String(), nil
}
