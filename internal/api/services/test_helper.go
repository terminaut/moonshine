package services

import (
	"log"
	"moonshine/internal/config"
	"os"
	"testing"

	"github.com/joho/godotenv"

	"moonshine/internal/repository"
)

var testDB *repository.Database

func TestMain(m *testing.M) {
	cfg := config.Load()

	log.Println("[TestMain services] Starting test setup")

	err := godotenv.Load("../../../.env.test")
	if err != nil {
		log.Printf("[TestMain services] Warning: .env.test not loaded: %v", err)
	} else {
		log.Println("[TestMain services] .env.test loaded successfully")
	}

	log.Println("[TestMain services] Attempting to connect to test database...")
	db, err := repository.New(cfg)
	if err != nil {
		log.Printf("[TestMain services] Failed to connect to database: %v", err)
		testDB = nil
		code := m.Run()
		os.Exit(code)
	}
	testDB = db
	log.Println("[TestMain services] Test database connected successfully")

	code := m.Run()

	if testDB != nil {
		testDB.Close()
	}
	os.Exit(code)
}
