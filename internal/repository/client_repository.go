package repository

import (
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"gorm.io/gorm"
)

// ClientRepositoryInterface - Interface del repositorio de clientes
type ClientRepositoryInterface interface {
	// CRUD básico
	Create(client *models.Client) error
	GetByID(id uint) (*models.Client, error)
	Update(client *models.Client) error
	Delete(id uint) error
	GetAll() ([]models.Client, error)

	// Listados con filtros
	GetByAdvisorID(advisorID uint) ([]models.Client, error)
	GetByAdvisorIDWithPagination(advisorID uint, page, limit int) ([]models.Client, int64, error)
	GetByStatus(advisorID uint, status models.ClientStatus) ([]models.Client, error)
	SearchByName(advisorID uint, query string) ([]models.Client, error)

	// Estadísticas
	CountByStatus(advisorID uint, status models.ClientStatus) (int64, error)
	GetFunnelStats(advisorID uint) (*FunnelStats, error)
	GetUpcomingContacts(advisorID uint, days int) ([]models.Client, error)

	// Operaciones complejas
	GetWithInteractions(clientID uint) (*models.Client, error)
	GetTimeline(clientID uint) ([]models.Interaction, error)
}

// ClientRepository - Implementación del repositorio
type ClientRepository struct {
	db *gorm.DB
}

// NewClientRepository - Crea una nueva instancia del repositorio
func NewClientRepository(db *gorm.DB) ClientRepositoryInterface {
	return &ClientRepository{db: db}
}

// Create - Crea un nuevo cliente
func (r *ClientRepository) Create(client *models.Client) error {
	return r.db.Create(client).Error
}

// GetByID - Obtiene un cliente por ID
func (r *ClientRepository) GetByID(id uint) (*models.Client, error) {
	var client models.Client
	err := r.db.Where("id = ?", id).First(&client).Error
	if err != nil {
		return nil, err
	}
	return &client, nil
}

// Update - Actualiza un cliente
func (r *ClientRepository) Update(client *models.Client) error {
	return r.db.Save(client).Error
}

// Delete - Elimina un cliente (soft delete)
func (r *ClientRepository) Delete(id uint) error {
	return r.db.Delete(&models.Client{}, id).Error
}

// GetAll - Obtiene todos los clientes
func (r *ClientRepository) GetAll() ([]models.Client, error) {
	var clients []models.Client
	err := r.db.Order("created_at DESC").Find(&clients).Error
	return clients, err
}

// GetByAdvisorID - Obtiene todos los clientes de un asesor
func (r *ClientRepository) GetByAdvisorID(advisorID uint) ([]models.Client, error) {
	var clients []models.Client
	err := r.db.Where("advisor_id = ?", advisorID).
		Preload("Property").
		Order("created_at DESC").
		Find(&clients).Error
	return clients, err
}

// GetByAdvisorIDWithPagination - Obtiene clientes paginados de un asesor
func (r *ClientRepository) GetByAdvisorIDWithPagination(advisorID uint, page, limit int) ([]models.Client, int64, error) {
	var clients []models.Client
	var total int64

	offset := (page - 1) * limit

	// Contar total
	if err := r.db.Model(&models.Client{}).Where("advisor_id = ?", advisorID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Obtener clientes
	err := r.db.Where("advisor_id = ?", advisorID).
		Preload("Property").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&clients).Error

	return clients, total, err
}

// GetByStatus - Obtiene clientes por estado
func (r *ClientRepository) GetByStatus(advisorID uint, status models.ClientStatus) ([]models.Client, error) {
	var clients []models.Client
	err := r.db.Where("advisor_id = ? AND estado = ?", advisorID, status).
		Preload("Property").
		Order("created_at DESC").
		Find(&clients).Error
	return clients, err
}

// SearchByName - Busca clientes por nombre
func (r *ClientRepository) SearchByName(advisorID uint, query string) ([]models.Client, error) {
	var clients []models.Client
	err := r.db.Where("advisor_id = ? AND nombre ILIKE ?", advisorID, "%"+query+"%").
		Preload("Property").
		Order("created_at DESC").
		Find(&clients).Error
	return clients, err
}

// CountByStatus - Cuenta clientes por estado
func (r *ClientRepository) CountByStatus(advisorID uint, status models.ClientStatus) (int64, error) {
	var count int64
	err := r.db.Model(&models.Client{}).
		Where("advisor_id = ? AND estado = ?", advisorID, status).
		Count(&count).Error
	return count, err
}

// FunnelStats - Estadísticas del embudo de ventas
type FunnelStats struct {
	Nuevos         int64 `json:"nuevos"`
	Contactados    int64 `json:"contactados"`
	Interesados    int64 `json:"interesados"`
	EnNegociacion  int64 `json:"en_negociacion"`
	Cerrados       int64 `json:"cerrados"`
	Perdidos       int64 `json:"perdidos"`
	Total          int64 `json:"total"`
}

// GetFunnelStats - Obtiene las estadísticas del embudo de un asesor
func (r *ClientRepository) GetFunnelStats(advisorID uint) (*FunnelStats, error) {
	stats := &FunnelStats{}

	// Contar por cada estado
	r.db.Model(&models.Client{}).
		Where("advisor_id = ? AND estado = ?", advisorID, models.ClientStatusNuevo).
		Count(&stats.Nuevos)

	r.db.Model(&models.Client{}).
		Where("advisor_id = ? AND estado = ?", advisorID, models.ClientStatusContactado).
		Count(&stats.Contactados)

	r.db.Model(&models.Client{}).
		Where("advisor_id = ? AND estado = ?", advisorID, models.ClientStatusInteresado).
		Count(&stats.Interesados)

	r.db.Model(&models.Client{}).
		Where("advisor_id = ? AND estado = ?", advisorID, models.ClientStatusEnNegociacion).
		Count(&stats.EnNegociacion)

	r.db.Model(&models.Client{}).
		Where("advisor_id = ? AND estado = ?", advisorID, models.ClientStatusCerrado).
		Count(&stats.Cerrados)

	r.db.Model(&models.Client{}).
		Where("advisor_id = ? AND estado = ?", advisorID, models.ClientStatusPerdido).
		Count(&stats.Perdidos)

	// Total
	stats.Total = stats.Nuevos + stats.Contactados + stats.Interesados +
		stats.EnNegociacion + stats.Cerrados + stats.Perdidos

	return stats, nil
}

// GetUpcomingContacts - Obtiene clientes con próximo contacto pronto
func (r *ClientRepository) GetUpcomingContacts(advisorID uint, days int) ([]models.Client, error) {
	var clients []models.Client

	// Obtener clientes cuyo próximo contacto es en los próximos X días
	err := r.db.Where("advisor_id = ? AND proximo_contacto IS NOT NULL AND proximo_contacto <= NOW() + INTERVAL '1 day' * ?",
		advisorID, days).
		Preload("Property").
		Order("proximo_contacto ASC").
		Find(&clients).Error

	return clients, err
}

// GetWithInteractions - Obtiene un cliente con todas sus interacciones
func (r *ClientRepository) GetWithInteractions(clientID uint) (*models.Client, error) {
	var client models.Client
	err := r.db.Where("id = ?", clientID).
		Preload("Property").
		Preload("Interactions", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		}).
		First(&client).Error
	if err != nil {
		return nil, err
	}
	return &client, nil
}

// GetTimeline - Obtiene el timeline de interacciones de un cliente
func (r *ClientRepository) GetTimeline(clientID uint) ([]models.Interaction, error) {
	var interactions []models.Interaction
	err := r.db.Where("client_id = ?", clientID).
		Order("created_at DESC").
		Find(&interactions).Error
	return interactions, err
}
