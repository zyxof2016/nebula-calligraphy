package main

import (
	"log"
	"os"

	"github.com/nebula-platform/nebula/services/calligraphy/internal/service"
)

func main() {
	databaseURL := os.Getenv("CALLIGRAPHY_DATABASE_URL")
	db, err := service.OpenPostgres(databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := service.MigratePostgres(db); err != nil {
		log.Fatal(err)
	}
	log.Println("calligraphy postgres migration completed")
}
