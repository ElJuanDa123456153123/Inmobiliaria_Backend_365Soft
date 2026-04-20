package repository

import (
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
)

// AuditRepository - Repositorio para logs de auditoría
type AuditRepository struct {
	DB *config.Database
}

// NewAuditRepository - Crea un nuevo repositorio de auditoría
func NewAuditRepository(db *config.Database) *AuditRepository {
	return &AuditRepository{DB: db}
}

// Create - Crea un nuevo log de auditoría
func (r *AuditRepository) Create(auditLog *models.AuditLog) error {
	return r.DB.DB.Create(auditLog).Error
}

// CreateAsync - Crea un log de auditoría de forma asíncrona (sin error handling)
func (r *AuditRepository) CreateAsync(auditLog *models.AuditLog) {
	go func() {
		_ = r.DB.DB.Create(auditLog).Error
	}()
}

// GetByID - Obtiene un log de auditoría por ID
func (r *AuditRepository) GetByID(id uint) (*models.AuditLog, error) {
	var auditLog models.AuditLog
	err := r.DB.DB.First(&auditLog, id).Error
	if err != nil {
		return nil, err
	}
	return &auditLog, nil
}

// GetByEntityID - Obtiene logs de auditoría por entidad y ID
func (r *AuditRepository) GetByEntityID(entity string, entityID uint, limit int) ([]models.AuditLog, error) {
	var auditLogs []models.AuditLog
	err := r.DB.DB.
		Where("entity = ? AND entity_id = ?", entity, entityID).
		Order("created_at DESC").
		Limit(limit).
		Find(&auditLogs).Error
	if err != nil {
		return nil, err
	}
	return auditLogs, nil
}

// GetByUserID - Obtiene logs de auditoría por usuario
func (r *AuditRepository) GetByUserID(userID uint, limit int) ([]models.AuditLog, error) {
	var auditLogs []models.AuditLog
	err := r.DB.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&auditLogs).Error
	if err != nil {
		return nil, err
	}
	return auditLogs, nil
}

// GetByAction - Obtiene logs de auditoría por acción
func (r *AuditRepository) GetByAction(action string, limit int) ([]models.AuditLog, error) {
	var auditLogs []models.AuditLog
	err := r.DB.DB.
		Where("action = ?", action).
		Order("created_at DESC").
		Limit(limit).
		Find(&auditLogs).Error
	if err != nil {
		return nil, err
	}
	return auditLogs, nil
}
