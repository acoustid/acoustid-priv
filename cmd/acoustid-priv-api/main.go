package main

import (
	"database/sql"
	"flag"
	"github.com/acoustid/priv"
	_ "github.com/lib/pq"
	"log"
	"net/http"
)

func main() {
	bindFlag := flag.String("bind", "0.0.0.0:5000", "PostgreSQL URL")
	dbFlag := flag.String("db", "postgresql://localhost/acoustid_priv", "PostgreSQL URL")
	flag.Parse()

	db, err := sql.Open("postgres", *dbFlag)
	if err != nil {
		log.Fatalf("Unable to connect to the database: %v", err)
	}

	service := priv.NewService(db)
	api := priv.NewAPI(service)

	srv := &http.Server{
		Handler: api,
		Addr:    *bindFlag,
	}
	log.Fatal(srv.ListenAndServe())
}
