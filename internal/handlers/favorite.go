package handlers

import (
	"fmt"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// FavoriteHandler - Handler para favoritos
type FavoriteHandler struct {
	DB *gorm.DB
}

// NewFavoriteHandler - Crea un nuevo handler de favoritos
func NewFavoriteHandler() (*FavoriteHandler, error) {
	dbConfig, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	return &FavoriteHandler{
		DB: dbConfig.DB,
	}, nil
}

// AddFavorite - Agrega una propiedad a favoritos
func (h *FavoriteHandler) AddFavorite(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	
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

	// Verificar si ya es favorito (incluyendo los eliminados lógicamente)
	var existing models.Favorite
	if err := h.DB.Unscoped().Where("user_id = ? AND property_id = ?", userID, req.PropertyID).First(&existing).Error; err == nil {
		if existing.DeletedAt.Valid {
			// Restaurar favorito si estaba eliminado
			if err := h.DB.Unscoped().Model(&existing).Update("deleted_at", nil).Error; err != nil {
				fmt.Printf("Error restaurando favorito: %v\n", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error adding favorite"})
			}
			// Retornar la versión restaurada
			existing.DeletedAt = gorm.DeletedAt{Valid: false}
			return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Added to favorites", "favorite": existing})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Already in favorites"})
	}

	fav := models.Favorite{
		UserID:     userID,
		PropertyID: req.PropertyID,
	}

	if err := h.DB.Create(&fav).Error; err != nil {
		fmt.Printf("Error creando favorito: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error adding favorite"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Added to favorites", "favorite": fav})
}

// RemoveFavorite - Elimina de favoritos
func (h *FavoriteHandler) RemoveFavorite(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	propertyID := c.Params("id")

	result := h.DB.Unscoped().Where("user_id = ? AND property_id = ?", userID, propertyID).Delete(&models.Favorite{})
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error removing favorite"})
	}

	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Favorite not found"})
	}

	return c.JSON(fiber.Map{"message": "Removed from favorites"})
}

// GetFavorites - Lista los favoritos del usuario
func (h *FavoriteHandler) GetFavorites(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var favorites []models.Favorite
	if err := h.DB.Preload("Property.PropertyImages").Preload("Property.PropertyAdvisors.Advisor").Where("user_id = ?", userID).Find(&favorites).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error fetching favorites"})
	}

	return c.JSON(fiber.Map{"favorites": favorites})
}

// CheckFavorite - Verifica si una propiedad es favorita
func (h *FavoriteHandler) CheckFavorite(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	propertyID := c.Params("propertyId")

	var fav models.Favorite
	err := h.DB.Where("user_id = ? AND property_id = ?", userID, propertyID).First(&fav).Error
	
	isFavorite := err == nil

	return c.JSON(fiber.Map{"isFavorite": isFavorite})
}
