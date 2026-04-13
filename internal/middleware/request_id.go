package middleware

import (
	"github.com/google/uuid"
	"github.com/gofiber/fiber/v2"
)

// RequestIDMiddleware - Añade request ID único a cada request
func RequestIDMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Generar o obtener request ID
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Guardar en contexto
		c.Locals("request_id", requestID)

		// Añadir header de respuesta
		c.Set("X-Request-ID", requestID)

		return c.Next()
	}
}
