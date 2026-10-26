package models

import (
	"time"

	"gorm.io/gorm"
)

// Habitacion - Unidad rentable dentro de una propiedad que se alquila por
// habitaciones (ej: habitaciones de un condominio). Cada una puede tener su
// propio precio; si es null hereda el precio de la propiedad.
type Habitacion struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Propiedad a la que pertenece
	PropertyID uint `gorm:"not null;index" json:"property_id"`
	Property   Property `gorm:"foreignKey:PropertyID" json:"-"`

	// Datos de la habitación
	Nombre      string  `gorm:"size:100;not null" json:"nombre"`                  // ej: "Hab. 1 - baño privado"
	Descripcion string  `gorm:"type:text" json:"descripcion,omitempty"`           // amoblado, closet, etc.
	Precio      *float64 `gorm:"type:decimal(15,2)" json:"precio,omitempty"`       // null = hereda precio de la propiedad
	Activa      bool    `gorm:"not null;default:true" json:"activa"`              // false = en mantenimiento, no se alquila

	// Contrato activo que la ocupa (calculado, no almacenado)
	ContratoActivoID *uint `gorm:"-" json:"contrato_activo_id,omitempty"`
	Disponible       bool  `gorm:"-" json:"disponible"`
}

// TableName - Nombre de la tabla en PostgreSQL
func (Habitacion) TableName() string {
	return "property_habitaciones"
}
