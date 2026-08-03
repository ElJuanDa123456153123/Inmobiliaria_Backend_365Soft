package models

import (
	"time"

	"gorm.io/gorm"
)

// VisitStatus - Estados de una visita
type VisitStatus string

const (
	VisitStatusPendiente   VisitStatus = "pendiente"   // Visita agendada
	VisitStatusCompletada  VisitStatus = "completada"  // Visita realizada
	VisitStatusCancelada   VisitStatus = "cancelada"   // Visita cancelada
)

// Visit - Modelo de visitas agendadas a propiedades
type Visit struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Propiedad a visitar
	PropertyID uint `gorm:"not null;index" json:"property_id"`
	Property   Property `gorm:"foreignKey:PropertyID" json:"property,omitempty"`

	// Usuario interesado (inquilino/comprador)
	InquilinoID uint `gorm:"not null;index" json:"inquilino_id"`
	Inquilino   User `gorm:"foreignKey:InquilinoID" json:"inquilino,omitempty"`

	// Dueño de la propiedad (vendedor)
	VendedorID uint `gorm:"not null;index" json:"vendedor_id"`
	Vendedor   User `gorm:"foreignKey:VendedorID" json:"vendedor,omitempty"`

	// Fecha y hora de la visita
	FechaVisita time.Time `gorm:"not null;index" json:"fecha_visita"`

	// Estado de la visita
	Estado VisitStatus `gorm:"size:20;not null;default:'pendiente'" json:"estado"`

	// Notas adicionales
	Notas string `gorm:"type:text" json:"notas"`
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (Visit) TableName() string {
	return "visits"
}
