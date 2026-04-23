package models

import (
	"time"

	"gorm.io/gorm"
)

// Advisor - Modelo de asesor inmobiliario
type Advisor struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Información del asesor
	Nombre       string  `gorm:"size:150;not null" json:"nombre"`
	Email        string  `gorm:"size:150;uniqueIndex;not null" json:"email"`
	Telefono     string  `gorm:"size:30" json:"telefono"`
	Especialidad string  `gorm:"size:100" json:"especialidad"` // Ej: "Lujo", "Venta", "Alquiler"

	// Estado
	Activo bool `gorm:"default:true" json:"activo"`

	// Relaciones
	Propiedades []Property `gorm:"foreignKey:AdvisorID" json:"propiedades,omitempty"`
}

// TableName especifica el nombre de la tabla en la base de datos
func (Advisor) TableName() string {
	return "asesores"
}
