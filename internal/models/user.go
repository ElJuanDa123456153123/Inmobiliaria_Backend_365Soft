package models

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/datatypes"
)

// UserRole - Roles de usuario en el sistema
type UserRole string

const (
	RoleUsuario       UserRole = "usuario"       // Usuario normal
	RoleAgente        UserRole = "agente"        // Agente inmobiliario
	RoleInmobiliaria  UserRole = "inmobiliaria"  // Inmobiliaria
	RoleAdmin         UserRole = "admin"         // Administrador del sistema
)

// User - Modelo de usuario
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Información básica
	Nombre  string `gorm:"size:100;not null" json:"nombre"`
	Apellido string `gorm:"size:100" json:"apellido"`
	Email   string `gorm:"size:150;uniqueIndex;not null" json:"email"`
	Password string `gorm:"size:255;not null" json:"-"` // No se envía en JSON
	Telefono string `gorm:"size:20" json:"telefono"`

	// Datos públicos para propiedades
	NombreContacto  string `gorm:"size:100" json:"nombre_contacto,omitempty"`
	TelefonoContacto string `gorm:"size:20" json:"telefono_contacto,omitempty"`
	HorarioContacto string `gorm:"size:100" json:"horario_contacto,omitempty"`

	// Información personal adicional
	FechaNacimiento *time.Time `json:"fecha_nacimiento,omitempty"`
	Sexo            string     `gorm:"size:20" json:"sexo,omitempty"` // masculino, femenino, no_especificar
	Direccion       string     `gorm:"type:text" json:"direccion,omitempty"`
	CI              string     `gorm:"size:20" json:"ci,omitempty"` // Carnet de Identidad

	// Tipo de usuario
	Rol UserRole `gorm:"size:20;not null;default:'usuario'" json:"rol"`

	// Información de inmobiliaria (si aplica)
	NombreInmobiliaria string `gorm:"size:200" json:"nombre_inmobiliaria,omitempty"`
	NIT                string `gorm:"size:20" json:"nit,omitempty"`

	// Estado
	Activo      bool   `gorm:"default:true" json:"activo"`
	Verificado  bool   `gorm:"default:false" json:"verificado"`
	EmailToken  string `gorm:"size:255" json:"-"` // Token para verificación

	// Avatar (URL)
	AvatarURL string `gorm:"size:500" json:"avatar_url"`

	// QR de cobros (URL de imagen) — los inquilinos lo escanean para pagar rentas
	QrPagoURL string `gorm:"size:500" json:"qr_pago_url,omitempty"`

	// Múltiples QR de cobro por plataforma: [{"plataforma":"Yape","url":"..."}, ...]
	QrsCobro datatypes.JSON `gorm:"type:jsonb" json:"qrs_cobro,omitempty"`

	// Disponibilidad para visitas (JSON: dias, rangos, duracion_min). Null = default Lun-Sáb 9-12/14-18
	HorarioVisitas datatypes.JSON `gorm:"type:jsonb" json:"horario_visitas,omitempty"`
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (User) TableName() string {
	return "usuarios"
}

// BeforeCreate - Hook antes de crear un usuario
func (u *User) BeforeCreate(tx *gorm.DB) error {
	return nil
}
