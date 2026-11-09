package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"golingko/internal/config"
	"golingko/pkg/database"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := strings.ToLower(os.Args[1])

	if command == "create" {
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: migration name required. Usage: migrate create <name>")
			os.Exit(1)
		}
		name := os.Args[2]
		if err := handleCreate(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating migration: %v\n", err)
			os.Exit(1)
		}
		return
	}

	cfg := config.Load()
	db, err := database.InitDB(cfg.DB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Database connection failed: %v\nHint: Ensure PostgreSQL is running via 'docker compose up -d' or 'make up'\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = db.Close()
	}()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize postgres migration driver: %v\n", err)
		os.Exit(1)
	}

	migrationsDir := findMigrationsDir()
	sourceURL := "file://" + filepath.ToSlash(migrationsDir)

	m, err := migrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize migration runner: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_, _ = m.Close()
	}()

	switch command {
	case "up":
		err = m.Up()
		if err != nil && !errors.Is(err, migrate.ErrNoChange) {
			fmt.Fprintf(os.Stderr, "Migration up failed: %v\n", err)
			os.Exit(1)
		}
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("No pending migrations. Database schema is up to date.")
		} else {
			fmt.Println("Applied pending database migrations successfully.")
		}

	case "down":
		steps := 1
		if len(os.Args) >= 3 {
			if s, parseErr := strconv.Atoi(os.Args[2]); parseErr == nil && s > 0 {
				steps = s
			}
		}
		err = m.Steps(-steps)
		if err != nil && !errors.Is(err, migrate.ErrNoChange) {
			fmt.Fprintf(os.Stderr, "Migration down failed: %v\n", err)
			os.Exit(1)
		}
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("No migrations to rollback.")
		} else {
			fmt.Printf("Rolled back %d migration step(s) successfully.\n", steps)
		}

	case "version", "status":
		version, dirty, vErr := m.Version()
		if vErr != nil {
			if errors.Is(vErr, migrate.ErrNilVersion) {
				fmt.Println("Current schema version: 0 (no migrations applied)")
				return
			}
			fmt.Fprintf(os.Stderr, "Failed to read migration version: %v\n", vErr)
			os.Exit(1)
		}
		status := "clean"
		if dirty {
			status = "DIRTY (requires manual intervention or 'migrate force <version>')"
		}
		fmt.Printf("Current schema version: %d (status: %s)\n", version, status)

	case "force":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: version required. Usage: migrate force <version>")
			os.Exit(1)
		}
		v, parseErr := strconv.Atoi(os.Args[2])
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "Invalid version: %s\n", os.Args[2])
			os.Exit(1)
		}
		if err := m.Force(v); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to force version %d: %v\n", v, err)
			os.Exit(1)
		}
		fmt.Printf("Forced schema version to %d successfully.\n", v)

	default:
		fmt.Fprintf(os.Stderr, "Unknown migration command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func handleCreate(name string) error {
	migrationsDir := findMigrationsDir()
	cleanName := sanitizeMigrationName(name)
	if cleanName == "" {
		return errors.New("migration name cannot be empty")
	}

	nextSeq := getNextMigrationSequence(migrationsDir)
	upFileName := fmt.Sprintf("%06d_%s.up.sql", nextSeq, cleanName)
	downFileName := fmt.Sprintf("%06d_%s.down.sql", nextSeq, cleanName)

	upPath := filepath.Join(migrationsDir, upFileName)
	downPath := filepath.Join(migrationsDir, downFileName)

	if err := os.WriteFile(upPath, []byte("-- Migration UP\n"), 0o644); err != nil {
		return fmt.Errorf("write up migration: %w", err)
	}
	if err := os.WriteFile(downPath, []byte("-- Migration DOWN\n"), 0o644); err != nil {
		_ = os.Remove(upPath)
		return fmt.Errorf("write down migration: %w", err)
	}

	fmt.Printf("Created migration files in %s:\n  + %s\n  + %s\n", migrationsDir, upFileName, downFileName)
	return nil
}

func sanitizeMigrationName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	re := regexp.MustCompile(`[^a-z0-9_]+`)
	name = re.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	return name
}

func getNextMigrationSequence(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 1
	}

	seqRe := regexp.MustCompile(`^(\d{6})_`)
	maxSeq := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := seqRe.FindStringSubmatch(entry.Name())
		if len(matches) == 2 {
			seq, err := strconv.Atoi(matches[1])
			if err == nil && seq > maxSeq {
				maxSeq = seq
			}
		}
	}
	return maxSeq + 1
}

func findMigrationsDir() string {
	candidates := []string{
		"migrations",
		"../migrations",
		"../../migrations",
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}
	return "migrations"
}

func printUsage() {
	fmt.Println("Golingko Database Migration Runner")
	fmt.Println("\nUsage:")
	fmt.Println("  go run ./cmd/migrate <command> [arguments]")
	fmt.Println("\nCommands:")
	fmt.Println("  up              Apply all pending migrations")
	fmt.Println("  down [N]        Rollback N migrations (default: 1)")
	fmt.Println("  version         Display current migration version and status")
	fmt.Println("  status          Alias for version")
	fmt.Println("  force <version> Set schema version without running migrations (recovers dirty states)")
	fmt.Println("  create <name>   Generate a new sequential up/down migration pair")
}
