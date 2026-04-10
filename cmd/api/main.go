package main

import (
	"log"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/database"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/handlers"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
)

func main() {
	// Cargar variables de entorno desde archivo .env
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No se encontró archivo .env, usando variables de entorno del sistema")
	}

	// Ejecutar migraciones al iniciar
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("❌ Error ejecutando migraciones: %v", err)
	}

	// Crear app de Fiber
	app := fiber.New(fiber.Config{
		AppName: "Inmobiliaria API v1.0",
	})

	// Middleware
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000,http://localhost:3001,http://localhost:3002",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, DELETE, PATCH, OPTIONS",
		AllowCredentials: true,
	}))

	// Inicializar handler de autenticación
	authHandler, err := handlers.NewAuthHandler()
	if err != nil {
		log.Fatalf("❌ Error inicializando auth handler: %v", err)
	}

	// Inicializar handler de Google Auth
	googleAuthHandler, err := handlers.NewGoogleAuthHandler()
	if err != nil {
		log.Printf("⚠️  Google Auth no disponible: %v", err)
		// No es fatal, continuar sin Google Auth
	}

	// Inicializar handler de administración
	adminHandler, err := handlers.NewAdminHandler()
	if err != nil {
		log.Printf("⚠️  Admin handler no disponible: %v", err)
		// No es fatal, continuar sin admin handler
	}

	// Rutas de salud
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "API is running",
		})
	})

	app.Get("/api", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Bienvenido a la API de Inmobiliaria Bolivia",
			"version": "1.0.0",
		})
	})

	// API Routes
	api := app.Group("/api")

	// Rutas públicas de autenticación
	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)

	// Ruta de login para administradores (harcodeada)
	auth.Post("/admin/login", authHandler.AdminLogin)

	// Rutas de Google OAuth (PÚBLICAS - sin autenticación)
	if googleAuthHandler != nil {
		app.Get("/api/auth/google", googleAuthHandler.GoogleLogin)
		app.Get("/api/auth/google/callback", googleAuthHandler.GoogleCallback)
	}

	// Rutas protegidas (requieren autenticación)
	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware())
	protected.Get("/auth/me", authHandler.GetMe)

	// Ruta de completar perfil de Google (protegida - requiere token)
	if googleAuthHandler != nil {
		protected.Post("/auth/google/complete-profile", googleAuthHandler.CompleteGoogleProfile)
	}

	// Rutas de administración (protegidas - requieren rol de admin)
	if adminHandler != nil {
		admin := api.Group("/admin")
		admin.Use(middleware.AuthMiddleware())      // Primero verificar autenticación
		admin.Use(middleware.AdminMiddleware())      // Luego verificar rol de admin
		admin.Get("/users", adminHandler.GetUsers)
		admin.Patch("/users/:id/toggle-active", adminHandler.ToggleUserActive)
		admin.Delete("/users/:id", adminHandler.DeleteUser)
	}

	// Iniciar servidor
	log.Println("🚀 Servidor corriendo en http://localhost:3001")
	log.Println("📊 Endpoints disponibles:")
	log.Println("  POST   /api/auth/register  - Registro de usuario")
	log.Println("  POST   /api/auth/login     - Login de usuario")
	log.Println("  GET    /api/auth/me        - Obtener usuario actual (requiere token)")
	log.Fatal(app.Listen(":3001"))
}
