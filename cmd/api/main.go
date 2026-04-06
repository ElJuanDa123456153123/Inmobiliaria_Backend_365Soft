package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// Crear app de Fiber
	app := fiber.New(fiber.Config{
		AppName: "Inmobiliaria API v1.0",
	})

	// Middleware
	app.Use(logger.New())
	app.Use(recover.New())

	// Rutas de salud
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "API is running",
		})
	})

	// Rutas API
	app.Get("/api", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Bienvenido a la API de Inmobiliaria Bolivia",
		})
	})

	// Iniciar servidor
	log.Println("🚀 Servidor corriendo en http://localhost:3001")
	log.Fatal(app.Listen(":3001"))
}
