package middleware

import (
	"fmt"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/gofiber/fiber/v2"
)

// RateLimitConfig - Configuración para rate limiting
type RateLimitConfig struct {
	RequestsPerMinute int
	RequestsPerHour   int
	BanDuration       time.Duration
}

// RateLimiter - Middleware de rate limiting con Redis
type RateLimiter struct {
	Redis *config.RedisClient
}

// NewRateLimiter - Crea un nuevo rate limiter
func NewRateLimiter(redis *config.RedisClient) *RateLimiter {
	return &RateLimiter{Redis: redis}
}

// RateLimitMiddleware - Middleware que aplica rate limiting por IP
func (r *RateLimiter) RateLimitMiddleware(config RateLimitConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Si Redis no está disponible, permitir todas las solicitudes
		if r.Redis == nil {
			return c.Next()
		}

		clientIP := c.IP()
		now := time.Now().Unix()
		minuteKey := fmt.Sprintf("ratelimit:%s:%d", clientIP, now/60)
		hourKey := fmt.Sprintf("ratelimit:%s:%d", clientIP, now/3600)

		// Verificar rate limit por minuto
		minuteCount, err := r.Redis.Client.Get(r.Redis.Ctx, minuteKey).Int()
		if err == nil && minuteCount >= config.RequestsPerMinute {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": fmt.Sprintf("Demasiadas solicitudes. Máximo %d por minuto. Intenta de nuevo en %d segundos.",
					config.RequestsPerMinute, 60-(int(now)%60),
				),
			})
		}

		// Verificar rate limit por hora
		hourCount, err := r.Redis.Client.Get(r.Redis.Ctx, hourKey).Int()
		if err == nil && hourCount >= config.RequestsPerHour {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": fmt.Sprintf("Límite horario excedido. Máximo %d por hora. Intenta de nuevo en %d minutos.",
					config.RequestsPerHour, 60-(int(now)%3600/60),
				),
			})
		}

		// Incrementar contadores
		pipe := r.Redis.Client.Pipeline()
		pipe.Incr(r.Redis.Ctx, minuteKey)
		pipe.Expire(r.Redis.Ctx, minuteKey, 1*time.Minute)
		pipe.Incr(r.Redis.Ctx, hourKey)
		pipe.Expire(r.Redis.Ctx, hourKey, 1*time.Hour)
		_, _ = pipe.Exec(r.Redis.Ctx)

		// Agregar headers de rate limit a la respuesta
		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", config.RequestsPerMinute))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", config.RequestsPerMinute-(minuteCount+1)))
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", 60-(int(now)%60)))

		return c.Next()
	}
}

// RateLimitByUserMiddleware - Middleware que aplica rate limiting por usuario (requiere autenticación)
func (r *RateLimiter) RateLimitByUserMiddleware(config RateLimitConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Si Redis no está disponible, permitir todas las solicitudes
		if r.Redis == nil {
			return c.Next()
		}

		// Obtener usuario del contexto
		user := c.Locals("user")
		if user == nil {
			// Si no hay usuario, usar rate limit por IP
			return r.RateLimitMiddleware(config)(c)
		}

		// userId := user.(*models.User).ID
		// NOTA: Descomentar y usar el ID del usuario cuando se implemente
		// Por ahora usar IP
		return r.RateLimitMiddleware(config)(c)
	}
}

// RateLimitByEndpointMiddleware - Rate limiting específico por endpoint
type EndpointRateLimit struct {
	Path                 string
	Config               RateLimitConfig
	ApplyToAuthenticated bool
}

// RateLimitByEndpointsMiddleware - Middleware que aplica diferentes límites según el endpoint
func (r *RateLimiter) RateLimitByEndpointsMiddleware(endpoints []EndpointRateLimit) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if r.Redis == nil {
			return c.Next()
		}

		path := c.Path()

		// Buscar configuración para este endpoint
		for _, endpoint := range endpoints {
			if path == endpoint.Path {
				if endpoint.ApplyToAuthenticated {
					return r.RateLimitByUserMiddleware(endpoint.Config)(c)
				}
				return r.RateLimitMiddleware(endpoint.Config)(c)
			}
		}

		// Si no hay configuración específica, continuar sin rate limit
		return c.Next()
	}
}

// GetStandardRateLimits - Retorna configuraciones estándar de rate limiting
func GetStandardRateLimits() []EndpointRateLimit {
	return []EndpointRateLimit{
		{
			Path: "/api/auth/login",
			Config: RateLimitConfig{

				RequestsPerMinute: 30,
				RequestsPerHour:   100,
				BanDuration:       5 * time.Minute,

				/**RequestsPerMinute: 5,  // 5 intentos de login por minuto
				RequestsPerHour:   20, // Máximo 20 por hora
				BanDuration:       15 * time.Minute,**/
			},
			ApplyToAuthenticated: false,
		},
		{
			Path: "/api/auth/register",
			Config: RateLimitConfig{
				RequestsPerMinute: 10,
				RequestsPerHour:   50,
				BanDuration:       5 * time.Minute,
				/**
				RequestsPerMinute: 3,  // 3 registros por minuto
				RequestsPerHour:   10, // Máximo 10 por hora
				BanDuration:       30 * time.Minute,**/
			},
			ApplyToAuthenticated: false,
		},
		{
			Path: "/api/properties",
			Config: RateLimitConfig{
				RequestsPerMinute: 30,  // 30 listados por minuto
				RequestsPerHour:   500, // 500 por hora
				BanDuration:       5 * time.Minute,
			},
			ApplyToAuthenticated: false,
		},
		{
			Path: "/api/properties",
			Config: RateLimitConfig{
				RequestsPerMinute: 10,  // 10 creaciones por minuto para usuarios autenticados
				RequestsPerHour:   100, // 100 por hora
				BanDuration:       10 * time.Minute,
			},
			ApplyToAuthenticated: true, // Solo para POST (se filtra por método)
		},
		{
			Path: "/api/imagenes/upload",
			Config: RateLimitConfig{
				RequestsPerMinute: 10,  // 10 uploads por minuto
				RequestsPerHour:   100, // 100 por hora
				BanDuration:       10 * time.Minute,
			},
			ApplyToAuthenticated: true,
		},
	}
}

// WhitelistMiddleware - Rate limit más permisivo para endpoints específicos
func (r *RateLimiter) WhitelistMiddleware(whitelist []string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()

		// Verificar si está en whitelist
		for _, allowedPath := range whitelist {
			if path == allowedPath {
				// Rate limit más permisivo para endpoints en whitelist
				return r.RateLimitMiddleware(RateLimitConfig{
					RequestsPerMinute: 100,
					RequestsPerHour:   5000,
					BanDuration:       1 * time.Minute,
				})(c)
			}
		}

		// Si no está en whitelist, aplicar rate limit estándar
		return r.RateLimitMiddleware(RateLimitConfig{
			RequestsPerMinute: 60,
			RequestsPerHour:   1000,
			BanDuration:       5 * time.Minute,
		})(c)
	}
}
