package models

import (
	"time"

	"gorm.io/gorm"
)

// SaleDocEstado - Estados de un documento del checklist de la venta
type SaleDocEstado string

const (
	SaleDocPendiente  SaleDocEstado = "pendiente"  // Aún no subido por el vendedor
	SaleDocSubido     SaleDocEstado = "subido"     // Subido, pendiente de verificación
	SaleDocVerificado SaleDocEstado = "verificado" // Verificado por el vendedor
)

// Tipos de documento requeridos al vendedor en una compraventa
const (
	SaleDocTipoCI       = "ci"                     // Cédula de identidad del propietario
	SaleDocTipoGravamen = "certificado_gravamenes" // Certificado de gravámenes (Derechos Reales)
	SaleDocTipoCatastro = "ficha_catastral"        // Ficha catastral
	SaleDocTipoIPVA     = "ipva"                   // IPVA al día (impuesto a la propiedad)
)

// TiposDocumentoVenta - Orden y etiquetas del checklist documental de la venta
var TiposDocumentoVenta = map[string]string{
	SaleDocTipoCI:       "Cédula de identidad del propietario",
	SaleDocTipoGravamen: "Certificado de gravámenes (Derechos Reales)",
	SaleDocTipoCatastro: "Ficha catastral",
	SaleDocTipoIPVA:     "IPVA al día (impuesto a la propiedad)",
}

// OrdenTiposDocumentoVenta - Orden fijo de presentación del checklist
var OrdenTiposDocumentoVenta = []string{SaleDocTipoCI, SaleDocTipoGravamen, SaleDocTipoCatastro, SaleDocTipoIPVA}

// SaleDocument - Documento del checklist documental de una compraventa.
// Al aceptarse la oferta se generan los 4 documentos requeridos al vendedor
// con estado "pendiente"; el vendedor sube el archivo y los va marcando como
// verificados. El comprador puede ver el avance desde Mis compras.
type SaleDocument struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Venta a la que pertenece
	PurchaseRequestID uint            `gorm:"not null;index" json:"purchase_request_id"`
	PurchaseRequest   PurchaseRequest `gorm:"foreignKey:PurchaseRequestID" json:"purchase_request,omitempty"`

	// Tipo de documento (ci | certificado_gravamenes | ficha_catastral | ipva)
	Tipo string `gorm:"size:40;not null;index" json:"tipo"`

	// Estado: pendiente → subido → verificado
	Estado SaleDocEstado `gorm:"size:20;not null;default:'pendiente'" json:"estado"`

	// Archivo subido por el vendedor (URL devuelta por /api/imagenes/upload)
	ArchivoURL string `gorm:"size:500" json:"archivo_url,omitempty"`
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (SaleDocument) TableName() string {
	return "documentos_venta"
}
