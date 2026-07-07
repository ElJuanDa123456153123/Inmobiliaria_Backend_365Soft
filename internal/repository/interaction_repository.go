package repository

import (
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"gorm.io/gorm"
)

// InteractionRepositoryInterface - Interface del repositorio de interacciones
type InteractionRepositoryInterface interface {
	Create(interaction *models.Interaction) error
	GetByID(id uint) (*models.Interaction, error)
	Update(interaction *models.Interaction) error
	Delete(id uint) error
	GetByClientID(clientID uint) ([]models.Interaction, error)
	CountRecentByClient(clientID uint, days int) (int64, error)
	GetLastByClient(clientID uint) (*models.Interaction, error)
}

// InteractionRepository - Implementación del repositorio
type InteractionRepository struct {
	db *gorm.DB
}

// NewInteractionRepository - Crea una nueva instancia del repositorio
func NewInteractionRepository(db *gorm.DB) InteractionRepositoryInterface {
	return &InteractionRepository{db: db}
}

// Create - Crea una nueva interacción
func (r *InteractionRepository) Create(interaction *models.Interaction) error {
	return r.db.Create(interaction).Error
}

// GetByID - Obtiene una interacción por ID
func (r *InteractionRepository) GetByID(id uint) (*models.Interaction, error) {
	var interaction models.Interaction
	err := r.db.Where("id = ?", id).First(&interaction).Error
	if err != nil {
		return nil, err
	}
	return &interaction, nil
}

// Update - Actualiza una interacción
func (r *InteractionRepository) Update(interaction *models.Interaction) error {
	return r.db.Save(interaction).Error
}

// Delete - Elimina una interacción (soft delete)
func (r *InteractionRepository) Delete(id uint) error {
	return r.db.Delete(&models.Interaction{}, id).Error
}

// GetByClientID - Obtiene todas las interacciones de un cliente
func (r *InteractionRepository) GetByClientID(clientID uint) ([]models.Interaction, error) {
	var interactions []models.Interaction
	err := r.db.Where("client_id = ?", clientID).
		Order("created_at DESC").
		Find(&interactions).Error
	return interactions, err
}

// CountRecentByClient - Cuenta las interacciones recientes de un cliente (últimos X días)
func (r *InteractionRepository) CountRecentByClient(clientID uint, days int) (int64, error) {
	var count int64
	err := r.db.Model(&models.Interaction{}).
		Where("client_id = ? AND created_at >= NOW() - INTERVAL '1 day' * ?", clientID, days).
		Count(&count).Error
	return count, err
}

// GetLastByClient - Obtiene la última interacción de un cliente
func (r *InteractionRepository) GetLastByClient(clientID uint) (*models.Interaction, error) {
	var interaction models.Interaction
	err := r.db.Where("client_id = ?", clientID).
		Order("created_at DESC").
		First(&interaction).Error
	if err != nil {
		return nil, err
	}
	return &interaction, nil
}
