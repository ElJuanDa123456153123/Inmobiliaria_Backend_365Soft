package repository

import (
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
)

// AdvisorRepository - Repositorio para operaciones de base de datos de asesores
type AdvisorRepository struct {
	DB *config.Database
}

// NewAdvisorRepository - Crea un nuevo repositorio de asesores
func NewAdvisorRepository(db *config.Database) *AdvisorRepository {
	return &AdvisorRepository{DB: db}
}

// GetAllActive - Obtiene todos los asesores activos
func (r *AdvisorRepository) GetAllActive() ([]models.Advisor, error) {
	var asesores []models.Advisor
	err := r.DB.DB.Where("activo = ?", true).Find(&asesores).Error
	if err != nil {
		return nil, err
	}
	return asesores, nil
}

// CountActivePropertiesByAdvisorID - Cuenta las propiedades activas de un asesor
func (r *AdvisorRepository) CountActivePropertiesByAdvisorID(advisorID uint) (int64, error) {
	var count int64
	err := r.DB.DB.Model(&models.Property{}).
		Where("advisor_id = ? AND estado_propiedad = ?", advisorID, models.PropertyStatusActivo).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetByID - Obtiene un asesor por ID
func (r *AdvisorRepository) GetByID(id uint) (*models.Advisor, error) {
	var advisor models.Advisor
	err := r.DB.DB.First(&advisor, id).Error
	if err != nil {
		return nil, err
	}
	return &advisor, nil
}

// GetPropertiesByAdvisorID - Obtiene todas las propiedades de un asesor
func (r *AdvisorRepository) GetPropertiesByAdvisorID(advisorID uint, status models.PropertyStatus) ([]models.Property, error) {
	var properties []models.Property
	query := r.DB.DB.Where("advisor_id = ?", advisorID)

	if status != "" {
		query = query.Where("estado_propiedad = ?", status)
	}

	err := query.Order("created_at DESC").Find(&properties).Error
	if err != nil {
		return nil, err
	}
	return properties, nil
}
