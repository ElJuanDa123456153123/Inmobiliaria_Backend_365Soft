package handlers

import (
	"strconv"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/repository"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/service"
	"github.com/gofiber/fiber/v2"
)

// AdvisorHandler - Handler para operaciones de asesores
type AdvisorHandler struct {
	AssignmentService *service.AssignmentService
	PropertyService   *service.PropertyService
}

// NewAdvisorHandler - Crea un nuevo handler de asesores
func NewAdvisorHandler() (*AdvisorHandler, error) {
	db, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	// Inicializar repositorios
	advisorRepo := repository.NewAdvisorRepository(db)
	propertyRepo := repository.NewPropertyRepository(db)
	notificationRepo := repository.NewNotificationRepository(db.DB)

	// Inicializar servicios
	assignmentService := service.NewAssignmentService(advisorRepo, propertyRepo, notificationRepo, db.DB)

	// PropertyService ya debería estar inicializado, lo pasamos como referencia
	propertyService := &service.PropertyService{} // Se inyectará desde fuera

	return &AdvisorHandler{
		AssignmentService: assignmentService,
		PropertyService:   propertyService,
	}, nil
}

// GetProperties - Obtiene las propiedades del asesor autenticado
// GET /api/advisor/properties?status=pendiente|activo|vendido
func (h *AdvisorHandler) GetProperties(c *fiber.Ctx) error {
	// Obtener asesor del contexto (inyectado por AdvisorMiddleware)
	advisor, ok := c.Locals("advisor").(*models.Advisor)
	if !ok || advisor == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No se pudo obtener el perfil de asesor",
		})
	}

	// Obtener filtro de estado (opcional)
	statusParam := c.Query("status", "")
	var status models.PropertyStatus
	if statusParam != "" {
		status = models.PropertyStatus(statusParam)
	}

	// Obtener propiedades
	properties, err := h.AssignmentService.GetAdvisorProperties(advisor.ID, status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo propiedades",
		})
	}

	return c.JSON(fiber.Map{
		"properties": properties,
		"total":      len(properties),
	})
}

// ActivateProperty - Activa una propiedad (solo el asesor asignado puede hacerlo)
// PATCH /api/advisor/properties/:id/activate
func (h *AdvisorHandler) ActivateProperty(c *fiber.Ctx) error {
	// Obtener asesor del contexto (inyectado por AdvisorMiddleware)
	advisor, ok := c.Locals("advisor").(*models.Advisor)
	if !ok || advisor == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No se pudo obtener el perfil de asesor",
		})
	}

	// Obtener ID de la propiedad
	propertyIDStr := c.Params("id")
	propertyID, err := strconv.ParseUint(propertyIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de propiedad inválido",
		})
	}

	// Activar propiedad
	property, err := h.AssignmentService.ActivateProperty(uint(propertyID), advisor.ID)
	if err != nil {
		// Manejar diferentes tipos de error
		if err.Error() == "property not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Propiedad no encontrada",
			})
		}
		if err.Error() == "forbidden" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "No tienes permiso para activar esta propiedad",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error activando propiedad",
		})
	}

	return c.JSON(fiber.Map{
		"message":   "Propiedad activada exitosamente",
		"property":  property,
	})
}

// GetStats - Obtiene las estadísticas del asesor
// GET /api/advisor/stats
func (h *AdvisorHandler) GetStats(c *fiber.Ctx) error {
	// Obtener asesor del contexto (inyectado por AdvisorMiddleware)
	advisor, ok := c.Locals("advisor").(*models.Advisor)
	if !ok || advisor == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No se pudo obtener el perfil de asesor",
		})
	}

	// Obtener estadísticas
	stats, err := h.AssignmentService.GetAdvisorStats(advisor.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo estadísticas",
		})
	}

	return c.JSON(stats)
}

// getAdvisorByUserID - Obtiene el asesor asociado a un usuario
// NOTA: Este método ya no se necesita porque el AdvisorMiddleware inyecta el advisor
// Se mantiene por compatibilidad temporal
func (h *AdvisorHandler) getAdvisorByUserID(userID uint) (*models.Advisor, error) {
	// Buscar asesor por UserID
	var advisor models.Advisor
	db, err := config.NewDatabase()
	if err != nil {
		return nil, err
	}

	result := db.DB.Where("user_id = ? AND activo = ?", userID, true).First(&advisor)
	if result.Error != nil {
		return nil, result.Error
	}

	return &advisor, nil
}
