package main

import (
	"context"
	"database/sql"
	"flag"
	"github.com/acoustid/priv"
	_ "github.com/lib/pq"
	"github.com/patrickmn/go-cache"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	addr := os.Getenv("ACOUSTID_PRIV_BIND")
	if addr == "" {
		addr = ":3382"
	}

	databaseURL, err := priv.ParseDatabaseEnv(false)
	if err != nil {
		log.Fatal(err)
	}

	auth := os.Getenv("ACOUSTID_PRIV_AUTH")
	if auth == "" {
		auth = "disabled"
	}

	authUsername := os.Getenv("ACOUSTID_PRIV_AUTH_USER")
	authPassword := os.Getenv("ACOUSTID_PRIV_AUTH_PASSWORD")

	shutdownDelay := time.Millisecond * 100
	shutdownDelayStr := os.Getenv("ACOUSTID_PRIV_SHUTDOWN_DELAY")
	if shutdownDelayStr != "" {
		d, err := time.ParseDuration(shutdownDelayStr)
		if err != nil {
			log.Fatalf("Error while parsing ACOUSTID_PRIV_SHUTDOWN_DELAY: %v", err)
		}
		shutdownDelay = d
	}

	flag.StringVar(&addr, "bind", addr, "Address on which the server should listen")
	flag.StringVar(&databaseURL, "db", databaseURL, "PostgreSQL URL")
	flag.StringVar(&auth, "auth", auth, "Authentication method (disabled, password, acoustid-biz)")
	flag.StringVar(&authUsername, "user", authUsername, "Username for password authentication")
	flag.StringVar(&authPassword, "password", authPassword, "Password for password authentication")
	flag.DurationVar(&shutdownDelay, "shutdown-delay", shutdownDelay, "Delay shutdown")
	flag.Parse()

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to the database: %v", err)
	}

	service := priv.NewService(db)
	handler := priv.NewAPI(service)

	if auth == "password" {
		handler.Auth = &priv.PasswordAuth{authUsername, authPassword}
	} else if auth == "acoustid-biz" {
		authenticator := priv.NewAcoustidBizAuth()
		authenticator.Cache = cache.New(time.Hour, time.Minute*10)
		handler.Auth = authenticator
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	httpServer := &http.Server{Addr: addr, Handler: handler}
	go func() {
		log.Printf("Starting HTTP server on %v", httpServer.Addr)
		httpServer.ListenAndServe()
	}()

	<-quit

	log.Printf("Marking as unhealthy and waiting for %s", shutdownDelay)
	handler.SetHealthStatus(false)
	time.Sleep(shutdownDelay)

	log.Print("Shutting down")
	shutdownContext, _ := context.WithTimeout(context.Background(), time.Second * 10)
	httpServer.Shutdown(shutdownContext)
	httpServer.Close()

	log.Print("Exit")
}
