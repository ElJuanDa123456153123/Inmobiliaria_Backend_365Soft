package main

import (
	"log"
	"os"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/handlers"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/middleware"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/repository"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// SetupRoutes - Configura todas las rutas de la aplicación
func SetupRoutes(app *fiber.App) {
	// 1. Conectar a base de datos
	dbConfig, err := config.NewDatabase()
	if err != nil {
		log.Fatalf("Error conectando a base de datos: %v", err)
	}

	// 2. Conectar a Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})

	// 3. Crear repositorios
	propertyRepo := repository.NewPropertyRepository(dbConfig.DB)
	categoryRepo := repository.NewCategoryRepository(dbConfig.DB)
	auditRepo := repository.NewAuditRepository(dbConfig.DB)

	// 4. Crear servicios
	storageService := services.NewStorageService(
		getEnv("UPLOAD_DIR", "./uploads"),
		getEnv("UPLOAD_PUBLIC_URL", "http://localhost:3001/uploads"),
	)

	cacheService, err := services.NewCacheService(
		getEnv("REDIS_ADDR", "localhost:6379"),
		getEnv("REDIS_PASSWORD", ""),
		0,
	)
	if err != nil {
		log.Printf("Warning: Error conectando a Redis: %v. El caché estará deshabilitado.", err)
		// Crear caché dummy que no hace nada
		cacheService = &dummyCacheService{}
	}

	propertyService := services.NewPropertyService(propertyRepo, categoryRepo, auditRepo)
	searchService := services.NewSearchService(propertyRepo)

	// 5. Crear handlers
	propertyHandler := handlers.NewPropertyHandler(
		propertyService,
		searchService,
		storageService,
		cacheService,
	)

	// 6. Configurar middleware
	app.Use(middleware.RecoverMiddleware())
	app.Use(middleware.RequestIDMiddleware())
	app.Use(middleware.LoggingMiddleware())
	app.Use(middleware.CORSMiddleware())
	app.Use(middleware.SecurityHeadersMiddleware())

	// Rate limiting (opcional, descomentar si se desea)
	// rateLimit := middleware.NewRateLimitMiddleware(redisClient, 100, 1*time.Minute)
	// app.Use(middleware.RateLimitMiddleware().RateLimit())

	// 7. Middleware de autenticación (para rutas protegidas)
	app.Use(middleware.AuthMiddleware())

	// 8. Middleware de auditoría
	// auditMiddleware := middleware.NewAuditMiddleware(auditRepo)
	// app.Use(auditMiddleware.Audit())

	// 9. Middleware de error handling
	app.Use(middleware.ErrorHandlerMiddleware())

	// 10. Registrar rutas
	registerRoutes(app, propertyHandler)

	// 11. Handler para 404
	app.Use(middleware.NotFoundMiddleware())
}

// registerRoutes - Registra todas las rutas de la API
func registerRoutes(app *fiber.App, propertyHandler *handlers.PropertyHandler) {
	// Rutas de propiedades
	propertyHandler.RegisterRoutes(app)

	// Rutas de autenticación (ya existen en auth_handler.go)
	// Se pueden reorganizar si es necesario

	// Rutas de salud (health check)
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(map[string]interface{}{
			"status": "ok",
			"service": "Inmobiliaria 365Soft API",
			"version": "1.0.0",
		})
	})

	app.Get("/ping", func(c *fiber.Ctx) error {
		return c.SendString("pong")
	})
}

// getEnv - Obtiene variable de entorno con valor por defecto
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		value = defaultValue
	}
	return value
}

// dummyCacheService - Implementación dummy de caché
type dummyCacheService struct{}

func (d *dummyCacheService) GetProperty(ctx interface{}, id uint) (interface{}, error) {
	return nil, nil
}

func (d *dummyCacheService) SetProperty(ctx interface{}, property interface{}, ttl interface{}) error {
	return nil
}

func (d *dummyCacheService) InvalidateProperty(ctx interface{}, id uint) error {
	return nil
}

func (d *dummyCacheService) GetPropertyList(ctx interface{}, filter string) (interface{}, error) {
	return nil, nil
}

func (d *dummyCacheService) SetPropertyList(ctx interface{}, filter string, properties interface{}, ttl interface{}) error {
	return nil
}

func (d *dummyCacheService) InvalidatePropertyList(ctx interface{}) error {
	return nil
}

func (d *dummyCacheService) GetCategories(ctx interface{}) (interface{}, error) {
	return nil, nil
}

func (d *dummyCacheService) SetCategories(ctx interface{}, categories interface{}, ttl interface{}) error {
	return nil
}

func (d *dummyCacheService) InvalidateCategories(ctx interface{}) error {
	return nil
}

func (d *dummyCacheService) IncrementVisits(ctx interface{}, propertyID uint) (int64, error) {
	return 0, nil
}

func (d *dummyCacheService) IncrementContacts(ctx interface{}, propertyID uint) (int64, error) {
	return 0, nil
}

func (d *dummyCacheService) ClearAll(ctx interface{}) error {
	return nil
}

func (d *dummyCacheService) GetStats(ctx interface{}) (interface{}, error) {
	return nil, nil
}
