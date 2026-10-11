package models

import (
	"time"

	"gorm.io/gorm"
)

// PaymentStatus - Estados de un pago de alquiler
type PaymentStatus string

const (
	PaymentStatusPendiente PaymentStatus = "pendiente" // Pago pendiente
	PaymentStatusPagado    PaymentStatus = "pagado"    // Pago realizado
	PaymentStatusAtrasado  PaymentStatus = "atrasado"  // Pago atrasado
)

// PaymentType - Tipo de pago de alquiler
type PaymentType string

const (
	PaymentTypeCuota              PaymentType = "cuota"               // Cuota mensual
	PaymentTypeDeposito           PaymentType = "deposito"            // Depósito de garantía
	PaymentTypeDevolucionDeposito PaymentType = "devolucion_deposito" // Devolución del depósito
	PaymentTypeAnticres           PaymentType = "anticres"            // Anticrés (pago único inicial del anticrético)
	PaymentTypeDevolucionAnticres PaymentType = "devolucion_anticres" // Devolución del anticrés al finalizar
)

// RentPayment - Modelo de pagos de alquiler
type RentPayment struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Contrato relacionado
	ContractID uint `gorm:"not null;index" json:"contract_id"`
	Contract   RentalContract `gorm:"foreignKey:ContractID" json:"contract,omitempty"`

	// Inquilino que realiza el pago
	InquilinoID uint `gorm:"not null;index" json:"inquilino_id"`
	Inquilino   User `gorm:"foreignKey:InquilinoID" json:"inquilino,omitempty"`

	// Vendedor que recibe el pago
	VendedorID uint `gorm:"not null;index" json:"vendedor_id"`
	Vendedor   User `gorm:"foreignKey:VendedorID" json:"vendedor,omitempty"`

	// Periodo del pago
	MesCorrespondiente int `gorm:"not null" json:"mes_correspondiente"` // 1 = enero, 12 = diciembre
	Anio              int `gorm:"not null" json:"anio"`

	// Tipo de pago (cuota mensual, depósito de garantía o devolución del depósito)
	Tipo PaymentType `gorm:"size:30;not null;default:'cuota'" json:"tipo"`

	// Monto del pago
	Monto float64 `gorm:"type:decimal(15,2);not null" json:"monto"`

	// Fecha en que se realizó el pago
	FechaPago time.Time `gorm:"not null" json:"fecha_pago"`

	// Método de pago
	MetodoPago string `gorm:"size:50" json:"metodo_pago"` // transferencia, efectivo, qr, etc.

	// Estado del pago
	Estado PaymentStatus `gorm:"size:20;not null;default:'pendiente'" json:"estado"`

	// URL del comprobante de pago
	ComprobanteURL string `gorm:"type:text" json:"comprobante_url"`
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (RentPayment) TableName() string {
	return "rent_payments"
}
