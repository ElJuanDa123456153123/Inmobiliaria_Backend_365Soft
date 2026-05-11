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

	// Foto de perfil del asesor (URL o base64)
	ImagenURL string `gorm:"type:text" json:"imagen_url,omitempty"` // Campo antiguo, mantener por compatibilidad
	AvatarURL string `gorm:"type:text" json:"avatar_url,omitempty"` // Nuevo campo de avatar

	// Usuario asociado (para login)
	UserID *uint `gorm:"uniqueIndex" json:"user_id,omitempty"`
	User   *User `gorm:"foreignKey:UserID" json:"user,omitempty"`

	// Estado
	Activo bool `gorm:"default:true" json:"activo"`

	// Relaciones
	// Nota: AdvisorID se mantiene por compatibilidad, pero ahora también soportamos múltiples asesores
	Propiedades []Property `gorm:"foreignKey:AdvisorID" json:"propiedades,omitempty"`
}

// TableName especifica el nombre de la tabla en la base de datos
func (Advisor) TableName() string {
	return "asesores"
}
