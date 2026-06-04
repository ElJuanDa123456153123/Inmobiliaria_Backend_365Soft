package models

import (
	"time"

	"gorm.io/gorm"
)

// QrPago - Modelo de pagos con QR del Banco MC4
type QrPago struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Información del pago
	Alias string `gorm:"size:100;uniqueIndex;not null" json:"alias"` // Alias único del Banco MC4

	// Estados: PENDIENTE, PAGADO, CANCELADO, EXPIRADO
	Estado string `gorm:"size:20;not null;default:'PENDIENTE';index" json:"estado"`

	// Monto y moneda
	Monto   float64 `gorm:"not null" json:"monto"`
	Moneda string  `gorm:"size:10;not null;default:'BOB'" json:"moneda"`

	// Relaciones
	PropertyID *uint `gorm:"default:null" json:"property_id"` // Puede ser null si se compra el plan antes de tener la propiedad
	Property   *Property `gorm:"foreignKey:PropertyID" json:"property,omitempty"`

	UserID uint `gorm:"not null;index" json:"user_id"`
	User   *User `gorm:"foreignKey:UserID" json:"user,omitempty"`

	// Plan de publicación seleccionado
	PlanID uint `gorm:"not null" json:"plan_id"`
	Plan   *Plan `gorm:"foreignKey:PlanID" json:"plan,omitempty"`

	// Fechas
	FechaVencimiento time.Time  `gorm:"not null" json:"fecha_vencimiento"` // Cuándo expira el QR (15 minutos)
	FechaProceso     *time.Time `json:"fecha_proceso"`                     // Cuándo se procesó el pago

	// Información del QR
	ImagenQr    string `gorm:"type:text" json:"imagen_qr,omitempty"`    // Base64 de la imagen del QR
	DetalleGlosa string `gorm:"type:text" json:"detalle_glosa"`         // Descripción del pago

	// Información del cliente (del webhook del banco)
	NumeroOrden       string `gorm:"size:100" json:"numero_orden,omitempty"`
	NombreCliente     string `gorm:"size:200" json:"nombre_cliente,omitempty"`
	DocumentoCliente  string `gorm:"size:50" json:"documento_cliente,omitempty"`
	CuentaCliente     string `gorm:"size:100" json:"cuenta_cliente,omitempty"`
}

// TableName especifica el nombre de la tabla en la base de datos
func (QrPago) TableName() string {
	return "qr_pagos"
}
