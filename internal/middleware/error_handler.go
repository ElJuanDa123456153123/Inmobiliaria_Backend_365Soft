package middleware

import (
	"fmt"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/response"
	apperrors "github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/errors"
	"github.com/gofiber/fiber/v2"
)

// ErrorHandlerMiddleware - Maneja errores de forma centralizada
func ErrorHandlerMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Continuar con el chain
		err := c.Next()

		// Si no hubo error, retornar
		if err == nil {
			return nil
		}

		// Manejar el error
		return c.Status(getHTTPStatus(err)).JSON(response.FromError(err))
	}
}

// getHTTPStatus - Obtiene HTTP status desde error
func getHTTPStatus(err error) int {
	// Si es AppError, usar su HTTP status
	if appErr, ok := err.(*apperrors.AppError); ok {
		if appErr.HTTPStatus != 0 {
			return appErr.HTTPStatus
		}
		return apperrors.GetHTTPStatus(appErr)
	}

	// Error genérico
	return 500
}

// RecoverMiddleware - Recupera de panics
func RecoverMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
		if r := recover(); r != nil {
			// Loggear el panic
			fmt.Printf("PANIC recovered: %v", r)

			// Retornar error 500
			c.Status(500).JSON(fiber.Map{
				"error": "Error interno del servidor",
				"request_id": c.Locals("request_id"),
			})
		}
	}()

		return c.Next()
	}
}

// NotFoundMiddleware - Maneja rutas no encontradas
func NotFoundMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Status(404).JSON(fiber.Map{
			"error": "Ruta no encontrada",
			"path":  c.Path(),
			"method": c.Method(),
		})
	}
}

// CORSMiddleware - Middleware de CORS
func CORSMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
		c.Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")

		if c.Method() == "OPTIONS" {
			return c.SendStatus(200)
		}

		return c.Next()
	}
}

// LoggingMiddleware - Middleware de logging
func LoggingMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Continuar
		err := c.Next()

		// Loggear request
		duration := time.Since(start)

		// Loggear en formato estructurado
		fmt.Printf("[%s] %s %s - %d (%dms)\n",
			time.Now().Format("2006-01-02 15:04:05"),
			c.Method(),
			c.Path(),
			c.Response().StatusCode(),
			duration.Milliseconds(),
		)

		return err
	}
}

// SecurityHeadersMiddleware - Añade headers de seguridad
func SecurityHeadersMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Set("X-Request-ID", c.Locals("request_id").(string))

		return c.Next()
	}
}
