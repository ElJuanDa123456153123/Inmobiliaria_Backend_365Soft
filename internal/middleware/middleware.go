package middleware

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gofiber/fiber/v2"
)

// ========== REQUEST ID ==========

// RequestIDMiddleware - Añade un ID único a cada request
func RequestIDMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Generar UUID para el request
		requestID := uuid.New().String()

		// Guardar en locals para usar después
		c.Locals("request_id", requestID)

		// Añadir header a la respuesta
		c.Set("X-Request-ID", requestID)

		return c.Next()
	}
}

// ========== SECURITY HEADERS ==========

// SecurityHeadersMiddleware - Añade headers de seguridad HTTP
func SecurityHeadersMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Prevenir clickjacking
		c.Set("X-Frame-Options", "DENY")

		// Prevenir MIME type sniffing
		c.Set("X-Content-Type-Options", "nosniff")

		// Activar XSS protection (modo block)
		c.Set("X-XSS-Protection", "1; mode=block")

		// HSTS (solo en HTTPS)
		// c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Referrer Policy
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy (básico)
		c.Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'")

		return c.Next()
	}
}

// ========== LOGGING ==========

// LoggingMiddleware - Loggea requests en formato estructurado
func LoggingMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Continuar con el handler
		err := c.Next()

		// Calcular duración
		duration := time.Since(start)

		// Obtener request ID
		requestID := c.Locals("request_id")
		if requestID == nil {
			requestID = "unknown"
		}

		// Loggear en formato estructurado
		// [timestamp] method path - status (duration) - request_id
		fmt.Printf(
			"[%s] %s %s - %d (%dms) - %s\n",
			time.Now().Format("2006-01-02 15:04:05"),
			c.Method(),
			c.Path(),
			c.Response().StatusCode(),
			duration.Milliseconds(),
			requestID,
		)

		return err
	}
}

// ========== RECOVER ==========

// RecoverMiddleware - Recupera de panics y retorna 500
func RecoverMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				// Obtener request ID
				requestID := c.Locals("request_id")
				if requestID == nil {
					requestID = "unknown"
				}

				// Loggear el panic
				fmt.Printf(
					"[PANIC] %s %s - %v - RequestID: %s\n",
					time.Now().Format("2006-01-02 15:04:05"),
					c.Path(),
					r,
					requestID,
				)

				// Retornar error 500
				c.Status(500).JSON(fiber.Map{
					"success": false,
					"error":    "Error interno del servidor",
					"request_id": requestID,
				})
			}
		}()

		return c.Next()
	}
}

// ========== ERROR HANDLING ==========

// ErrorHandlerMiddleware - Maneja errores de forma centralizada
func ErrorHandlerMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Continuar con el chain
		err := c.Next()

		// Si no hubo error, retornar
		if err == nil {
			return nil
		}

		// Obtener request ID
		requestID := c.Locals("request_id")
		if requestID == nil {
			requestID = "unknown"
		}

		// Loggear error
		fmt.Printf(
			"[ERROR] %s %s - %v - RequestID: %s\n",
			time.Now().Format("2006-01-02 15:04:05"),
			c.Path(),
			err,
			requestID,
		)

		// Retornar respuesta de error
		statusCode := c.Response().StatusCode()
		if statusCode == 0 {
			statusCode = 500
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"error":    err.Error(),
			"request_id": requestID,
		})
	}
}

// ========== NOT FOUND ==========

// NotFoundMiddleware - Maneja rutas no encontradas
func NotFoundMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Continuar al siguiente middleware/ruta
		err := c.Next()

		// Si el status code ya fue seteado (no es 404 por defecto), una ruta manejó la petición
		// Fiber setea status code 200 por default cuando una ruta maneja la petición
		if c.Response().StatusCode() != 404 {
			return err
		}

		// Si llegamos aquí con status 404, ninguna ruta coincidió
		// Solo retornar JSON 404 si no hay otro error
		if err == nil {
			return c.Status(404).JSON(fiber.Map{
				"success": false,
				"error":    "Ruta no encontrada",
				"path":     c.Path(),
				"method":   c.Method(),
			})
		}

		return err
	}
}
