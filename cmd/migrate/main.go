package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	dbUrl := os.Getenv("DB_URL")
	if dbUrl == "" {
		log.Fatal("DB_URL environment variable is required")
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	candidates := []string{}
	current := cwd
	for i := 0; i < 6; i++ {
		candidates = append(candidates, filepath.Join(current, "migrations"))
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, "migrations"),
			filepath.Join(exeDir, "..", "migrations"),
			filepath.Join(exeDir, "..", "..", "migrations"),
		)
	}
	var migrationsPath string
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			migrationsPath = candidate
			break
		}
	}
	if migrationsPath == "" {
		log.Fatal("Migrations directory not found")
	}
	absMigrationsPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		log.Fatal(err)
	}

	m, err := migrate.New(
		"file://"+absMigrationsPath,
		dbUrl,
	)
	if err != nil {
		log.Fatal(err)
	}

	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	if cmd == "down" {
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
		log.Println("Migration down successful")
	} else {
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
		log.Println("Migration up successful")
	}
}
