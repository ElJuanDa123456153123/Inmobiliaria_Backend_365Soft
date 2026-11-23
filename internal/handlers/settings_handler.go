package handlers

import (
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// SettingsHandler - Configuración global del sitio
type SettingsHandler struct {
	DB *gorm.DB
}

// NewSettingsHandler - Crea un nuevo handler de configuración
func NewSettingsHandler() (*SettingsHandler, error) {
	dbConfig, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	return &SettingsHandler{
		DB: dbConfig.DB,
	}, nil
}

// accentColorValid - Valores permitidos para el color de acento
func accentColorValid(color string) bool {
	switch color {
	case "", "cyan", "blue", "green", "purple", "orange", "red":
		return true
	}
	return false
}

// UpdateSettingsRequest - Estructura para actualizar configuración
type UpdateSettingsRequest struct {
	AccentColor string `json:"accent_color"`
}

// GetPublicSettings - Devuelve la configuración pública del sitio
// GET /api/settings/public
func (h *SettingsHandler) GetPublicSettings(c *fiber.Ctx) error {
	var settings []models.SiteSetting
	if err := h.DB.Find(&settings).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo configuración",
		})
	}

	result := fiber.Map{}
	for _, s := range settings {
		result[s.Key] = s.Value
	}

	return c.JSON(result)
}

// UpdateSettings - Actualiza la configuración del sitio (solo admin)
// PUT /api/admin/settings
func (h *SettingsHandler) UpdateSettings(c *fiber.Ctx) error {
	var req UpdateSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parsing request",
		})
	}

	if !accentColorValid(req.AccentColor) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Color de acento inválido",
		})
	}

	// "cyan" o vacío = volver al color por defecto: se elimina la clave
	var setting models.SiteSetting
	err := h.DB.Where("setting_key = ?", "accent_color").First(&setting).Error

	if req.AccentColor == "" || req.AccentColor == "cyan" {
		if err == nil {
			if err := h.DB.Delete(&setting).Error; err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Error eliminando configuración",
				})
			}
		}

		return c.JSON(fiber.Map{
			"message":      "Configuración actualizada",
			"accent_color": "",
		})
	}

	if err != nil {
		// No existe: crear
		setting = models.SiteSetting{
			Key:   "accent_color",
			Value: req.AccentColor,
		}
		if err := h.DB.Create(&setting).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error guardando configuración",
			})
		}
	} else {
		// Existe: actualizar
		setting.Value = req.AccentColor
		if err := h.DB.Save(&setting).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error actualizando configuración",
			})
		}
	}

	return c.JSON(fiber.Map{
		"message":      "Configuración actualizada",
		"accent_color": setting.Value,
	})
}
