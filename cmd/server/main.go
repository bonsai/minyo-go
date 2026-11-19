package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/bonsai/minyo-go/internal/db"
	"github.com/bonsai/minyo-go/internal/httpapi"
	"github.com/bonsai/minyo-go/internal/seed"
)

func main() {
	databasePath := flag.String("db", envOr("MINYO_GO_DB", "minyo-go.db"), "SQLite database path")
	importPath := flag.String("import-spots", "", "import a verified spots JSON file and exit")
	address := flag.String("addr", envOr("MINYO_GO_ADDR", ":8080"), "HTTP listen address")
	flag.Parse()

	database, err := db.Open(*databasePath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer database.Close()

	if *importPath != "" {
		count, err := seed.ImportSpots(database, *importPath)
		if err != nil {
			log.Fatalf("import spots: %v", err)
		}
		log.Printf("imported %d verified spots", count)
		return
	}

	server := &http.Server{
		Addr: *address,
		Handler: httpapi.New(database).Handler(),
	}
	log.Printf("MINYO-GO API listening on %s", *address)
	log.Fatal(server.ListenAndServe())
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
