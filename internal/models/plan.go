package models

import (
	"time"

	"gorm.io/gorm"
)

// Plan - Modelo para los planes de suscripción
type Plan struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Nombre            string  `gorm:"size:100;not null" json:"nombre"`
	Precio            float64 `gorm:"not null" json:"precio"`
	Descripcion       string  `gorm:"type:text" json:"descripcion"`
	DuracionMeses     int     `gorm:"default:1" json:"duracion_meses"`
	LimitePropiedades int     `gorm:"default:5" json:"limite_propiedades"`
	Caracteristicas   string  `gorm:"type:text" json:"caracteristicas"` // Almacenado como JSON string o texto separado
	Destacado         bool    `gorm:"default:false" json:"destacado"`
	Activo            bool    `gorm:"default:true" json:"activo"`
	Orden             int     `gorm:"default:0" json:"orden"`
}

// TableName - Nombre de la tabla
func (Plan) TableName() string {
	return "planes"
}
