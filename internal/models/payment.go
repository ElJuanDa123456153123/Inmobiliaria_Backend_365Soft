package models

import (
	"time"

	"gorm.io/gorm"
)

// Payment - Modelo para registro de pagos
type Payment struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID uint `gorm:"not null;index" json:"user_id"`
	User   User `gorm:"foreignKey:UserID" json:"user,omitempty"`

	Monto  float64 `gorm:"type:decimal(10,2);not null" json:"monto"`
	Moneda string  `gorm:"size:10;not null;default:'BOB'" json:"moneda"` // BOB, USD
	Metodo string  `gorm:"size:50;not null" json:"metodo"`              // qr_bisa, transferencia, tarjeta
	Estado string  `gorm:"size:50;not null;default:'pendiente'" json:"estado"` // pendiente, completado, fallido, expirado, reembolsado

	ReferenciaPago string     `gorm:"size:100;uniqueIndex" json:"referencia_pago,omitempty"`
	FechaPago      *time.Time `json:"fecha_pago,omitempty"`
	Notas          string     `gorm:"type:text" json:"notas,omitempty"`
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (Payment) TableName() string {
	return "pagos"
}
