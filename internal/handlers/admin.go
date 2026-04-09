package handlers

import (
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// AdminHandler - Handler para administración
type AdminHandler struct {
	DB *gorm.DB
}

// NewAdminHandler - Crea un nuevo handler de administración
func NewAdminHandler() (*AdminHandler, error) {
	dbConfig, err := config.NewDatabase()
	if err != nil {
		return nil, err
	}

	return &AdminHandler{
		DB: dbConfig.DB,
	}, nil
}

// GetUsers - Obtiene todos los usuarios
func (h *AdminHandler) GetUsers(c *fiber.Ctx) error {
	var users []models.User

	if err := h.DB.Find(&users).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo usuarios",
		})
	}

	return c.JSON(fiber.Map{
		"users": users,
	})
}

// ToggleUserActive - Activa/desactiva un usuario
func (h *AdminHandler) ToggleUserActive(c *fiber.Ctx) error {
	// Obtener ID del usuario
	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de usuario es requerido",
		})
	}

	// Buscar usuario
	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Usuario no encontrado",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error buscando usuario",
		})
	}

	// Toggle activo
	user.Activo = !user.Activo

	// Guardar cambios
	if err := h.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error actualizando usuario",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Usuario actualizado exitosamente",
		"user":    user,
	})
}

// DeleteUser - Elimina un usuario
func (h *AdminHandler) DeleteUser(c *fiber.Ctx) error {
	// Obtener ID del usuario
	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de usuario es requerido",
		})
	}

	// Eliminar usuario
	if err := h.DB.Delete(&models.User{}, userID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error eliminando usuario",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Usuario eliminado exitosamente",
	})
}
