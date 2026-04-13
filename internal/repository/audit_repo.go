package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"gorm.io/gorm"
)

// AuditRepository - Interface para repositorio de auditoría
type AuditRepository interface {
	// Log de acciones
	LogCreate(ctx context.Context, userID uint, entidad string, entityID uint, data interface{}) error
	LogUpdate(ctx context.Context, userID uint, entidad string, entityID uint, before, after interface{}) error
	LogDelete(ctx context.Context, userID uint, entidad string, entityID uint, data interface{}) error
	LogAction(ctx context.Context, entry *AuditLogEntry) error

	// Consultas
	GetByUserID(ctx context.Context, userID uint, page, limit int) ([]*models.AuditLog, int64, error)
	GetByEntity(ctx context.Context, entidad string, entityID uint, page, limit int) ([]*models.AuditLog, int64, error)
	GetByAction(ctx context.Context, accion models.AuditAction, page, limit int) ([]*models.AuditLog, int64, error)
	GetByRequestID(ctx context.Context, requestID string) ([]*models.AuditLog, error)

	// Estadísticas
	GetActionStats(ctx context.Context, userID uint, startDate, endDate time.Time) (*ActionStats, error)
}

// AuditLogEntry - Entrada de log de auditoría
type AuditLogEntry struct {
	UserID    *uint
	IPAddress string
	UserAgent string
	Accion    models.AuditAction
	Entidad   models.AuditEntity
	EntityID  *uint
	Before    interface{}
	After     interface{}
	Metadata  map[string]interface{}
	RequestID string
}

// ActionStats - Estadísticas de acciones
type ActionStats struct {
	Creations    int64 `json:"creations"`
	Updates      int64 `json:"updates"`
	Deletes      int64 `json:"deletes"`
	Logins       int64 `json:"logins"`
	TotalActions int64 `json:"total_actions"`
}

// auditRepository - Implementación del repositorio
type auditRepository struct {
	db *gorm.DB
}

// NewAuditRepository - Crea una nueva instancia del repositorio
func NewAuditRepository(db *gorm.DB) AuditRepository {
	return &auditRepository{
		db: db,
	}
}

// ========== LOG DE ACCIONES ==========

// LogCreate - Registra creación de entidad
func (r *auditRepository) LogCreate(ctx context.Context, userID uint, entidad string, entityID uint, data interface{}) error {
	afterJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error serializando datos: %w", err)
	}

	entry := &models.AuditLog{
		UserID:     &userID,
		Accion:     models.ActionCreate,
		Entidad:    models.AuditEntity(entidad),
		EntityID:   &entityID,
		CambiosNuevos: models.JSON(afterJSON),
	}

	return r.db.WithContext(ctx).Create(entry).Error
}

// LogUpdate - Registra actualización de entidad
func (r *auditRepository) LogUpdate(ctx context.Context, userID uint, entidad string, entityID uint, before, after interface{}) error {
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		return fmt.Errorf("error serializando datos anteriores: %w", err)
	}

	afterJSON, err := json.Marshal(after)
	if err != nil {
		return fmt.Errorf("error serializando nuevos datos: %w", err)
	}

	entry := &models.AuditLog{
		UserID:         &userID,
		Accion:         models.ActionUpdate,
		Entidad:        models.AuditEntity(entidad),
		EntityID:       &entityID,
		CambiosAnteriores: models.JSON(beforeJSON),
		CambiosNuevos:     models.JSON(afterJSON),
	}

	return r.db.WithContext(ctx).Create(entry).Error
}

// LogDelete - Registra eliminación de entidad
func (r *auditRepository) LogDelete(ctx context.Context, userID uint, entidad string, entityID uint, data interface{}) error {
	beforeJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error serializando datos: %w", err)
	}

	entry := &models.AuditLog{
		UserID:          &userID,
		Accion:          models.ActionDelete,
		Entidad:         models.AuditEntity(entidad),
		EntityID:        &entityID,
		CambiosAnteriores: models.JSON(beforeJSON),
	}

	return r.db.WithContext(ctx).Create(entry).Error
}

// LogAction - Registra una acción personalizada
func (r *auditRepository) LogAction(ctx context.Context, entry *AuditLogEntry) error {
	var beforeJSON, afterJSON, metadataJSON *json.RawMessage


	if entry.Before != nil {
		beforeData, err := json.Marshal(entry.Before)
		if err != nil {
			return fmt.Errorf("error serializando before: %w", err)
		}
		beforeJSON = (*json.RawMessage)(&beforeData)
	}

	if entry.After != nil {
		afterData, err := json.Marshal(entry.After)
		if err != nil {
			return fmt.Errorf("error serializando after: %w", err)
		}
		afterJSON = (*json.RawMessage)(&afterData)
	}

	if entry.Metadata != nil {
		metadataData, err := json.Marshal(entry.Metadata)
		if err != nil {
			return fmt.Errorf("error serializando metadata: %w", err)
		}
		metadataJSON = (*json.RawMessage)(&metadataData)
	}

	auditLog := &models.AuditLog{
		UserID:            entry.UserID,
		IPAddress:         entry.IPAddress,
		UserAgent:         entry.UserAgent,
		Accion:            entry.Accion,
		Entidad:           entry.Entidad,
		EntityID:          entry.EntityID,
		CambiosAnteriores: models.JSON(*beforeJSON),
		CambiosNuevos:     models.JSON(*afterJSON),
		Metadata:          models.JSON(*metadataJSON),
		RequestID:         entry.RequestID,
	}

	return r.db.WithContext(ctx).Create(auditLog).Error
}

// ========== CONSULTAS ==========

// GetByUserID - Obtiene logs por usuario
func (r *auditRepository) GetByUserID(ctx context.Context, userID uint, page, limit int) ([]*models.AuditLog, int64, error) {
	var logs []*models.AuditLog
	var total int64

	offset := (page - 1) * limit

	// Contar
	if err := r.db.WithContext(ctx).
		Model(&models.AuditLog{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Listar
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetByEntity - Obtiene logs por entidad
func (r *auditRepository) GetByEntity(ctx context.Context, entidad string, entityID uint, page, limit int) ([]*models.AuditLog, int64, error) {
	var logs []*models.AuditLog
	var total int64

	offset := (page - 1) * limit

	// Contar
	if err := r.db.WithContext(ctx).
		Model(&models.AuditLog{}).
		Where("entidad = ? AND entity_id = ?", entidad, entityID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Listar
	if err := r.db.WithContext(ctx).
		Where("entidad = ? AND entity_id = ?", entidad, entityID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetByAction - Obtiene logs por tipo de acción
func (r *auditRepository) GetByAction(ctx context.Context, accion models.AuditAction, page, limit int) ([]*models.AuditLog, int64, error) {
	var logs []*models.AuditLog
	var total int64

	offset := (page - 1) * limit

	// Contar
	if err := r.db.WithContext(ctx).
		Model(&models.AuditLog{}).
		Where("accion = ?", accion).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Listar
	if err := r.db.WithContext(ctx).
		Where("accion = ?", accion).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetByRequestID - Obtiene todos los logs de un request
func (r *auditRepository) GetByRequestID(ctx context.Context, requestID string) ([]*models.AuditLog, error) {
	var logs []*models.AuditLog

	err := r.db.WithContext(ctx).
		Where("request_id = ?", requestID).
		Order("created_at ASC").
		Find(&logs).Error

	if err != nil {
		return nil, fmt.Errorf("error obteniendo logs por request ID: %w", err)
	}

	return logs, nil
}

// ========== ESTADÍSTICAS ==========

// GetActionStats - Obtiene estadísticas de acciones de un usuario
func (r *auditRepository) GetActionStats(ctx context.Context, userID uint, startDate, endDate time.Time) (*ActionStats, error) {
	stats := &ActionStats{}

	// Query base
	query := r.db.WithContext(ctx).
		Model(&models.AuditLog{}).
		Where("user_id = ?", userID).
		Where("created_at >= ? AND created_at <= ?", startDate, endDate)

	// Total de acciones
	if err := query.Count(&stats.TotalActions).Error; err != nil {
		return nil, err
	}

	// Creaciones
	query.Where("accion = ?", models.ActionCreate).Count(&stats.Creations)

	// Actualizaciones
	query.Where("accion = ?", models.ActionUpdate).Count(&stats.Updates)

	// Eliminaciones
	query.Where("accion = ?", models.ActionDelete).Count(&stats.Deletes)

	// Logins
	query.Where("accion = ?", models.ActionLogin).Count(&stats.Logins)

	return stats, nil
}
