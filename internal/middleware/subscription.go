package middleware

import (
	"fmt"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// SubscriptionMiddleware - Middleware para verificar que el usuario tenga suscripción activa
func SubscriptionMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Obtener user_id del JWT (ya verificado por AuthMiddleware)
		userID := c.Locals("user_id")
		if userID == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "No autorizado",
			})
		}

		// Convertir user_id a uint
		userIDUint, ok := userID.(uint)
		if !ok {
			// Intentar convertir desde float64 (JSON numbers)
			if userIDFloat, ok := userID.(float64); ok {
				userIDUint = uint(userIDFloat)
			} else {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Error obteniendo user_id",
				})
			}
		}

		// Conectar a la base de datos
		dbConfig, err := config.GetDatabase()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error conectando a la base de datos",
			})
		}

		// Buscar suscripción activa Y seleccionada del usuario
		var subscription models.Subscription
		err = dbConfig.DB.
			Where("user_id = ? AND estado = ? AND seleccionada = ?", userIDUint, "ACTIVA", true).
			Preload("Plan").
			First(&subscription).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// No tiene suscripción seleccionada, intentar buscar cualquier suscripción activa
				err = dbConfig.DB.
					Where("user_id = ? AND estado = ?", userIDUint, "ACTIVA").
					Preload("Plan").
					First(&subscription).Error

				if err != nil {
					if err == gorm.ErrRecordNotFound {
						// No tiene ninguna suscripción activa
						return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
							"error":       "No tienes una suscripción activa",
							"error_code":  "NO_SUBSCRIPTION",
							"message":     "Debes adquirir un plan para publicar propiedades",
							"redirect_to": "/planes",
						})
					}
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error": "Error verificando suscripción",
					})
				}

				// Si encontró una suscripción pero no estaba seleccionada, marcarla como seleccionada
				fmt.Printf("⚠️  Usuario %d no tenía suscripción seleccionada, marcando suscripción %d como seleccionada\n", userIDUint, subscription.ID)

				// Desmarcar otras suscripciones del usuario
				dbConfig.DB.Model(&models.Subscription{}).
					Where("user_id = ? AND id != ?", userIDUint, subscription.ID).
					Update("seleccionada", false)

				// Marcar esta como seleccionada
				dbConfig.DB.Model(&subscription).Update("seleccionada", true)
			} else {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Error verificando suscripción",
				})
			}
		}

		// Verificar que la suscripción no esté vencida
		if subscription.EstaVencida() {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":       "Tu suscripción ha vencido",
				"error_code":  "SUBSCRIPTION_EXPIRED",
				"message":     "Debes renovar tu suscripción para publicar propiedades",
				"redirect_to": "/planes",
			})
		}

		// Verificar que tenga propiedades disponibles
		if !subscription.PuedePublicar() {
			// Cargar el plan para mostrar el límite
			var plan models.Plan
			dbConfig.DB.First(&plan, subscription.PlanID)

			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":       "Has alcanzado el límite de propiedades de tu plan",
				"error_code":  "LIMIT_REACHED",
				"message":     fmt.Sprintf("Tu plan %s te permite publicar hasta %d propiedades", plan.Nombre, plan.LimitePropiedades),
				"redirect_to": "/planes",
				"limit":       plan.LimitePropiedades,
				"used":        subscription.PropiedadesUsadas,
			})
		}

		// Guardar subscription_id en el contexto para uso posterior
		c.Locals("subscription_id", subscription.ID)

		// Continuar al siguiente handler
		return c.Next()
	}
}
