package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/lib/pq"
	"github.com/pressly/goose/v3"

	"moonshine/internal/config"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(".env not loaded, relying on environment")
	}

	command := flag.String("command", "up", "Migration command: up, down, down-to, status, create")
	name := flag.String("name", "", "Migration name (required for create)")
	targetVersion := flag.Int64("version", 0, "Target version for down-to command")
	flag.Parse()

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

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}

	if err := db.Ping(); err != nil {
		if *command == "up" && isDatabaseDoesNotExistError(err) {
			if err := createDatabase(cfg); err != nil {
				log.Fatalf("Failed to create database: %v", err)
			}
			db, err = sql.Open("postgres", dsn)
			if err != nil {
				log.Fatalf("Failed to open database connection: %v", err)
			}
			if err := db.Ping(); err != nil {
				log.Fatalf("Failed to connect to database: %v", err)
			}
		} else {
			log.Fatalf("Failed to connect to database: %v", err)
		}
	}
	if err := goose.SetDialect("postgres"); err != nil {
		db.Close()
		log.Fatalf("Failed to set dialect: %v", err)
	}

	migrationsDir := "migrations"

	switch *command {
	case "up":
		if err := goose.Up(db, migrationsDir); err != nil {
			db.Close()
			log.Fatalf("Failed to run migrations: %v", err)
		}
		log.Println("Migrations applied successfully")
	case "down":
		if err := goose.Down(db, migrationsDir); err != nil {
			db.Close()
			log.Fatalf("Failed to rollback migrations: %v", err)
		}
		log.Println("Migrations rolled back successfully")
	case "down-to":
		if err := goose.DownTo(db, migrationsDir, *targetVersion); err != nil {
			db.Close()
			log.Fatalf("Failed to rollback migrations to version %d: %v", *targetVersion, err)
		}
		log.Printf("Migrations rolled back to version %d successfully", *targetVersion)
	case "status":
		if err := goose.Status(db, migrationsDir); err != nil {
			db.Close()
			log.Fatalf("Failed to get migration status: %v", err)
		}
	case "create":
		if *name == "" {
			db.Close()
			log.Fatal("Migration name is required for create command")
		}
		if err := goose.Create(db, migrationsDir, *name, "sql"); err != nil {
			db.Close()
			log.Fatalf("Failed to create migration: %v", err)
		}
		log.Printf("Created migration: %s", *name)
	default:
		db.Close()
		log.Fatalf("Unknown command: %s", *command)
	}
	db.Close()
}

func isDatabaseDoesNotExistError(err error) bool {
	if err == nil {
		return false
	}
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return false
	}
	return pqErr.Code == "3D000"
}

func createDatabase(cfg *config.Config) error {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to connect to postgres database: %w", err)
	}
	defer db.Close()

	query := fmt.Sprintf("CREATE DATABASE %s", cfg.Database.Name)
	_, err = db.Exec(query)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "42P04" {
			return nil
		}
		return fmt.Errorf("failed to create database: %w", err)
	}

	log.Printf("Database '%s' created successfully", cfg.Database.Name)
	return nil
}
