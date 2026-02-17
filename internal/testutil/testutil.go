package testutil

import (
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	"moonshine/internal/config"
)

func SetupTestDB(envRelPath, migrationsRelPath string) (*sqlx.DB, error) {
	_ = godotenv.Load(envRelPath)
	cfg := config.Load()

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("connect to test db: %w", err)
	}

	if err = goose.SetDialect("postgres"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set dialect: %w", err)
	}

	if err = goose.Up(db.DB, migrationsRelPath); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	if err = ensureTestSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	return db, nil
}

func RequireDB(t *testing.T, db *sqlx.DB) {
	t.Helper()
	if db == nil {
		t.Skip("Test database not initialized")
	}
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
