package main

import (
	"context"
	"database/sql"
	"flag"
	"github.com/acoustid/priv"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	flag.StringVar(&addr, "bind", addr, "Address on which the server should listen")
	flag.StringVar(&databaseURL, "db", databaseURL, "PostgreSQL URL")
	flag.Parse()

	db, err := sql.Open("postgres", databaseURL)
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
