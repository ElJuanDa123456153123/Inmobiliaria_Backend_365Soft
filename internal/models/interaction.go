package models

import (
	"time"

	"gorm.io/gorm"
)

// InteractionType - Tipos de interacción posibles
type InteractionType string

const (
	InteractionTypeLlamada    InteractionType = "Llamada"    // 📞 Llamada telefónica
	InteractionTypeWhatsApp   InteractionType = "WhatsApp"   // 💬 Mensaje de WhatsApp
	InteractionTypeEmail      InteractionType = "Email"      // ✉️ Correo electrónico
	InteractionTypeVisita     InteractionType = "Visita"      // 🏠 Visita a la propiedad
	InteractionTypeNota       InteractionType = "Nota"       // 📝 Nota general
	InteractionTypeReunion    InteractionType = "Reunion"    // 👥 Reunión presencial
)

// Interaction - Modelo de interacción con un cliente
type Interaction struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// === RELACIÓN CON EL CLIENTE ===
	ClientID uint    `gorm:"not null;index" json:"client_id"`
	Client   *Client `gorm:"foreignKey:ClientID" json:"client,omitempty"`

	// === ASESOR QUE HIZO LA INTERACCIÓN ===
	AdvisorID uint     `gorm:"not null;index" json:"advisor_id"`
	Advisor   *Advisor `gorm:"foreignKey:AdvisorID" json:"advisor,omitempty"`

	// === TIPO DE INTERACCIÓN ===
	Tipo InteractionType `gorm:"size:20;not null" json:"tipo"`

	// === CONTENIDO DE LA INTERACCIÓN ===
	Notas string `gorm:"type:text;not null" json:"notas"`

	// === PROGRAMACIÓN DE PRÓXIMO CONTACTO ===
	ProximoContacto *time.Time `json:"proximo_contacto,omitempty"`

	// === ESTADO DEL CLIENTE DESPUÉS DE ESTA INTERACCIÓN ===
	// Permite rastrear cómo cambió el estado con cada interacción
	ClienteEstado ClientStatus `gorm:"size:20" json:"cliente_estado,omitempty"`
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (Interaction) TableName() string {
	return "interactions"
}

// GetTypeIcon - Retorna el icono para el tipo de interacción
func (i *Interaction) GetTypeIcon() string {
	switch i.Tipo {
	case InteractionTypeLlamada:
		return "📞"
	case InteractionTypeWhatsApp:
		return "💬"
	case InteractionTypeEmail:
		return "✉️"
	case InteractionTypeVisita:
		return "🏠"
	case InteractionTypeNota:
		return "📝"
	case InteractionTypeReunion:
		return "👥"
	default:
		return "💭"
	}
}

// GetTypeColor - Retorna el color para el tipo de interacción
func (i *Interaction) GetTypeColor() string {
	switch i.Tipo {
	case InteractionTypeLlamada:
		return "blue"
	case InteractionTypeWhatsApp:
		return "green"
	case InteractionTypeEmail:
		return "purple"
	case InteractionTypeVisita:
		return "orange"
	case InteractionTypeNota:
		return "gray"
	case InteractionTypeReunion:
		return "indigo"
	default:
		return "gray"
	}
}
