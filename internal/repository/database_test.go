package repository

import (
	"log"
	"moonshine/internal/config"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

var testDB *Database

func TestMain(m *testing.M) {
	log.Println("[TestMain] Starting test setup for repository package")

	cfg := config.Load()
	err := godotenv.Load("../../.env.test")
	if err != nil {
		log.Printf("[TestMain] Warning: .env.test not loaded: %v", err)
	} else {
		log.Println("[TestMain] .env.test loaded successfully")
	}

	log.Println("[TestMain] Attempting to connect to test database...")
	db, err := New(cfg)
	if err != nil {
		log.Fatalf("[TestMain] Failed to initialize test database: %v", err)
	}
	testDB = db
	log.Println("[TestMain] Test database connected successfully")

	code := m.Run()

	testDB.Close()
	os.Exit(code)
}
