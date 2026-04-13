package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// JSON - Custom type for JSONB fields
type JSON json.RawMessage

// Scan - Implement sql.Scanner interface
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	*j = JSON(bytes)
	return nil
}

// Value - Implement driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.RawMessage(j).MarshalJSON()
}

// MarshalJSON - Implement json.Marshaler interface
func (j JSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return json.RawMessage(j).MarshalJSON()
}

// UnmarshalJSON - Implement json.Unmarshaler interface
func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return nil
	}
	*j = JSON(data)
	return nil
}

// AuditAction - Tipos de acciones de auditoría
type AuditAction string

const (
	ActionCreate   AuditAction = "crear"
	ActionUpdate   AuditAction = "actualizar"
	ActionDelete   AuditAction = "eliminar"
	ActionLogin    AuditAction = "login"
	ActionLogout   AuditAction = "logout"
	ActionApprove  AuditAction = "aprobar"
	ActionReject   AuditAction = "rechazar"
	ActionView     AuditAction = "ver"
	ActionExport   AuditAction = "exportar"
)

// AuditEntity - Entidades del sistema
type AuditEntity string

const (
	EntityUser      AuditEntity = "usuario"
	EntityProperty  AuditEntity = "propiedad"
	EntityPhoto     AuditEntity = "foto"
	EntityCategory  AuditEntity = "categoria"
	EntityClient    AuditEntity = "cliente"
	EntityPayment   AuditEntity = "pago"
	EntityPublication AuditEntity = "publicacion"
	EntityAdmin     AuditEntity = "admin"
)

// AuditLog - Registro de auditoría de todas las acciones
type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `gorm:"index;not null" json:"created_at"`

	// Quién realizó la acción
	UserID     *uint   `gorm:"index" json:"user_id,omitempty"`
	User       *User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	IPAddress  string  `gorm:"size:50" json:"ip_address,omitempty"`
	UserAgent  string  `gorm:"size:500" json:"user_agent,omitempty"`

	// Acción realizada
	Accion  AuditAction  `gorm:"size:100;not null;index" json:"accion"`
	Entidad AuditEntity  `gorm:"size:100;not null;index" json:"entidad"`
	EntityID *uint       `gorm:"index" json:"entity_id,omitempty"`

	// Cambios (antes y después)
	CambiosAnteriores JSON `gorm:"type:jsonb" json:"cambios_anteriores,omitempty"`
	CambiosNuevos     JSON `gorm:"type:jsonb" json:"cambios_nuevos,omitempty"`

	// Contexto adicional
	Metadata JSON `gorm:"type:jsonb" json:"metadata,omitempty"`

	// Request tracking
	RequestID string `gorm:"size:100;index" json:"request_id,omitempty"`
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (AuditLog) TableName() string {
	return "audit_logs"
}

// BeforeCreate - Hook antes de crear
func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	return nil
}
