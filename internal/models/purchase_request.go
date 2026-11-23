package models

import (
	"time"

	"gorm.io/gorm"
)

// PurchaseRequestStatus - Estados de una solicitud de compra
type PurchaseRequestStatus string

const (
	PurchaseRequestStatusPendiente PurchaseRequestStatus = "pendiente"
	PurchaseRequestStatusAceptada  PurchaseRequestStatus = "aceptada"
	PurchaseRequestStatusRechazada PurchaseRequestStatus = "rechazada"
	PurchaseRequestStatusAnulada   PurchaseRequestStatus = "anulada"
)

// PurchaseRequest - Solicitud de compra que envía un interesado sobre una
// propiedad publicada en venta. Al ser aceptada por el propietario, la
// propiedad queda marcada como vendida y las demás solicitudes se rechazan.
type PurchaseRequest struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Propiedad solicitada (siempre de tipo venta)
	PropertyID uint     `gorm:"not null;index" json:"property_id"`
	Property   Property `gorm:"foreignKey:PropertyID" json:"property,omitempty"`

	// Comprador que envía la solicitud
	CompradorID uint `gorm:"not null;index" json:"comprador_id"`
	Comprador   User `gorm:"foreignKey:CompradorID" json:"comprador,omitempty"`

	// Dueño de la propiedad (denormalizado para consultas rápidas)
	VendedorID uint `gorm:"not null;index" json:"vendedor_id"`
	Vendedor   User `gorm:"foreignKey:VendedorID" json:"vendedor,omitempty"`

	// Datos de la oferta
	Mensaje          string  `gorm:"type:text" json:"mensaje,omitempty"`
	TelefonoContacto string  `gorm:"size:30" json:"telefono_contacto,omitempty"`
	MontoOfertado    float64 `gorm:"type:decimal(15,2);default:0" json:"monto_ofertado"` // 0 = al precio publicado

	// Al aceptarse: monto final de la venta y fecha del cierre
	MontoAceptado float64    `gorm:"type:decimal(15,2);default:0" json:"monto_aceptado"`
	FechaAceptada *time.Time `json:"fecha_aceptada,omitempty"`

	// Anulación de la venta (fraude, arrepentimiento, etc.). Los pagos ya
	// registrados se conservan como historial/evidencia.
	MotivoAnulacion string     `gorm:"size:300" json:"motivo_anulacion,omitempty"`
	FechaAnulada    *time.Time `json:"fecha_anulada,omitempty"`

	// Pagos de la venta (único, arras, saldo...)
	Pagos []SalePayment `gorm:"foreignKey:PurchaseRequestID" json:"pagos,omitempty"`

	// Estado de la solicitud
	Estado PurchaseRequestStatus `gorm:"size:20;not null;default:'pendiente';index" json:"estado"`
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (PurchaseRequest) TableName() string {
	return "solicitudes_compra"
}
