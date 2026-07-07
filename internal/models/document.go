package models

import (
	"time"

	"gorm.io/gorm"
)

// Document - Modelo para documentos públicos del sistema
// LIMITACIÓN: Solo almacena documentos públicos, NO confidenciales
// - Fotos de propiedades (públicas)
// - Comprobantes de pago (propios del usuario)
// - Documentos generados por el sistema (contratos PDF)
// - NO: Documentos legales confidenciales del cliente
type Document struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Usuario que subió el documento
	UserID uint `gorm:"not null;index" json:"user_id"`
	User   User `gorm:"foreignKey:UserID" json:"-"`

	// Propiedad asociada (opcional)
	PropertyID  *uint    `gorm:"index" json:"property_id,omitempty"`
	Property    Property `gorm:"foreignKey:PropertyID" json:"property,omitempty"`

	// Tipo de documento
	// PROPERTY_PHOTO: Foto de propiedad (pública)
	// RECEIPT: Comprobante de pago (privado, solo del usuario)
	// CONTRACT: Contrato generado por el sistema (privado)
	// PUBLIC: Documento público general
	Type string `gorm:"size:50;not null" json:"type"`

	// Título descriptivo del documento
	Title string `gorm:"size:200;not null" json:"title"`

	// URL del archivo (Cloudflare R2, S3, etc.)
	FileURL string `gorm:"size:500;not null" json:"file_url"`

	// Nombre original del archivo
	FileName string `gorm:"size:255;not null" json:"file_name"`

	// Tamaño del archivo en bytes
	FileSize int64 `json:"file_size"`

	// Tipo MIME (image/jpeg, application/pdf, etc.)
	MimeType string `gorm:"size:100;not null" json:"mime_type"`

	// Indica si el documento es público (accesible por cualquiera)
	// TRUE para fotos de propiedades
	// FALSE para comprobantes de pago y contratos
	IsPublic bool `gorm:"default:false;not null" json:"is_public"`

	// Descripción opcional del documento
	Description string `gorm:"type:text" json:"description,omitempty"`
}

func (Document) TableName() string {
	return "documents"
}
