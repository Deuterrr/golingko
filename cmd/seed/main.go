package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golingko/internal/config"
	"golingko/pkg/database"
)

func main() {
	var downFlag bool
	var resetFlag bool
	flag.BoolVar(&downFlag, "down", false, "Rollback seeded transit data")
	flag.BoolVar(&resetFlag, "reset", false, "Reset seeded transit data (down then up)")
	flag.Parse()

	cfg := config.Load()

	db, err := database.InitDB(cfg.DB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Database connection failed: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = db.Close()
	}()

	seedsDir := findSeedsDir()

	if resetFlag {
		if err := executeSQLFile(db, filepath.Join(seedsDir, "001_transit_seeds.down.sql")); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to rollback seeds: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Rolled back seeded transit data.")

		if err := executeSQLFile(db, filepath.Join(seedsDir, "001_transit_seeds.sql")); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to apply seeds: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Applied fresh transit seeds successfully.")
		return
	}

	if downFlag {
		if err := executeSQLFile(db, filepath.Join(seedsDir, "001_transit_seeds.down.sql")); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to rollback seeds: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Rolled back seeded transit data successfully.")
		return
	}

	if err := executeSQLFile(db, filepath.Join(seedsDir, "001_transit_seeds.sql")); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to apply seeds: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Applied transit seeds successfully (57 stations, 30 routes, 9 cards, 5 journeys, 4 ledger records).")
}

func executeSQLFile(db *sql.DB, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read sql file %s: %w", filePath, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = db.ExecContext(ctx, string(data))
	if err != nil {
		return fmt.Errorf("exec sql: %w", err)
	}
	return nil
}

func findSeedsDir() string {
	candidates := []string{
		"seeds",
		"../seeds",
		"../../seeds",
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}
	return "seeds"
}
