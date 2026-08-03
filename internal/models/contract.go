package models

import (
	"time"

	"gorm.io/gorm"
)

// ContractStatus - Estados de un contrato de alquiler
type ContractStatus string

const (
	ContractStatusActivo    ContractStatus = "activo"     // Contrato vigente
	ContractStatusVencido   ContractStatus = "vencido"    // Contrato vencido
	ContractStatusTerminado ContractStatus = "terminado"  // Contrato terminado anticipadamente
)

// RentalContract - Modelo de contratos de alquiler
type RentalContract struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Propiedad en alquiler
	PropertyID uint `gorm:"not null;index" json:"property_id"`
	Property   Property `gorm:"foreignKey:PropertyID" json:"property,omitempty"`

	// Quién alquila (inquilino)
	InquilinoID uint `gorm:"not null;index" json:"inquilino_id"`
	Inquilino   User `gorm:"foreignKey:InquilinoID" json:"inquilino,omitempty"`

	// Dueño de la propiedad (vendedor/arrendador)
	VendedorID uint `gorm:"not null;index" json:"vendedor_id"`
	Vendedor   User `gorm:"foreignKey:VendedorID" json:"vendedor,omitempty"`

	// Fechas del contrato
	FechaInicio time.Time `gorm:"not null" json:"fecha_inicio"`
	FechaFin    time.Time `gorm:"not null" json:"fecha_fin"`

	// Monto mensual de alquiler
	MontoMensual float64 `gorm:"type:decimal(15,2);not null" json:"monto_mensual"`

	// Estado del contrato
	Estado ContractStatus `gorm:"size:20;not null;default:'activo'" json:"estado"`

	// URL del documento del contrato (PDF)
	DocumentoURL string `gorm:"type:text" json:"documento_url"`
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (RentalContract) TableName() string {
	return "rental_contracts"
}
