package handlers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ViewHistoryHandler - Handler para historial de vistas
type ViewHistoryHandler struct {
	DB *gorm.DB
}

// NewViewHistoryHandler - Crea un nuevo handler de historial de vistas
func NewViewHistoryHandler() (*ViewHistoryHandler, error) {
	dbConfig, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	return &ViewHistoryHandler{
		DB: dbConfig.DB,
	}, nil
}

// RecordView - Registra una vista de propiedad (POST /api/view-history)
func (h *ViewHistoryHandler) RecordView(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	type Request struct {
		PropertyID uint `json:"property_id"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Verificar si la propiedad existe
	var prop models.Property
	if err := h.DB.First(&prop, req.PropertyID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Property not found"})
	}

	// Verificar si ya existe un registro de vista para esta propiedad del usuario
	var existing models.ViewHistory
	if err := h.DB.Where("user_id = ? AND property_id = ?", userID, req.PropertyID).First(&existing).Error; err == nil {
		// Ya existe: actualizar la fecha de vista
		existing.ViewedAt = time.Now()
		if err := h.DB.Save(&existing).Error; err != nil {
			fmt.Printf("Error actualizando historial de vista: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error recording view"})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "View updated", "view": existing})
	}

	// No existe: crear nuevo registro
	view := models.ViewHistory{
		UserID:     userID,
		PropertyID: req.PropertyID,
		ViewedAt:   time.Now(),
	}

	if err := h.DB.Create(&view).Error; err != nil {
		fmt.Printf("Error creando historial de vista: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error recording view"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "View recorded", "view": view})
}

// GetViewHistory - Obtiene el historial de vistas del usuario (GET /api/view-history)
func (h *ViewHistoryHandler) GetViewHistory(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	// Paginación
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Contar total
	var total int64
	h.DB.Model(&models.ViewHistory{}).Where("user_id = ?", userID).Count(&total)

	// Obtener historial con las propiedades y sus imágenes
	var views []models.ViewHistory
	if err := h.DB.
		Preload("Property.PropertyImages").
		Where("user_id = ?", userID).
		Order("viewed_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&views).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error fetching view history"})
	}

	return c.JSON(fiber.Map{
		"views": views,
		"total": total,
		"page":  page,
		"limit": limit,
		"pages": (total + int64(limit) - 1) / int64(limit),
	})
}

// ClearViewHistory - Limpia todo el historial del usuario (DELETE /api/view-history)
func (h *ViewHistoryHandler) ClearViewHistory(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	result := h.DB.Where("user_id = ?", userID).Delete(&models.ViewHistory{})
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error clearing view history"})
	}

	return c.JSON(fiber.Map{
		"message": "View history cleared",
		"deleted": result.RowsAffected,
	})
}

// DeleteViewHistoryItem - Elimina un registro del historial (DELETE /api/view-history/:id)
func (h *ViewHistoryHandler) DeleteViewHistoryItem(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	viewID := c.Params("id")

	result := h.DB.Where("id = ? AND user_id = ?", viewID, userID).Delete(&models.ViewHistory{})
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error deleting view history item"})
	}

	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "View history item not found"})
	}

	return c.JSON(fiber.Map{"message": "View history item deleted"})
}
