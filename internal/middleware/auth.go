package middleware

import (
	"errors"
	"os"
	"strings"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// AuthMiddleware - Middleware de autenticación JWT
func AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Permitir preflight CORS OPTIONS requests sin autenticación
		if c.Method() == "OPTIONS" {
			return c.Next()
		}

		// Obtener token del header Authorization
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token no proporcionado",
			})
		}

		// Extraer token del formato "Bearer <token>"
		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)

		// Obtener JWT secret
		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			jwtSecret = "tu_secreto_super_seguro_cambialo_en_produccion"
		}

		// Parsear y validar token
		token, err := jwt.ParseWithClaims(tokenString, &jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token inválido",
			})
		}

		// Extraer claims
		claims, ok := token.Claims.(*jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Error extrayendo claims del token",
			})
		}

		// Obtener userID desde claims
		userID, ok := (*claims)["user_id"].(float64)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token inválido: no contiene user_id",
			})
		}

			// Buscar usuario en la base de datos
		dbConfig, err := config.NewDatabase()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error conectando a la base de datos",
			})
		}

		var user models.User
		result := dbConfig.DB.First(&user, uint(userID))
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Usuario no encontrado",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error buscando usuario",
			})
		}

		// Verificar si el usuario está activo
		if !user.Activo {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Usuario inactivo",
			})
		}

		// Inyectar usuario en el contexto
		c.Locals("user", &user)
		c.Locals("userID", user.ID)

		// Continuar al siguiente handler
		return c.Next()
	}
}

// AdminMiddleware - Middleware para verificar rol de administrador
func AdminMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Permitir preflight CORS OPTIONS requests sin verificación
		if c.Method() == "OPTIONS" {
			return c.Next()
		}

		// Obtener usuario del contexto (inyectado por AuthMiddleware)
		user, ok := c.Locals("user").(*models.User)
		if !ok || user == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Usuario no autenticado",
			})
		}

		// Verificar rol
		if user.Rol != models.RoleAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Acceso denegado: se requiere rol de administrador",
			})
		}

		return c.Next()
	}
}
