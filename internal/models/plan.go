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

	Nombre                string  `gorm:"size:100;not null" json:"nombre"`
	Precio                float64 `gorm:"not null" json:"precio"`
	Descripcion           string  `gorm:"type:text" json:"descripcion"`
	DuracionMeses         int     `gorm:"default:1" json:"duracion_meses"` // Vigencia de la suscripción en meses
	LimitePropiedades     int     `gorm:"default:5" json:"limite_propiedades"`
	DuracionPropiedadMinutos int  `gorm:"default:43200" json:"duracion_propiedad_minutos"` // Duración de cada propiedad en minutos (43200 = 30 días por defecto)
	Caracteristicas       string  `gorm:"type:text" json:"caracteristicas"` // Almacenado como JSON string o texto separado
	MetodoPago            string  `gorm:"size:20;default:'QR'" json:"metodo_pago"` // QR, TARJETA, TRANSFERENCIA
	Color                 string  `gorm:"size:50;default:'from-gray-500 to-gray-600'" json:"color"` // Gradiente de color para el header
	Destacado             bool    `gorm:"default:false" json:"destacado"`
	Activo                bool    `gorm:"default:true" json:"activo"`
	Orden                 int     `gorm:"default:0" json:"orden"`
	PermitirRepublicacion bool    `gorm:"default:false" json:"permitir_republicacion"` // ¿Al eliminar permite republicar?
}

// TableName - Nombre de la tabla
func (Plan) TableName() string {
	return "planes"
}
