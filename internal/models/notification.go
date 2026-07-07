package models

import (
	"time"

	"gorm.io/gorm"
)

// Notification - Modelo para notificaciones del sistema
type Notification struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID        uint         `gorm:"not null;index" json:"user_id"`
	User          User         `gorm:"foreignKey:UserID" json:"-"`
	SavedSearchID *uint        `gorm:"index" json:"saved_search_id,omitempty"`
	SavedSearch   *SavedSearch `gorm:"foreignKey:SavedSearchID" json:"saved_search,omitempty"`
	PropertyID    *uint        `gorm:"index" json:"property_id,omitempty"`
	Property      Property     `gorm:"foreignKey:PropertyID" json:"property,omitempty"`

	// Campos nuevos para soportar todos los tipos de notificaciones
	Tipo         string `gorm:"size:50;not null" json:"tipo"` // propiedad_aprobada, pago_confirmado, etc.
	Titulo       string `gorm:"size:200" json:"titulo"`       // Título opcional de la notificación
	Mensaje      string `gorm:"size:500;not null" json:"mensaje"`
	IsRead       bool   `gorm:"default:false;not null" json:"is_read"`

	// Metadata adicional en formato JSON (opcional)
	Metadata string `gorm:"type:text" json:"metadata,omitempty"`
}

func (Notification) TableName() string {
	return "notificaciones"
}
