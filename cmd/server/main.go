package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/saeid-a/CoachAppBack/internal/config"
	"github.com/saeid-a/CoachAppBack/internal/database"
	"github.com/saeid-a/CoachAppBack/internal/routes"
)

func main() {
	// 1. Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Connect to Database
	if cfg.DBUrl == "" {
		log.Fatal("DB_URL is required")
	}
	if err := database.ConnectDB(cfg.DBUrl); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.CloseDB()

	// 3. Setup Fiber
	app := fiber.New()

	// Middleware
	app.Use(cors.New())
	app.Use(logger.New())
	app.Use(recover.New())

	// Routes
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
		})
	})
	if err := routes.RegisterRoutes(app, cfg, database.DB); err != nil {
		log.Fatalf("Failed to register routes: %v", err)
	}

	// 4. Start Server
	log.Printf("Server starting on port %s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
