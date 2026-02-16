package services

import (
	"fmt"
	"moonshine/internal/config"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	"moonshine/internal/repository"
)

var testDB *repository.Database

func TestMain(m *testing.M) {
	logPath := "/tmp/testmain-services.log"
	logFile, _ := os.Create(logPath)
	defer logFile.Close()

	log := func(msg string, args ...interface{}) {
		fmt.Fprintf(logFile, msg+"\n", args...)
		logFile.Sync()
	}

	wd, _ := os.Getwd()
	log("=== TestMain services starting ===")
	log("Working directory: %s", wd)

	envPath := "../../../.env.test"
	log("Trying to load: %s", envPath)

	err := godotenv.Load(envPath)
	if err != nil {
		log("ERROR loading .env.test: %v", err)
	} else {
		log(".env.test loaded OK")
	}

	log("DATABASE_HOST=%s", os.Getenv("DATABASE_HOST"))
	log("DATABASE_PORT=%s", os.Getenv("DATABASE_PORT"))
	log("DATABASE_NAME=%s", os.Getenv("DATABASE_NAME"))

	cfg := config.Load()
	log("Config: DB=%s:%s/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)

	db, err := repository.New(cfg)
	if err != nil {
		log("FATAL: Cannot connect to test database: %v", err)
		log("=== TestMain services failed ===")
		testDB = nil
	} else {
		log("Test database connected OK")
		if err = goose.SetDialect("postgres"); err != nil {
			log("FATAL: Cannot set migration dialect: %v", err)
			testDB = nil
		} else if err = goose.Up(db.DB().DB, "../../../migrations"); err != nil {
			log("FATAL: Cannot apply migrations: %v", err)
			testDB = nil
		} else if err = ensureTestSchema(db.DB()); err != nil {
			log("FATAL: Cannot ensure test schema: %v", err)
			testDB = nil
		} else {
			testDB = db
			log("=== TestMain services ready ===")
		}
	}

	code := m.Run()

	if testDB != nil {
		testDB.Close()
	}
	log("TestMain exiting with code %d", code)
	logFile.Close()
	os.Exit(code) //nolint:gocritic
}

func ensureTestSchema(db *sqlx.DB) error {
	statements := []string{
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP`,
		`ALTER TABLE fights ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP`,
		`ALTER TABLE fights ADD COLUMN IF NOT EXISTS exp INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE equipment_categories ADD COLUMN IF NOT EXISTS created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE equipment_categories ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP`,
		`ALTER TABLE equipment_items ADD COLUMN IF NOT EXISTS created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE equipment_items ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP`,
		`ALTER TABLE equipment_items ADD COLUMN IF NOT EXISTS artifact BOOLEAN NOT NULL DEFAULT false`,
		`ALTER TABLE equipment_items ADD COLUMN IF NOT EXISTS image VARCHAR(255)`,
		`DO $$ BEGIN
			IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'body_part') THEN
				IF NOT EXISTS (
					SELECT 1
					FROM pg_enum
					WHERE enumlabel = 'NECK'
					  AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'body_part')
				) THEN
					ALTER TYPE body_part ADD VALUE 'NECK';
				END IF;
			END IF;
		END $$`,
		`CREATE TABLE IF NOT EXISTS inventory (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at TIMESTAMP,
			user_id UUID NOT NULL,
			equipment_item_id UUID NOT NULL,
			CONSTRAINT fk_inventory_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			CONSTRAINT fk_inventory_item FOREIGN KEY (equipment_item_id) REFERENCES equipment_items(id) ON DELETE CASCADE
		)`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
