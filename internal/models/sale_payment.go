package models

import (
	"time"

	"gorm.io/gorm"
)

// SalePaymentStatus - Estados de un pago de venta
type SalePaymentStatus string

const (
	SalePaymentStatusPendiente     SalePaymentStatus = "pendiente"
	SalePaymentStatusComprobante   SalePaymentStatus = "comprobante_enviado"
	SalePaymentStatusConfirmado    SalePaymentStatus = "confirmado"
)

// SalePayment - Pago de una venta registrada en el sistema. El dinero circula
// fuera de la plataforma (QR interbancario / transferencia del comprador hacia
// el vendedor); el sistema actúa como libro contable: registra cada pago con
// su monto, fecha y comprobante fotográfico hasta su confirmación.
type SalePayment struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Venta a la que pertenece
	PurchaseRequestID uint            `gorm:"not null;index" json:"purchase_request_id"`
	PurchaseRequest   PurchaseRequest `gorm:"foreignKey:PurchaseRequestID" json:"purchase_request,omitempty"`

	// Partes (denormalizadas para consultas rápidas)
	CompradorID uint `gorm:"not null;index" json:"comprador_id"`
	VendedorID  uint `gorm:"not null;index" json:"vendedor_id"`

	// Tipo de pago: unico | arras | saldo
	Tipo string `gorm:"size:20;not null" json:"tipo"`

	// Monto y vencimiento
	Monto             float64    `gorm:"type:decimal(15,2);not null" json:"monto"`
	FechaVencimiento  *time.Time `json:"fecha_vencimiento,omitempty"`

	// Registro del pago por el comprador
	MetodoPago        string     `gorm:"size:30" json:"metodo_pago,omitempty"` // qr | transferencia | efectivo
	ComprobanteURL    string     `gorm:"size:500" json:"comprobante_url,omitempty"`
	FechaPago         *time.Time `json:"fecha_pago,omitempty"`
	FechaComprobante  *time.Time `json:"fecha_comprobante,omitempty"` // cuándo envió el comprobante

	// Estado: pendiente → comprobante_enviado → confirmado
	Estado SalePaymentStatus `gorm:"size:30;not null;default:'pendiente';index" json:"estado"`
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (SalePayment) TableName() string {
	return "pagos_venta"
}
