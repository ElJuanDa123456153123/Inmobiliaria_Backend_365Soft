package models

import (
	"time"

	"gorm.io/gorm"
)

// Notification - Modelo para notificaciones del sistema sobre búsquedas guardadas
type Notification struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID        uint         `gorm:"not null;index" json:"user_id"`
	User          User         `gorm:"foreignKey:UserID" json:"-"`
	SavedSearchID *uint        `gorm:"index" json:"saved_search_id,omitempty"`
	SavedSearch   *SavedSearch `gorm:"foreignKey:SavedSearchID" json:"saved_search,omitempty"`
	PropertyID    uint         `gorm:"not null;index" json:"property_id"`
	Property      Property     `gorm:"foreignKey:PropertyID" json:"property"`
	Mensaje       string       `gorm:"size:500;not null" json:"mensaje"`
	IsRead        bool         `gorm:"default:false;not null" json:"is_read"`
}

func (Notification) TableName() string {
	return "notificaciones"
}
