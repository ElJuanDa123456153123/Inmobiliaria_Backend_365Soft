package middleware

import (
	"errors"
	"fmt"
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
			// LOG TEMPORAL PARA DEBUG
			fmt.Printf("❌ Token rechazado - Error: %v, Token válido: %v\n", err, token.Valid)
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

		var user models.User

		// Si userID es 0, es el admin harcodeado
		if userID == 0 {
			user = models.User{
				ID:         0,
				Nombre:     "Administrador",
				Email:      "admin@365soft.com",
				Rol:        "admin",
				Activo:     true,
				Verificado: true,
			}
		} else {
			// Buscar usuario en la base de datos usando conexión singleton
			dbConfig, err := config.GetDatabase()
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Error conectando a la base de datos",
				})
			}

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
		}

		// Verificar si el usuario está activo
		if !user.Activo {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Usuario inactivo",
			})
		}

		// Inyectar usuario en el contexto
		c.Locals("user", &user)
		c.Locals("user_id", user.ID)
		c.Locals("is_admin", user.Rol == models.RoleAdmin)

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

		// Verificar rol (SOLO admin tiene acceso a estas rutas)
		if user.Rol != models.RoleAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Acceso denegado: se requiere rol de administrador",
			})
		}

		return c.Next()
	}
}

// AdvisorMiddleware - Middleware para verificar rol de asesor
func AdvisorMiddleware() fiber.Handler {
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

		// Conectar a la base de datos usando conexión singleton
		dbConfig, err := config.GetDatabase()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error conectando a la base de datos",
			})
		}

		// Verificar que el usuario tenga un perfil de asesor activo
		var advisor models.Advisor
		result := dbConfig.DB.Where("user_id = ? AND activo = ?", user.ID, true).First(&advisor)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Acceso denegado: no tienes perfil de asesor activo",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error verificando perfil de asesor",
			})
		}

		// Inyectar asesor en el contexto
		c.Locals("advisor", &advisor)

		return c.Next()
	}
}

// AdminOrAdvisorMiddleware - Middleware para verificar rol de admin o asesor
// Permite acceso a admins y asesores, pero inyecta el advisor si es asesor
func AdminOrAdvisorMiddleware() fiber.Handler {
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

		// Si es admin, permitir acceso completo
		if user.Rol == models.RoleAdmin {
			c.Locals("isAdmin", true)
			return c.Next()
		}

		// Si es agente o inmobiliaria, verificar que tenga perfil de asesor activo
		if user.Rol == models.RoleAgente || user.Rol == models.RoleInmobiliaria {
			dbConfig, err := config.GetDatabase()
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Error conectando a la base de datos",
				})
			}

			var advisor models.Advisor
			result := dbConfig.DB.Where("user_id = ? AND activo = ?", user.ID, true).First(&advisor)
			if result.Error != nil {
				if errors.Is(result.Error, gorm.ErrRecordNotFound) {
					return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
						"error": "Acceso denegado: no tienes perfil de asesor activo",
					})
				}
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Error verificando perfil de asesor",
				})
			}

			// Inyectar asesor en el contexto para que los handlers puedan filtrar datos
			c.Locals("advisor", &advisor)
			c.Locals("isAdvisor", true)
			return c.Next()
		}

		// Si no es admin, agente ni inmobiliaria, denegar acceso
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Acceso denegado: se requiere rol de administrador o asesor",
		})
	}
}

// PropertyOwnerMiddleware - Middleware para verificar que el usuario tiene al menos 1 propiedad ACTIVA
// Requiere: Usuario autenticado (debe ejecutarse después de AuthMiddleware)
// Usado en: /panel/*
func PropertyOwnerMiddleware() fiber.Handler {
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

		// Conectar a la base de datos usando conexión singleton
		dbConfig, err := config.GetDatabase()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error conectando a la base de datos",
			})
		}

		// Contar propiedades activas del usuario
		var count int64
		if err := dbConfig.DB.Model(&models.Property{}).
			Where("user_id = ? AND estado_propiedad = ? AND deleted_at IS NULL", user.ID, models.PropertyStatusActivo).
			Count(&count).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error verificando propiedades activas",
			})
		}

		// Si no tiene propiedades activas, denegar acceso
		if count == 0 {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":       "Acceso denegado: necesitas al menos 1 propiedad activa para acceder al panel",
				"message":     "Debes publicar una propiedad para acceder a esta sección",
				"redirect_to": "/publicar",
			})
		}

		// Inyectar cantidad de propiedades activas en el contexto
		c.Locals("active_properties_count", count)

		return c.Next()
	}
}

// ContractOwnerMiddleware - Middleware para verificar que el usuario tiene al menos 1 contrato (activo o histórico)
// Requiere: Usuario autenticado (debe ejecutarse después de AuthMiddleware)
// Usado en: /mis-contratos/*
func ContractOwnerMiddleware() fiber.Handler {
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

		// Conectar a la base de datos usando conexión singleton
		dbConfig, err := config.GetDatabase()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error conectando a la base de datos",
			})
		}

		// Contar contratos del usuario (como vendedor o inquilino)
		var count int64
		if err := dbConfig.DB.Model(&models.RentalContract{}).
			Where("(vendedor_id = ? OR inquilino_id = ?) AND deleted_at IS NULL", user.ID, user.ID).
			Count(&count).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error verificando contratos",
			})
		}

		// Si no tiene contratos, denegar acceso
		if count == 0 {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "Acceso denegado: necesitas al menos 1 contrato para acceder a esta sección",
				"message": "No tienes contratos registrados",
			})
		}

		// Inyectar cantidad de contratos en el contexto
		c.Locals("contracts_count", count)

		return c.Next()
	}
}
