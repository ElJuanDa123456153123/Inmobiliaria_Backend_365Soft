package repository

import (
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"gorm.io/gorm"
)

// FavoriteRepositoryInterface - Interface del repositorio de favoritos
type FavoriteRepositoryInterface interface {
	GetByUser(userID uint) ([]models.Favorite, error)
	CountByUser(userID uint) (int64, error)
	HasFavorites(userID uint) (bool, error)
}

// FavoriteRepository - Repositorio de favoritos
type FavoriteRepository struct {
	DB *gorm.DB
}

// NewFavoriteRepository - Crea un nuevo repositorio de favoritos
func NewFavoriteRepository(db *gorm.DB) FavoriteRepositoryInterface {
	return &FavoriteRepository{DB: db}
}

// GetByUser - Obtiene todos los favoritos de un usuario
func (r *FavoriteRepository) GetByUser(userID uint) ([]models.Favorite, error) {
	var favorites []models.Favorite
	err := r.DB.Preload("Property.PropertyImages").
		Preload("Property.PropertyAdvisors.Advisor").
		Where("user_id = ?", userID).
		Find(&favorites).Error
	return favorites, err
}

// CountByUser - Cuenta los favoritos de un usuario
func (r *FavoriteRepository) CountByUser(userID uint) (int64, error) {
	var count int64
	err := r.DB.Model(&models.Favorite{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

// HasFavorites - Verifica si un usuario tiene favoritos
func (r *FavoriteRepository) HasFavorites(userID uint) (bool, error) {
	var count int64
	err := r.DB.Model(&models.Favorite{}).Where("user_id = ?", userID).Count(&count).Error
	return count > 0, err
}
