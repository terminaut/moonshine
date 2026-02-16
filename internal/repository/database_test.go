package repository

import (
	"log"
	"moonshine/internal/config"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
)

var testDB *Database

func TestMain(m *testing.M) {
	log.Println("[TestMain] Starting test setup for repository package")

	err := godotenv.Load("../../.env.test")
	if err != nil {
		log.Printf("[TestMain] Warning: .env.test not loaded: %v", err)
	} else {
		log.Println("[TestMain] .env.test loaded successfully")
	}
	cfg := config.Load()

	log.Println("[TestMain] Attempting to connect to test database...")
	db, err := New(cfg)
	if err != nil {
		log.Fatalf("[TestMain] Failed to initialize test database: %v", err)
	}
	if err = goose.SetDialect("postgres"); err != nil {
		log.Fatalf("[TestMain] Failed to set migration dialect: %v", err)
	}
	if err = goose.Up(db.DB().DB, "../../migrations"); err != nil {
		log.Fatalf("[TestMain] Failed to apply migrations: %v", err)
	}
	testDB = db
	log.Println("[TestMain] Test database connected successfully")

	code := m.Run()

	testDB.Close()
	os.Exit(code)
}
