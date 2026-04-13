package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// RateLimitMiddleware - Middleware de rate limiting con Redis
type RateLimitMiddleware struct {
	redisClient *redis.Client
	requests    int
	window      time.Duration
}

// NewRateLimitMiddleware - Crea una nueva instancia
func NewRateLimitMiddleware(redisClient *redis.Client, requests int, window time.Duration) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		redisClient: redisClient,
		requests:    requests,
		window:      window,
	}
}

// RateLimit - Aplica rate limiting
func (m *RateLimitMiddleware) RateLimit() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Obtener identificador del cliente
		clientID := m.getClientID(c)

		// 2. Verificar rate limit
		key := fmt.Sprintf("ratelimit:%s:%s", clientID, c.Path())

		// Usar Redis INCR para contador
		ctx := c.Context()
		current, err := m.redisClient.Get(ctx, key).Int()
		if err != nil && err != redis.Nil {
			// Error de Redis, permitir request (fail open)
			return c.Next()
		}

		if current >= m.requests {
			return c.Status(429).JSON(fiber.Map{
				"error": "Too many requests",
				"message": fmt.Sprintf("Máximo %d requests por %s", m.requests, m.window),
			})
		}

		// 3. Incrementar contador
		if current == 0 {
			// Primer request, crear key con expiry
			m.redisClient.Set(ctx, key, 1, m.window)
		} else {
			m.redisClient.Incr(ctx, key)
		}

		return c.Next()
	}
}

// getClientID - Obtiene identificador del cliente
func (m *RateLimitMiddleware) getClientID(c *fiber.Ctx) string {
	// Prioridades:
	// 1. API Key (si existe)
	// 2. User ID (si está autenticado)
	// 3. IP Address

	// 1. Verificar API Key
	apiKey := c.Get("X-API-Key")
	if apiKey != "" {
		return "apikey:" + apiKey
	}

	// 2. Verificar usuario autenticado
	if userID := c.Locals("user_id"); userID != nil {
		return fmt.Sprintf("user:%v", userID)
	}

	// 3. Usar IP address
	return "ip:" + c.IP()
}

// RateLimitByEndpoint - Rate limiting específico por endpoint
func (m *RateLimitMiddleware) RateLimitByEndpoint(limits map[string]int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()

		// Obtener límite para este endpoint
		limit, exists := limits[path]
		if !exists {
			limit = 100 // Default
		}

		// Usar rate limiting específico
		key := fmt.Sprintf("ratelimit:endpoint:%s:%s", path, m.getClientID(c))

		ctx := c.Context()
		current, err := m.redisClient.Get(ctx, key).Int()
		if err != nil && err != redis.Nil {
			return c.Next()
		}

		if int64(current) >= int64(limit) {
			return c.Status(429).JSON(fiber.Map{
				"error": "Too many requests for this endpoint",
			})
		}

		// Incrementar
		if current == 0 {
			m.redisClient.Set(ctx, key, 1, time.Minute)
		} else {
			m.redisClient.Incr(ctx, key)
		}

		return c.Next()
	}
}
