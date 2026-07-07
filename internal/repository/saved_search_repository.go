package repository

import (
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"gorm.io/gorm"
)

// SavedSearchRepositoryInterface - Interface del repositorio de búsquedas guardadas
type SavedSearchRepositoryInterface interface {
	GetActiveByUser(userID uint) ([]models.SavedSearch, error)
	GetByUser(userID uint) ([]models.SavedSearch, error)
}

// SavedSearchRepository - Repositorio de búsquedas guardadas
type SavedSearchRepository struct {
	DB *gorm.DB
}

// NewSavedSearchRepository - Crea un nuevo repositorio de búsquedas guardadas
func NewSavedSearchRepository(db *gorm.DB) SavedSearchRepositoryInterface {
	return &SavedSearchRepository{DB: db}
}

// GetActiveByUser - Obtiene las búsquedas guardadas activas de un usuario
func (r *SavedSearchRepository) GetActiveByUser(userID uint) ([]models.SavedSearch, error) {
	var savedSearches []models.SavedSearch
	err := r.DB.Where("user_id = ? AND is_active = ?", userID, true).
		Order("created_at DESC").
		Find(&savedSearches).Error
	return savedSearches, err
}

// GetByUser - Obtiene todas las búsquedas guardadas de un usuario (incluyendo inactivas)
func (r *SavedSearchRepository) GetByUser(userID uint) ([]models.SavedSearch, error) {
	var savedSearches []models.SavedSearch
	err := r.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&savedSearches).Error
	return savedSearches, err
}
