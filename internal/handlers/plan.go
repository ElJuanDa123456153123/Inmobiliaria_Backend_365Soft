package handlers

import (
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// PlanHandler - Handler para la gestión de planes
type PlanHandler struct {
	DB *gorm.DB
}

// NewPlanHandler - Crea un nuevo handler de planes
func NewPlanHandler() (*PlanHandler, error) {
	dbConfig, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	return &PlanHandler{
		DB: dbConfig.DB,
	}, nil
}

// GetPlans - Obtiene todos los planes (Admin - muestra todos incluyendo inactivos)
func (h *PlanHandler) GetPlans(c *fiber.Ctx) error {
	var planes []models.Plan

	if err := h.DB.Order("orden asc").Find(&planes).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo planes",
		})
	}

	return c.JSON(fiber.Map{
		"planes": planes,
	})
}

// GetActivePlans - Obtiene solo los planes activos (Público - para la página de publicar)
func (h *PlanHandler) GetActivePlans(c *fiber.Ctx) error {
	var planes []models.Plan

	if err := h.DB.Where("activo = ?", true).Order("orden asc").Find(&planes).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo planes",
		})
	}

	return c.JSON(fiber.Map{
		"planes": planes,
	})
}

// GetPlan - Obtiene un plan por ID (solo activos para público)
func (h *PlanHandler) GetPlan(c *fiber.Ctx) error {
	id := c.Params("id")
	var plan models.Plan

	if err := h.DB.Where("activo = ?", true).First(&plan, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Plan no encontrado",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error buscando el plan",
		})
	}

	return c.JSON(plan)
}

// CreatePlan - Crea un nuevo plan (Admin access)
func (h *PlanHandler) CreatePlan(c *fiber.Ctx) error {
	var plan models.Plan
	if err := c.BodyParser(&plan); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	if plan.Nombre == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "El nombre del plan es requerido",
		})
	}

	if err := h.DB.Create(&plan).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creando el plan",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Plan creado exitosamente",
		"plan":    plan,
	})
}

// UpdatePlan - Actualiza un plan existente (Admin access)
func (h *PlanHandler) UpdatePlan(c *fiber.Ctx) error {
	id := c.Params("id")
	var existingPlan models.Plan

	if err := h.DB.First(&existingPlan, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Plan no encontrado",
		})
	}

	if err := c.BodyParser(&existingPlan); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	if err := h.DB.Save(&existingPlan).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error actualizando el plan",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Plan actualizado exitosamente",
		"plan":    existingPlan,
	})
}

// DeletePlan - Elimina un plan (Admin access)
func (h *PlanHandler) DeletePlan(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.DB.Delete(&models.Plan{}, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error eliminando el plan",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Plan eliminado exitosamente",
	})
}

