package models

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/datatypes"
)

// PropertyStatus - Estados de una propiedad
type PropertyStatus string

const (
	PropertyStatusActivo       PropertyStatus = "activo"        // Propiedad activa y publicada
	PropertyStatusInactivo     PropertyStatus = "inactivo"      // Propiedad desactivada por el usuario
	PropertyStatusPendiente    PropertyStatus = "pendiente"     // Pendiente de revisión
	PropertyStatusVendido      PropertyStatus = "vendido"       // Propiedad vendada/alquilada
	PropertyStatusRechazado    PropertyStatus = "rechazado"     // Rechazada por admin
)

// PropertyType - Tipos de propiedad
type PropertyType string

const (
	PropertyTypeCasa          PropertyType = "casa"
	PropertyTypeDepartamento  PropertyType = "departamento"
	PropertyTypeTerreno       PropertyType = "terreno"
	PropertyTypeOficina       PropertyType = "oficina"
	PropertyTypeLocal         PropertyType = "local"
)

// OperationType - Tipos de operación
type OperationType string

const (
	OperationTypeVenta       OperationType = "venta"
	OperationTypeAlquiler    OperationType = "alquiler"
	OperationTypeAnticretico OperationType = "anticretico"
)

// Property - Modelo de propiedad
type Property struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Usuario que publica la propiedad
	UserID uint `gorm:"not null;index" json:"user_id"`
	User   User `gorm:"foreignKey:UserID" json:"user,omitempty"`

	// Imágenes de la propiedad
	PropertyImages []PropertyImage `gorm:"foreignKey:PropertyID" json:"imagenes,omitempty"`

	// Asesor asignado a la propiedad (opcional) - MANTENIDO POR COMPATIBILIDAD
	AdvisorID *uint    `gorm:"index" json:"advisor_id,omitempty"`
	Advisor   *Advisor `gorm:"foreignKey:AdvisorID" json:"advisor,omitempty"`

	// Múltiples asesores asignados a la propiedad (relación muchos a muchos)
	PropertyAdvisors []PropertyAdvisor `gorm:"foreignKey:PropertyID" json:"asesores_asignados,omitempty"`

	// === DATOS DE LA OPERACIÓN (PASO 1) ===
	Titulo          string        `gorm:"size:200;not null" json:"titulo"`
	TipoOperacion   OperationType `gorm:"size:20;not null;default:'venta'" json:"tipo_operacion"`
	Precio          float64       `gorm:"type:decimal(15,2);not null" json:"precio"`
	Moneda          string        `gorm:"size:5;not null;default:'BOB'" json:"moneda"` // USD, BOB
	AceptaPermuta   bool          `gorm:"default:false" json:"acepta_permuta"`
	Financia        bool          `gorm:"default:false" json:"financia"`

	// Campos específicos para ALQUILER
	Deposito        string `gorm:"size:20" json:"deposito,omitempty"`                   // 1 mes, 2 meses, etc.
	PlazoMinimo     string `gorm:"size:20" json:"plazo_minimo,omitempty"`              // 6 meses, 1 año, etc.
	ServiciosIncluidos datatypes.JSON `gorm:"type:jsonb" json:"servicios_incluidos,omitempty"` // ["luz", "agua", "internet"]

	// Campos específicos para ANTICRÉTICO
	DuracionContrato string `gorm:"size:20" json:"duracion_contrato,omitempty"`         // 1 año, 2 años, etc.
	OpcionCompra     bool   `gorm:"default:false" json:"opcion_compra,omitempty"`      // true/false

	// === DATOS GENERALES (PASO 2) ===
	TipoPropiedad    PropertyType `gorm:"size:30;not null" json:"tipo_propiedad"`
	Direccion        string       `gorm:"type:text" json:"direccion"`
	Zona             string       `gorm:"size:200;not null" json:"zona"`
	TipoMarcador     string       `gorm:"size:20;default:'punto-exacto'" json:"tipo_marcador"` // punto-exacto, aproximado
	Latitud          *float64     `json:"latitud,omitempty"`
	Longitud         *float64     `json:"longitud,omitempty"`
	MostrarDireccion bool         `gorm:"default:false" json:"mostrar_direccion"`

	// === DETALLES DE LA PROPIEDAD (PASO 3) ===
	Dormitorios      string  `gorm:"size:10" json:"dormitorios"` // 1Amb., 1, 2, 3, 4, 5+
	Banos            string  `gorm:"size:10" json:"banos"`       // 1, 2, 3+
	Garaje           string  `gorm:"size:10" json:"garaje"`      // 0, 1, 2, 3+
	MetrosEdificados float64 `gorm:"type:decimal(10,2)" json:"metros_edificados"`
	MetrosTerreno    float64 `gorm:"type:decimal(10,2)" json:"metros_terreno"`
	Estado           string  `gorm:"size:30" json:"estado"`      // sin-definir, excelente, muy-bueno, bueno, regular
	AnoConstruccion  int     `json:"ano_construccion,omitempty"`
	AptoOficina      bool    `gorm:"default:false" json:"apto_oficina"`
	ViviendaSocial   bool    `gorm:"default:false" json:"vivienda_social"`
	Condominio       bool    `gorm:"default:false" json:"condominio"`
	CantidadPisos    int     `gorm:"default:1" json:"cantidad_pisos"`

	// === CONFORT Y SEGURIDAD (PASO 4) ===
	Confort        datatypes.JSON `gorm:"type:jsonb" json:"confort,omitempty"`        // aire-acondicionado, calefaccion, etc.
	Seguridad      datatypes.JSON `gorm:"type:jsonb" json:"seguridad,omitempty"`      // seguridad-24h, porton-electrico, etc.
	NoAplicaConfort bool    `gorm:"default:false" json:"no_aplica_confort"`

	// === IMÁGENES Y VIDEO (PASO 5) ===
	VideoURL string `gorm:"size:500" json:"video_url,omitempty"`

	// === DESCRIPCIÓN (PASO 6) ===
	Descripcion         string `gorm:"type:text;not null" json:"descripcion"`
	TerminosCondiciones string `gorm:"type:text" json:"terminos_condiciones,omitempty"`

	// === ESTADO Y PUBLICACIÓN ===
	EstadoPropiedad PropertyStatus `gorm:"size:20;not null;default:'pendiente'" json:"estado_propiedad"`
	Destacado       bool           `gorm:"default:false" json:"destacado"`
	VistasContador  int            `gorm:"default:0" json:"vistas_contador"`
	FechaPublicacion *time.Time    `json:"fecha_publicacion,omitempty"`

	// === DATOS PÚBLICOS DEL VENDEDOR (SNAPSHOT) ===
	// Estos son snapshots de los datos del vendedor al momento de la publicación
	// Se mantienen aunque el usuario actualice su perfil posteriormente
	VendedorNombre     string `gorm:"size:150" json:"vendedor_nombre"`
	VendedorEmail      string `gorm:"size:150" json:"vendedor_email,omitempty"`
	VendedorTelefono   string `gorm:"size:30" json:"vendedor_telefono,omitempty"`
	VendedorContacto   string `gorm:"size:100" json:"vendedor_contacto,omitempty"`
	VendedorHorario    string `gorm:"size:100" json:"vendedor_horario,omitempty"`
	MostrarVendedor    bool   `gorm:"default:false" json:"mostrar_vendedor"` // Por defecto false (muestra 365Soft), true muestra vendedor real
	MostrarAgencia     bool   `gorm:"default:false" json:"mostrar_agencia"` // Controla si se muestra el nombre de la agencia independientemente
	AgenciaNombrePersonalizado string `gorm:"size:150" json:"agencia_nombre_personalizado,omitempty"` // Nombre personalizado para la agencia (opcional)
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (Property) TableName() string {
	return "propiedades"
}

// PropertyImage - Modelo de imágenes de propiedad
type PropertyImage struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Propiedad a la que pertenece la imagen
	PropertyID uint `gorm:"not null;index" json:"property_id"`
	Property   Property `gorm:"foreignKey:PropertyID" json:"-"`

	// URL de la imagen o base64 (almacenada en S3, Cloudinary, o como base64)
	ImageURL string `gorm:"type:text;not null" json:"image_url"`

	// Orden de la imagen para mostrar en el listado
	Orden int `gorm:"default:0" json:"orden"`

	// Si es la imagen principal/portada
	Principal bool `gorm:"default:false" json:"principal"`
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (PropertyImage) TableName() string {
	return "propiedades_imagenes"
}

// PropertyAdvisor - Modelo de unión para relación muchos a muchos entre Propiedades y Asesores
// Permite asignar múltiples asesores a una propiedad
type PropertyAdvisor struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Propiedad
	PropertyID uint `gorm:"not null;index" json:"property_id"`
	Property   Property `gorm:"foreignKey:PropertyID" json:"-"`

	// Asesor
	AdvisorID uint `gorm:"not null;index" json:"advisor_id"`
	Advisor   Advisor `gorm:"foreignKey:AdvisorID" json:"asesor,omitempty"`
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (PropertyAdvisor) TableName() string {
	return "propiedades_asesores"
}
