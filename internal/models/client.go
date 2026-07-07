package models

import (
	"time"

	"gorm.io/gorm"
)

// ClientStatus - Estados de un cliente en el pipeline de ventas
type ClientStatus string

const (
	ClientStatusNuevo          ClientStatus = "Nuevo"           // 🔵 Cliente recién creado
	ClientStatusContactado     ClientStatus = "Contactado"      // 🟡 Se le contactó
	ClientStatusInteresado     ClientStatus = "Interesado"      // 🟢 Mostró interés
	ClientStatusEnNegociacion  ClientStatus = "EnNegociacion"   // 🟠 Negociando precio
	ClientStatusCerrado        ClientStatus = "Cerrado"         // ✅ Venta cerrada
	ClientStatusPerdido        ClientStatus = "Perdido"         // 🔴 No compró
)

// ClientOrigin - Origen de donde vino el cliente
type ClientOrigin string

const (
	ClientOriginWeb         ClientOrigin = "Web"           // Vino de la web
	ClientOriginWhatsApp     ClientOrigin = "WhatsApp"      // Vino por WhatsApp
	ClientOriginReferencia   ClientOrigin = "Referencia"    // Referencia de otro cliente
	ClientOriginInstagram    ClientOrigin = "Instagram"     // Vio en Instagram
	ClientOriginFacebook     ClientOrigin = "Facebook"      // Vio en Facebook
	ClientOriginTelefono     ClientOrigin = "Telefono"      // Llamó por teléfono
)

// Client - Modelo de cliente/interesado para el CRM
type Client struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// === DATOS DEL CLIENTE ===
	Nombre    string `gorm:"size:150;not null" json:"nombre"`
	Telefono  string `gorm:"size:20;not null" json:"telefono"`
	WhatsApp  string `gorm:"size:20" json:"whatsapp,omitempty"`
	Email     string `gorm:"size:150" json:"email,omitempty"`

	// === ESTADO DEL CLIENTE (PIPELINE) ===
	Estado ClientStatus `gorm:"size:20;not null;default:'Nuevo'" json:"estado"`

	// === ORIGEN DEL CLIENTE ===
	Origen ClientOrigin `gorm:"size:50;not null" json:"origen"`

	// === PROPIEDAD DE INTERÉS ===
	PropertyID *uint     `gorm:"default:null" json:"property_id,omitempty"`
	Property   *Property `gorm:"foreignKey:PropertyID" json:"property,omitempty"`

	// === NOTAS Y SEGUIMIENTO ===
	Notas           string     `gorm:"type:text" json:"notas,omitempty"`
	ProximoContacto *time.Time `json:"proximo_contacto,omitempty"`

	// === ASESOR RESPONSABLE ===
	AdvisorID uint    `gorm:"not null;index" json:"advisor_id"`
	Advisor   *Advisor `gorm:"foreignKey:AdvisorID" json:"advisor,omitempty"`

	// === INTERACCIONES DEL CLIENTE ===
	Interactions []Interaction `gorm:"foreignKey:ClientID" json:"interacciones,omitempty"`
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (Client) TableName() string {
	return "clients"
}

// GetStatusColor - Retorna el color del estado para el frontend
func (c *Client) GetStatusColor() string {
	switch c.Estado {
	case ClientStatusNuevo:
		return "blue"
	case ClientStatusContactado:
		return "yellow"
	case ClientStatusInteresado:
		return "green"
	case ClientStatusEnNegociacion:
		return "orange"
	case ClientStatusCerrado:
		return "green"
	case ClientStatusPerdido:
		return "red"
	default:
		return "gray"
	}
}

// GetLastInteraction - Obtiene la última interacción del cliente
func (c *Client) GetLastInteraction() *Interaction {
	if len(c.Interactions) == 0 {
		return nil
	}
	return &c.Interactions[0]
}
