package models

import (
	"time"

	"gorm.io/gorm"
)

// Message - Modelo de mensajería entre usuarios
// Permite la comunicación directa entre usuarios del sistema
type Message struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Contexto opcional: propiedad relacionada
	PropertyID *uint     `gorm:"index" json:"property_id,omitempty"`
	Property   *Property `gorm:"foreignKey:PropertyID" json:"property,omitempty"`

	// Usuario que envía el mensaje
	DeUserID uint `gorm:"not null;index" json:"de_user_id"`
	DeUser   User `gorm:"foreignKey:DeUserID" json:"de_user,omitempty"`

	// Usuario que recibe el mensaje
	ParaUserID uint `gorm:"not null;index" json:"para_user_id"`
	ParaUser   User `gorm:"foreignKey:ParaUserID" json:"para_user,omitempty"`

	// Contenido del mensaje
	Mensaje string `gorm:"type:text;not null" json:"mensaje"`

	// Tipo de mensaje: "texto", "imagen", "documento"
	TipoMensaje string `gorm:"type:varchar(20);default:texto" json:"tipo_mensaje"`

	// Archivo adjunto (URL y nombre original)
	ArchivoURL    *string `gorm:"type:varchar(500)" json:"archivo_url,omitempty"`
	ArchivoNombre *string `gorm:"type:varchar(255)" json:"archivo_nombre,omitempty"`

	// Estado de lectura
	Leido bool `gorm:"default:false" json:"leido"`
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (Message) TableName() string {
	return "messages"
}
