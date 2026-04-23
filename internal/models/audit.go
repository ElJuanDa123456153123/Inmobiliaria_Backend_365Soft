package models

import (
	"time"
)

// AuditLog - Log de auditoría para acciones importantes
type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    *uint     `gorm:"index" json:"user_id,omitempty"`    // NULL para acciones anónimas
	Action    string    `gorm:"size:50;index" json:"action"`         // Ej: create_property, update_property, delete_property
	Entity    string    `gorm:"size:50;index" json:"entity"`         // Ej: property, user, image
	EntityID  *uint     `gorm:"index" json:"entity_id,omitempty"`   // ID de la entidad afectada
	Details   string    `gorm:"type:text" json:"details,omitempty"` // JSON con detalles adicionales
	IPAddress string    `gorm:"size:45" json:"ip_address,omitempty"` // IP del cliente
	UserAgent string    `gorm:"size:255" json:"user_agent,omitempty"` // User-Agent del cliente
	CreatedAt time.Time `json:"created_at"`
}

// TableName - Especifica el nombre de la tabla
func (AuditLog) TableName() string {
	return "audit_logs"
}
