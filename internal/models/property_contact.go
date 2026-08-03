package models

import (
	"time"

	"gorm.io/gorm"
)

// PropertyContact - Modelo de interesados en propiedades
// Representa cuando un usuario (comprador) muestra interés en una propiedad
type PropertyContact struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Propiedad de interés
	PropertyID uint `gorm:"not null;index" json:"property_id"`
	Property   Property `gorm:"foreignKey:PropertyID" json:"property,omitempty"`

	// Usuario interesado (comprador)
	UserID uint `gorm:"not null;index" json:"user_id"`
	User   User `gorm:"foreignKey:UserID" json:"user,omitempty"`

	// Dueño de la propiedad (vendedor)
	VendedorID uint `gorm:"not null;index" json:"vendedor_id"`
	Vendedor   User `gorm:"foreignKey:VendedorID" json:"vendedor,omitempty"`

	// Mensaje del interesado
	Mensaje string `gorm:"type:text" json:"mensaje"`

	// Estado de lectura y respuesta
	Leido      bool `gorm:"default:false" json:"leido"`
	Respondido bool `gorm:"default:false" json:"respondido"`
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (PropertyContact) TableName() string {
	return "property_contacts"
}
