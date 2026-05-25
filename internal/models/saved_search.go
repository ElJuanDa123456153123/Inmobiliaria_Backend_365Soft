package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SavedSearch - Modelo para búsquedas guardadas de filtros de propiedades
type SavedSearch struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID   uint           `gorm:"not null;index" json:"user_id"`
	User     User           `gorm:"foreignKey:UserID" json:"-"`
	Nombre   string         `gorm:"size:200;not null" json:"nombre"`
	Filtros  datatypes.JSON `gorm:"type:jsonb;not null" json:"filtros"` // {tipo, zona, precio_min, precio_max, habitaciones}
	IsActive bool           `gorm:"default:true;not null" json:"is_active"`
}

func (SavedSearch) TableName() string {
	return "busquedas_guardadas"
}
