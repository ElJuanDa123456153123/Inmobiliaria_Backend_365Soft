package repository

import (
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"gorm.io/gorm"
)

// ViewHistoryRepositoryInterface - Interface del repositorio de historial de vistas
type ViewHistoryRepositoryInterface interface {
	GetByUser(userID uint) ([]models.ViewHistory, error)
	CountByUser(userID uint) (int64, error)
	HasViewedProperties(userID uint) (bool, error)
}

// ViewHistoryRepository - Repositorio de historial de vistas
type ViewHistoryRepository struct {
	DB *gorm.DB
}

// NewViewHistoryRepository - Crea un nuevo repositorio de historial de vistas
func NewViewHistoryRepository(db *gorm.DB) ViewHistoryRepositoryInterface {
	return &ViewHistoryRepository{DB: db}
}

// GetByUser - Obtiene todo el historial de vistas de un usuario
func (r *ViewHistoryRepository) GetByUser(userID uint) ([]models.ViewHistory, error) {
	var viewHistory []models.ViewHistory
	err := r.DB.Preload("Property.PropertyImages").
		Where("user_id = ?", userID).
		Order("viewed_at DESC").
		Find(&viewHistory).Error
	return viewHistory, err
}

// CountByUser - Cuenta las vistas de un usuario
func (r *ViewHistoryRepository) CountByUser(userID uint) (int64, error) {
	var count int64
	err := r.DB.Model(&models.ViewHistory{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

// HasViewedProperties - Verifica si un usuario ha visto propiedades
func (r *ViewHistoryRepository) HasViewedProperties(userID uint) (bool, error) {
	var count int64
	err := r.DB.Model(&models.ViewHistory{}).Where("user_id = ?", userID).Count(&count).Error
	return count > 0, err
}
