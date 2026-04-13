package models

import (
	"time"

	"gorm.io/gorm"
)

// PropertyType - Tipos de propiedad válidos
type PropertyType string

const (
	PropertyTypeCasa        PropertyType = "casa"
	PropertyTypeDepartamento PropertyType = "departamento"
	PropertyTypeTerreno     PropertyType = "terreno"
	PropertyTypeLocal       PropertyType = "local"
	PropertyTypeOficina     PropertyType = "oficina"
	PropertyTypeGalpon      PropertyType = "galpon"
	PropertyTypeHacienda    PropertyType = "hacienda"
	PropertyTypeLote        PropertyType = "lote"
)

// PropertyOperation - Tipos de operación
type PropertyOperation string

const (
	OperationVenta      PropertyOperation = "venta"
	OperationAlquiler   PropertyOperation = "alquiler"
	OperationAnticretico PropertyOperation = "anticretico"
)

// PropertyStatus - Estados de propiedad
type PropertyStatus string

const (
	StatusBorrador  PropertyStatus = "borrador"
	StatusPendiente PropertyStatus = "pendiente"
	StatusActiva    PropertyStatus = "activa"
	StatusInactiva  PropertyStatus = "inactiva"
	StatusVendida   PropertyStatus = "vendida"
	StatusAlquilada PropertyStatus = "alquilada"
	StatusRechazada PropertyStatus = "rechazada"
)

// Property - Modelo de propiedad inmobiliaria
type Property struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Información básica
	Titulo        string          `gorm:"size:200;not null" json:"titulo"`
	Descripcion   string          `gorm:"type:text;not null" json:"descripcion"`
	TipoPropiedad PropertyType    `gorm:"size:50;not null" json:"tipo_propiedad"`
	Operacion     PropertyOperation `gorm:"size:50;not null" json:"operacion"`
	Estado        PropertyStatus  `gorm:"size:50;not null;default:'borrador'" json:"estado"`

	// Ubicación
	Direccion     string  `gorm:"size:300" json:"direccion"`
	Ciudad        string  `gorm:"size:100;not null" json:"ciudad"`
	Zona          string  `gorm:"size:100" json:"zona"`
	Latitud       *float64 `gorm:"type:decimal(10,8)" json:"latitud,omitempty"`
	Longitud      *float64 `gorm:"type:decimal(11,8)" json:"longitud,omitempty"`
	GoogleMapsURL string  `gorm:"size:500" json:"google_maps_url,omitempty"`

	// Características principales
	Precio          float64 `gorm:"type:decimal(15,2);not null" json:"precio"`
	Moneda          string  `gorm:"size:10;not null;default:'USD'" json:"moneda"`
	MetrajeTotal    *float64 `gorm:"type:decimal(10,2)" json:"metraje_total,omitempty"`
	MetrajeConstruido *float64 `gorm:"type:decimal(10,2)" json:"metraje_construido,omitempty"`
	NumeroHabitaciones *int   `json:"numero_habitaciones,omitempty"`
	NumeroBanos      *int    `json:"numero_banos,omitempty"`
	NumeroGarages    *int    `json:"numero_garages,omitempty"`
	AntiguedadAnos   *int    `json:"antiguedad_anos,omitempty"`

	// Amenities
	TienePiscina     bool `gorm:"default:false" json:"tiene_piscina"`
	TieneJardin      bool `gorm:"default:false" json:"tiene_jardin"`
	TieneTerraza     bool `gorm:"default:false" json:"tiene_terraza"`
	TieneServicio    bool `gorm:"default:false" json:"tiene_servicio"`
	TieneSeguridad   bool `gorm:"default:false" json:"tiene_seguridad"`
	EsAmueblado      bool `gorm:"default:false" json:"es_amueblado"`
	AceptaMascotas   bool `gorm:"default:false" json:"acepta_mascotas"`

	// Información del propietario
	UserID          uint   `gorm:"not null" json:"user_id"`
	User            *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ContactoNombre  string `gorm:"size:150" json:"contacto_nombre,omitempty"`
	ContactoTelefono string `gorm:"size:20" json:"contacto_telefono,omitempty"`
	ContactoEmail   string `gorm:"size:150" json:"contacto_email,omitempty"`

	// Visualización y métricas
	NumeroVisitas  int    `gorm:"default:0" json:"numero_visitas"`
	NumeroContactos int   `gorm:"default:0" json:"numero_contactos"`
	Destacado      bool   `gorm:"default:false;index" json:"destacado"`

	// SEO
	Slug           string `gorm:"size:200;uniqueIndex;not null" json:"slug"`
	MetaTitulo     string `gorm:"size:200" json:"meta_titulo,omitempty"`
	MetaDescripcion string `gorm:"size:500" json:"meta_descripcion,omitempty"`

	// Relaciones
	Fotos     []PropertyPhoto  `gorm:"foreignKey:PropertyID;constraint:OnDelete:CASCADE" json:"fotos,omitempty"`
	Categorias []Category      `gorm:"many2many:propiedades_categorias;" json:"categorias,omitempty"`
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (Property) TableName() string {
	return "propiedades"
}

// BeforeCreate - Hook antes de crear (generar slug por defecto)
func (p *Property) BeforeCreate(tx *gorm.DB) error {
	// El slug se generará en el service layer
	return nil
}

// GetPortada - Retorna la foto marcada como portada
func (p *Property) GetPortada() *PropertyPhoto {
	for i := range p.Fotos {
		if p.Fotos[i].EsPortada {
			return &p.Fotos[i]
		}
	}
	// Si no hay portada, retornar la primera foto
	if len(p.Fotos) > 0 {
		return &p.Fotos[0]
	}
	return nil
}

// IsActive - Retorna true si la propiedad está activa
func (p *Property) IsActive() bool {
	return p.Estado == StatusActiva
}

// IsOwnedBy - Verifica si la propiedad pertenece a un usuario
func (p *Property) IsOwnedBy(userID uint) bool {
	return p.UserID == userID
}
