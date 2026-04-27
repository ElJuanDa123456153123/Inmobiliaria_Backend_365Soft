package main

import (
	"log"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/database"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/handlers"
	adminhandlers "github.com/365Soft-Bolivia/inmobiliaria-backend/internal/handlers/admin"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
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

	// 🔒 Middleware profesional
	app.Use(middleware.RecoverMiddleware())       // Recupera de panics
	app.Use(middleware.RequestIDMiddleware())     // Añade ID único
	app.Use(middleware.LoggingMiddleware())       // Logs estructurados
	app.Use(middleware.SecurityHeadersMiddleware()) // Headers de seguridad
	app.Use(middleware.ErrorHandlerMiddleware())  // Manejo centralizado de errores

	// 🚀 Rate Limiting con Redis (si está disponible)
	var redisClient *config.RedisClient
	propertyHandler, err := handlers.NewPropertyHandler()
	if err == nil && propertyHandler.Redis != nil {
		redisClient = propertyHandler.Redis
		rateLimiter := middleware.NewRateLimiter(redisClient)

		// Rate limiting estándar para todas las rutas
		app.Use(rateLimiter.RateLimitMiddleware(middleware.RateLimitConfig{
			RequestsPerMinute: 60,  // 60 requests por minuto por IP
			RequestsPerHour:   1000, // 1000 por hora
			BanDuration:       5 * time.Minute,
		}))

		// Rate limiting específico por endpoint
		endpointLimits := middleware.GetStandardRateLimits()
		app.Use(rateLimiter.RateLimitByEndpointsMiddleware(endpointLimits))
	} else {
		log.Println("⚠️  Rate limiting no disponible (Redis no conectado)")
	}

	// CORS (personalizado)
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000,http://localhost:3001,http://localhost:3002",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Request-ID",
		AllowMethods:     "GET, POST, PUT, DELETE, PATCH, OPTIONS",
		AllowCredentials: true,
		ExposeHeaders:    "Content-Length, Content-Type",
		MaxAge:           86400, // 24 horas
		// Permitir preflight requests
		Next:             nil,
	}))

	// Manejar explícitamente preflight OPTIONS requests
	app.Options("/*", func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", c.Get("Origin"))
		c.Set("Access-Control-Allow-Credentials", "true")
		c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
		c.Set("Access-Control-Max-Age", "86400")
		return c.SendStatus(fiber.StatusNoContent)
	})

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

	// Inicializar handlers de administración
	userHandler, err := adminhandlers.NewUserHandler()
	if err != nil {
		log.Printf("⚠️  User admin handler no disponible: %v", err)
	}

	advisorHandler, err := adminhandlers.NewAdvisorHandler()
	if err != nil {
		log.Printf("⚠️  Advisor admin handler no disponible: %v", err)
	}

	adminPropertyHandler, err := adminhandlers.NewPropertyHandler()
	if err != nil {
		log.Printf("⚠️  Property admin handler no disponible: %v", err)
	}

	// Inicializar handler de planes
	planHandler, err := handlers.NewPlanHandler()
	if err != nil {
		log.Printf("⚠️  Plan handler no disponible: %v", err)
	}

	// NOTA: propertyHandler ya fue inicializado arriba para rate limiting

	// Inicializar handler de imágenes
	imageHandler, err := handlers.NewImageHandler()
	if err != nil {
		log.Printf("⚠️  Image handler no disponible: %v", err)
		// No es fatal, continuar sin image handler
	}

	// Inicializar handler de WebSocket (para actualizaciones en tiempo real)
	var wsHandler *handlers.WebSocketHandler
	if propertyHandler != nil && propertyHandler.Redis != nil {
		wsHandler = handlers.NewWebSocketHandler(propertyHandler.Redis)
		log.Println("✅ WebSocket handler inicializado para actualizaciones en tiempo real")
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

	// Ruta de login estricta para administradores
	auth.Post("/admin/login", authHandler.AdminLogin)

	// Rutas de Google OAuth (PÚBLICAS - sin autenticación)
	if googleAuthHandler != nil {
		app.Get("/api/auth/google", googleAuthHandler.GoogleLogin)
		app.Get("/api/auth/google/callback", googleAuthHandler.GoogleCallback)
	}

	// Rutas protegidas de autenticación (aplicamos middleware individualmente)
	api.Get("/auth/me", middleware.AuthMiddleware(), authHandler.GetMe)
	api.Put("/auth/profile", middleware.AuthMiddleware(), authHandler.UpdateProfile) // Actualizar perfil
	api.Put("/auth/datos-publicos", middleware.AuthMiddleware(), authHandler.UpdateDatosPublicos) // Actualizar datos públicos

	// Ruta de completar perfil de Google (protegida - requiere token)
	if googleAuthHandler != nil {
		api.Post("/auth/google/complete-profile", middleware.AuthMiddleware(), googleAuthHandler.CompleteGoogleProfile)
	}

	// Rutas de administración (protegidas - requieren rol de admin)
	admin := api.Group("/admin")
	admin.Use(middleware.AuthMiddleware())      // Primero verificar autenticación
	admin.Use(middleware.AdminMiddleware())      // Luego verificar rol de admin

	// Rutas de gestión de usuarios
	if userHandler != nil {
		admin.Get("/users", userHandler.GetUsers)
		admin.Post("/users", userHandler.CreateUser)
		admin.Put("/users/:id", userHandler.UpdateUser)
		admin.Patch("/users/:id/toggle-active", userHandler.ToggleUserActive)
		admin.Delete("/users/:id", userHandler.DeleteUser)
	}

	// Gestión de Planes (Planes de suscripción)
	if planHandler != nil {
		admin.Get("/planes", planHandler.GetPlans)
		admin.Post("/planes", planHandler.CreatePlan)
		admin.Put("/planes/:id", planHandler.UpdatePlan)
		admin.Delete("/planes/:id", planHandler.DeletePlan)
	}

	// Rutas Públicas de Planes
	if planHandler != nil {
		api.Get("/planes", planHandler.GetPlans)
		api.Get("/planes/:id", planHandler.GetPlan)
	}

	// Rutas de gestión de asesores
	if advisorHandler != nil {
		admin.Get("/advisors", advisorHandler.GetAdvisors)
		admin.Get("/advisors/:id", advisorHandler.GetAdvisor)
		admin.Post("/advisors", advisorHandler.CreateAdvisor)
		admin.Put("/advisors/:id", advisorHandler.UpdateAdvisor)
		admin.Delete("/advisors/:id", advisorHandler.DeleteAdvisor)
		admin.Patch("/advisors/:id/toggle-active", advisorHandler.ToggleAdvisorActive)
	}

	// Rutas de administración de propiedades
	if adminPropertyHandler != nil {
		admin.Get("/properties", adminPropertyHandler.ListProperties)
		admin.Get("/properties/stats", adminPropertyHandler.GetPropertiesStats)
		admin.Get("/properties/:id", adminPropertyHandler.GetProperty)
		admin.Post("/properties", adminPropertyHandler.CreateProperty)
		admin.Put("/properties/:id", adminPropertyHandler.UpdateProperty)
		admin.Patch("/properties/:id/status", adminPropertyHandler.UpdatePropertyStatus)
		admin.Patch("/properties/:id/featured", adminPropertyHandler.ToggleFeatured)
		admin.Delete("/properties/:id", adminPropertyHandler.DeleteProperty)
		admin.Patch("/properties/:id/advisor", adminPropertyHandler.AssignAdvisor)
	}

	// Rutas de imágenes (DEBEN IR ANTES que /properties/:id para evitar conflictos)
	if imageHandler != nil {
		// Rutas protegidas para subir imágenes
		api.Post("/imagenes/upload", middleware.AuthMiddleware(), imageHandler.UploadImage)
		api.Post("/imagenes/upload-base64", middleware.AuthMiddleware(), imageHandler.UploadImagesFromBase64)

		// Ruta pública para servir imágenes (ESTA PRIMERO por tener :filename)
		api.Get("/imagenes/:filename", imageHandler.ServeImage)
	}

	// Rutas de propiedades
	if propertyHandler != nil {
		// IMPORTANTE: En Fiber, el orden de las rutas importa.
		// Las rutas MÁS ESPECÍFICAS deben ir ANTES que las rutas con parámetros.

		// Rutas ESPECÍFICAS (protegidas) - van primero
		api.Get("/properties/my", middleware.AuthMiddleware(), propertyHandler.GetUserProperties)         // Mis propiedades (requiere login)
		api.Patch("/properties/:id/status", middleware.AuthMiddleware(), propertyHandler.UpdatePropertyStatus) // Actualizar estado (requiere login)

		// Rutas PÚBLICAS (van después)
		api.Get("/properties", propertyHandler.ListProperties)              // Listar propiedades (público)
		api.Get("/properties/search", propertyHandler.SearchProperties)    // Búsqueda full-text (público)
		api.Get("/properties/:id", propertyHandler.GetProperty)            // Ver detalle de propiedad (público)

		// Rutas protegidas restantes
		api.Post("/properties", middleware.AuthMiddleware(), propertyHandler.CreateProperty)              // Crear propiedad (requiere login)
		api.Delete("/properties/:id", middleware.AuthMiddleware(), propertyHandler.DeleteProperty)        // Eliminar propiedad (requiere login)
	}

	// 🚀 WebSocket para actualizaciones en tiempo real
	if wsHandler != nil {
		// WebSocket upgrade endpoint
		app.Get("/ws/properties", websocket.New(wsHandler.HandlePropertyUpdates))
	}

	// 🔒 Middleware NotFound (debe ir al final)
	app.Use(middleware.NotFoundMiddleware())

	// Iniciar servidor
	log.Println("🚀 Servidor corriendo en http://localhost:3001")
	log.Println("📊 Endpoints disponibles:")
	log.Println("  POST   /api/auth/register        - Registro de usuario")
	log.Println("  POST   /api/auth/login           - Login de usuario")
	log.Println("  GET    /api/auth/me              - Obtener usuario actual (requiere token)")
	log.Println("  PUT    /api/auth/profile         - Actualizar perfil (requiere token)")
	log.Println("  PUT    /api/auth/datos-publicos  - Actualizar datos públicos (requiere token)")
	log.Println("  GET    /api/properties           - Listar propiedades (público)")
	log.Println("  GET    /api/properties/search    - Búsqueda full-text (público)")
	log.Println("  GET    /api/properties/:id       - Obtener propiedad (público)")
	log.Println("  POST   /api/properties           - Crear propiedad (requiere token)")
	log.Println("  GET    /api/properties/my        - Mis propiedades (requiere token)")
	log.Println("  PATCH  /api/properties/:id/status - Actualizar estado (requiere token)")
	log.Println("  DELETE /api/properties/:id       - Eliminar propiedad (requiere token)")
	if wsHandler != nil {
		log.Println("  WS     /ws/properties          - Actualizaciones en tiempo real (WebSocket)")
	}
	log.Fatal(app.Listen(":3001"))
}
