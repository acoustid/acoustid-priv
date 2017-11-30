package main

import (
	"context"
	"database/sql"
	"flag"
	"github.com/acoustid/priv"
	_ "github.com/lib/pq"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"io/ioutil"
)

func main() {
	addr := os.Getenv("ACOUSTID_PRIV_BIND")
	if addr == "" {
		addr = ":5000"
	}

	databaseURL := os.Getenv("ACOUSTID_PRIV_DB_URL")
	if databaseURL == "" {
		var u url.URL
		u.Scheme = "postgresql"
		host := os.Getenv("ACOUSTID_PRIV_DB_HOST")
		port := os.Getenv("ACOUSTID_PRIV_DB_PORT")
		if host != "" {
			if port != "" {
				u.Host = net.JoinHostPort(host, port)
			} else {
				u.Host = host
			}
		} else {
			u.Host = "localhost"
		}
		user := os.Getenv("ACOUSTID_PRIV_DB_USER")
		if user == "" {
			user = "acoustid"
		}
		password := os.Getenv("ACOUSTID_PRIV_DB_PASSWORD")
		if password != "" {
			u.User = url.UserPassword(user, password)
		} else {
			passwordFile := os.Getenv("ACOUSTID_PRIV_DB_PASSWORD_FILE")
			if passwordFile != "" {
				passwordData, err := ioutil.ReadFile(passwordFile)
				if err != nil {
					log.Fatalf("Unable to read password from %s: %v", passwordFile, err)
				}
				password = string(passwordData)
				u.User = url.UserPassword(user, password)
			} else {
				u.User = url.User(user)
			}
		}
		u.Path = os.Getenv("ACOUSTID_PRIV_DB_NAME")
		if u.Path == "" {
			u.Path = "acoustid_priv"
		}
		sslMode := os.Getenv("ACOUSTID_PRIV_DB_SSL")
		if sslMode == "" {
			sslMode = "disable"
		}
		v := url.Values{}
		v.Set("sslmode", sslMode)
		u.RawQuery = v.Encode()
		databaseURL = u.String()
	}

	flag.StringVar(&addr, "bind", addr, "Address on which the server should listen")
	flag.StringVar(&databaseURL, "db", databaseURL, "PostgreSQL URL")
	flag.Parse()

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to the database: %v", err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatalf("Unable to connect to the database: %v", err)
	}

	service := priv.NewService(db)
	handler := priv.NewAPI(service)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	httpServer := &http.Server{Addr: addr, Handler: handler}
	go func() {
		log.Printf("Listening on %v", httpServer.Addr)
		httpServer.ListenAndServe()
	}()

	<-quit
	log.Print("Stopping...")

	httpServer.Shutdown(context.Background())
	log.Print("Done")
}
