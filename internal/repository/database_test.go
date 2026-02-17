package repository

import (
	"log"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"

	"moonshine/internal/testutil"
)

var testDB *sqlx.DB

func TestMain(m *testing.M) {
	db, err := testutil.SetupTestDB("../../.env.test", "../../migrations")
	if err != nil {
		log.Printf("Test database not available: %v", err)
	}
	testDB = db

	code := m.Run()

	if testDB != nil {
		testDB.Close()
	}
	os.Exit(code) //nolint:gocritic
}
