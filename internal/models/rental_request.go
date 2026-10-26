package models

import (
	"time"

	"gorm.io/gorm"
)

// RentalRequestStatus - Estados de una solicitud de alquiler
type RentalRequestStatus string

const (
	RentalRequestStatusPendiente RentalRequestStatus = "pendiente"
	RentalRequestStatusAceptada  RentalRequestStatus = "aceptada"
	RentalRequestStatusRechazada RentalRequestStatus = "rechazada"
	RentalRequestStatusCancelada RentalRequestStatus = "cancelada"
)

// RentalRequest - Solicitud de alquiler que envía un interesado sobre una propiedad.
// Al ser aceptada por el propietario, genera un RentalContract con sus cuotas.
type RentalRequest struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Propiedad solicitada
	PropertyID uint     `gorm:"not null;index" json:"property_id"`
	Property   Property `gorm:"foreignKey:PropertyID" json:"property,omitempty"`

	// Habitación elegida (cuando la propiedad se alquila por habitaciones)
	HabitacionID *uint      `gorm:"default:null;index" json:"habitacion_id,omitempty"`
	Habitacion   *Habitacion `gorm:"foreignKey:HabitacionID" json:"habitacion,omitempty"`

	// Interesado que envía la solicitud
	InquilinoID uint `gorm:"not null;index" json:"inquilino_id"`
	Inquilino   User `gorm:"foreignKey:InquilinoID" json:"inquilino,omitempty"`

	// Dueño de la propiedad (denormalizado para consultas rápidas)
	VendedorID uint `gorm:"not null;index" json:"vendedor_id"`
	Vendedor   User `gorm:"foreignKey:VendedorID" json:"vendedor,omitempty"`

	// Datos de contacto opcionales del interesado
	Mensaje          string `gorm:"type:text" json:"mensaje,omitempty"`
	TelefonoContacto string `gorm:"size:30" json:"telefono_contacto,omitempty"`

	// Estado de la solicitud
	Estado RentalRequestStatus `gorm:"size:20;not null;default:'pendiente';index" json:"estado"`

	// Contrato generado cuando la solicitud es aceptada
	ContractID *uint           `gorm:"default:null" json:"contract_id,omitempty"`
	Contract   *RentalContract `gorm:"foreignKey:ContractID" json:"contract,omitempty"`
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (RentalRequest) TableName() string {
	return "solicitudes_alquiler"
}
