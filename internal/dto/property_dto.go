package dto

import (
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
)

// ========== REQUEST DTOS ==========

// CreatePropertyRequest - DTO para crear propiedad
type CreatePropertyRequest struct {
	// Información básica
	Titulo        string                     `json:"titulo" validate:"required,min=10,max=200"`
	Descripcion   string                     `json:"descripcion" validate:"required,min=50"`
	TipoPropiedad models.PropertyType        `json:"tipo_propiedad" validate:"required,oneof=casa departamento terreno local oficina galpon hacienda lote"`
	Operacion     models.PropertyOperation   `json:"operacion" validate:"required,oneof=venta alquiler anticretico"`

	// Ubicación
	Direccion string  `json:"direccion" validate:"max=300"`
	Ciudad    string  `json:"ciudad" validate:"required,min=3,max=100"`
	Zona      string  `json:"zona" validate:"max=100"`
	Latitud   *float64 `json:"latitud" validate:"omitempty,min=-90,max=90"`
	Longitud  *float64 `json:"longitud" validate:"omitempty,min=-180,max=180"`

	// Características principales
	Precio            *float64 `json:"precio" validate:"required,gt=0"`
	Moneda            string   `json:"moneda" validate:"omitempty,oneof=USD BOB"`
	MetrajeTotal      *float64 `json:"metraje_total" validate:"omitempty,gt=0"`
	MetrajeConstruido *float64 `json:"metraje_construido" validate:"omitempty,gt=0"`
	NumeroHabitaciones *int    `json:"numero_habitaciones" validate:"omitempty,min=0,max=50"`
	NumeroBanos       *int    `json:"numero_banos" validate:"omitempty,min=0,max=20"`
	NumeroGarages     *int    `json:"numero_garages" validate:"omitempty,min=0,max=10"`
	AntiguedadAnos    *int    `json:"antiguedad_anos" validate:"omitempty,min=0,max=200"`

	// Amenities
	TienePiscina    bool `json:"tiene_piscina"`
	TieneJardin     bool `json:"tiene_jardin"`
	TieneTerraza    bool `json:"tiene_terraza"`
	TieneServicio   bool `json:"tiene_servicio"`
	TieneSeguridad  bool `json:"tiene_seguridad"`
	EsAmueblado     bool `json:"es_amueblado"`
	AceptaMascotas  bool `json:"acepta_mascotas"`

	// Información de contacto (opcional, usa del usuario por defecto)
	ContactoNombre   string `json:"contacto_nombre" validate:"max=150"`
	ContactoTelefono string `json:"contacto_telefono" validate:"max=20"`
	ContactoEmail    string `json:"contacto_email" validate:"omitempty,email,max=150"`

	// Categorías (array de IDs o slugs)
	Categorias []string `json:"categorias" validate:"max=5"`

	// SEO (opcional, se genera automáticamente)
	MetaTitulo     string `json:"meta_titulo" validate:"max=200"`
	MetaDescripcion string `json:"meta_descripcion" validate:"max=500"`
}

// UpdatePropertyRequest - DTO para actualizar propiedad
type UpdatePropertyRequest struct {
	// Todos los campos opcionales para actualización parcial
	Titulo          *string                     `json:"titulo" validate:"omitempty,min=10,max=200"`
	Descripcion     *string                     `json:"descripcion" validate:"omitempty,min=50"`
	TipoPropiedad   *models.PropertyType        `json:"tipo_propiedad" validate:"omitempty,oneof=casa departamento terreno local oficina galpon hacienda lote"`
	Operacion       *models.PropertyOperation   `json:"operacion" validate:"omitempty,oneof=venta alquiler anticretico"`
	Estado          *models.PropertyStatus      `json:"estado" validate:"omitempty,oneof=borrador pendiente activa inactiva vendida alquilada rechazada"`

	Direccion       *string  `json:"direccion" validate:"omitempty,max=300"`
	Ciudad          *string  `json:"ciudad" validate:"omitempty,min=3,max=100"`
	Zona            *string  `json:"zona" validate:"omitempty,max=100"`
	Latitud         *float64 `json:"latitud" validate:"omitempty,min=-90,max=90"`
	Longitud        *float64 `json:"longitud" validate:"omitempty,min=-180,max=180"`

	Precio          *float64 `json:"precio" validate:"omitempty,gt=0"`
	Moneda          *string  `json:"moneda" validate:"omitempty,oneof=USD BOB"`
	MetrajeTotal    *float64 `json:"metraje_total" validate:"omitempty,gt=0"`
	MetrajeConstruido *float64 `json:"metraje_construido" validate:"omitempty,gt=0"`
	NumeroHabitaciones *int   `json:"numero_habitaciones" validate:"omitempty,min=0,max=50"`
	NumeroBanos      *int    `json:"numero_banos" validate:"omitempty,min=0,max=20"`
	NumeroGarages    *int    `json:"numero_garages" validate:"omitempty,min=0,max=10"`
	AntiguedadAnos   *int    `json:"antiguedad_anos" validate:"omitempty,min=0,max=200"`

	TienePiscina     *bool `json:"tiene_piscina"`
	TieneJardin      *bool `json:"tiene_jardin"`
	TieneTerraza     *bool `json:"tiene_terraza"`
	TieneServicio    *bool `json:"tiene_servicio"`
	TieneSeguridad   *bool `json:"tiene_seguridad"`
	EsAmueblado      *bool `json:"es_amueblado"`
	AceptaMascotas   *bool `json:"acepta_mascotas"`

	ContactoNombre   *string `json:"contacto_nombre" validate:"omitempty,max=150"`
	ContactoTelefono *string `json:"contacto_telefono" validate:"omitempty,max=20"`
	ContactoEmail    *string `json:"contacto_email" validate:"omitempty,email,max=150"`

	Categorias       []string `json:"categorias" validate:"omitempty,max=5"`

	Destacado        *bool    `json:"destacado"`
	MetaTitulo       *string  `json:"meta_titulo" validate:"omitempty,max=200"`
	MetaDescripcion *string  `json:"meta_descripcion" validate:"omitempty,max=500"`
}

// UploadPhotosRequest - DTO para subir fotos
type UploadPhotosRequest struct {
	Files     []PhotoFile `json:"files" validate:"required,min=1,max=7"`
	Orden     []int       `json:"orden"` // Orden opcional
	Descripciones []string `json:"descripciones"` // Descripciones opcionales
}

// PhotoFile - Representa un archivo de foto
type PhotoFile struct {
	Filename string    `json:"filename"`
	Size     int64     `json:"size" validate:"gt=0,max=5242880"` // Max 5MB
	Content  []byte    `json:"-"` // No se serializa en JSON
	MimeType string    `json:"mime_type" validate:"oneof=image/jpeg image/png image/webp"`
}

// SetPortadaRequest - DTO para marcar foto como portada
type SetPortadaRequest struct {
	FotoID uint `json:"foto_id" validate:"required"`
}

// ReorderPhotosRequest - DTO para reordenar fotos
type ReorderPhotosRequest struct {
	Orden []uint `json:"orden" validate:"required,min=1"`
}

// PropertyFilter - Filtros para búsqueda de propiedades
type PropertyFilter struct {
	// Paginación
	Page  int `json:"page" validate:"min=1,max=1000"`
	Limit int `json:"limit" validate:"min=1,max=100"`

	// Filtros básicos
	TipoPropiedad []string `json:"tipo_propiedad"` // casa,departamento,...
	Operacion     []string `json:"operacion"`     // venta,alquiler,anticretico
	Ciudad        string   `json:"ciudad"`
	Zona          string   `json:"zona"`
	Estado        string   `json:"estado"`

	// Filtros de rango
	PrecioMin      *float64 `json:"precio_min" validate:"omitempty,gt=0"`
	PrecioMax      *float64 `json:"precio_max" validate:"omitempty,gt=0"`
	MetrajeMin     *float64 `json:"metraje_min" validate:"omitempty,gt=0"`
	MetrajeMax     *float64 `json:"metraje_max" validate:"omitempty,gt=0"`
	HabitacionesMin *int    `json:"habitaciones_min" validate:"omitempty,min=0"`
	HabitacionesMax *int    `json:"habitaciones_max" validate:"omitempty,min=0"`
	BanosMin        *int    `json:"banos_min" validate:"omitempty,min=0"`
	BanosMax        *int    `json:"banos_max" validate:"omitempty,min=0"`

	// Filtros de amenities
	TienePiscina    *bool `json:"tiene_piscina"`
	TieneJardin     *bool `json:"tiene_jardin"`
	TieneTerraza    *bool `json:"tiene_terraza"`
	TieneServicio   *bool `json:"tiene_servicio"`
	TieneSeguridad  *bool `json:"tiene_seguridad"`
	EsAmueblado     *bool `json:"es_amueblado"`
	AceptaMascotas  *bool `json:"acepta_mascotas"`

	// Usuario (para listar propiedades de un usuario)
	UserID *uint `json:"user_id"`

	// Ordenamiento
	OrderBy string `json:"order_by" validate:"omitempty,oneof=precio_asc precio_desc mas_recientes mas_visitados relevancia"`

	// Búsqueda de texto
	SearchQuery string `json:"search_query"` // Búsqueda full-text
}

// ========== RESPONSE DTOS ==========

// PropertyResponse - Respuesta completa de propiedad
type PropertyResponse struct {
	ID        uint    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Información básica
	Titulo        string                `json:"titulo"`
	Descripcion   string                `json:"descripcion"`
	TipoPropiedad models.PropertyType   `json:"tipo_propiedad"`
	Operacion     models.PropertyOperation `json:"operacion"`
	Estado        models.PropertyStatus `json:"estado"`

	// Ubicación
	Direccion     string  `json:"direccion,omitempty"`
	Ciudad        string  `json:"ciudad"`
	Zona          string  `json:"zona,omitempty"`
	Latitud       *float64 `json:"latitud,omitempty"`
	Longitud      *float64 `json:"longitud,omitempty"`
	GoogleMapsURL string  `json:"google_maps_url,omitempty"`

	// Características
	Precio            *float64 `json:"precio"`
	Moneda            string   `json:"moneda"`
	MetrajeTotal      *float64 `json:"metraje_total,omitempty"`
	MetrajeConstruido *float64 `json:"metraje_construido,omitempty"`
	NumeroHabitaciones *int    `json:"numero_habitaciones,omitempty"`
	NumeroBanos       *int    `json:"numero_banos,omitempty"`
	NumeroGarages     *int    `json:"numero_garages,omitempty"`
	AntiguedadAnos    *int    `json:"antiguedad_anos,omitempty"`

	// Amenities
	TienePiscina    bool `json:"tiene_piscina"`
	TieneJardin     bool `json:"tiene_jardin"`
	TieneTerraza    bool `json:"tiene_terraza"`
	TieneServicio   bool `json:"tiene_servicio"`
	TieneSeguridad  bool `json:"tiene_seguridad"`
	EsAmueblado     bool `json:"es_amueblado"`
	AceptaMascotas  bool `json:"acepta_mascotas"`

	// Información del propietario
	UserID uint `json:"user_id"`

	// Métricas
	NumeroVisitas  int  `json:"numero_visitas"`
	NumeroContactos int `json:"numero_contactos"`
	Destacado      bool `json:"destacado"`

	// SEO
	Slug string `json:"slug"`

	// Relaciones (se cargan con eager loading)
	Fotos     []PhotoResponse      `json:"fotos,omitempty"`
	Categorias []CategoryResponse  `json:"categorias,omitempty"`
	Contacto  *ContactoResponse    `json:"contacto,omitempty"` // Solo para autenticados
}

// PropertyListResponse - Respuesta de listado de propiedades
type PropertyListResponse struct {
	Data       []PropertyListItemResponse `json:"data"`
	Pagination PaginationResponse         `json:"pagination"`
}

// PropertyListItemResponse - Item simplificado para listados
type PropertyListItemResponse struct {
	ID            uint                 `json:"id"`
	Titulo        string               `json:"titulo"`
	Descripcion   string               `json:"descripcion"` // Truncada en el handler
	Precio        *float64             `json:"precio"`
	Moneda        string               `json:"moneda"`
	TipoPropiedad models.PropertyType  `json:"tipo_propiedad"`
	Operacion     models.PropertyOperation `json:"operacion"`
	Ciudad        string               `json:"ciudad"`
	Zona          string               `json:"zona,omitempty"`
	MetrajeConstruido *float64         `json:"metraje_construido,omitempty"`
	NumeroHabitaciones *int            `json:"numero_habitaciones,omitempty"`
	NumeroBanos   *int                 `json:"numero_banos,omitempty"`
	FotoPortada   *string              `json:"foto_portada,omitempty"` // URL thumbnail
	Destacado      bool                `json:"destacado"`
	NumeroVisitas  int                 `json:"numero_visitas"`
	CreatedAt     time.Time            `json:"created_at"`
}

// PhotoResponse - Respuesta de foto
type PhotoResponse struct {
	ID           uint   `json:"id"`
	URL          string `json:"url"`
	URLThumbnail string `json:"url_thumbnail,omitempty"`
	Orden        int    `json:"orden"`
	Descripcion  string `json:"descripcion,omitempty"`
	EsPortada    bool   `json:"es_portada"`
}

// CategoryResponse - Respuesta de categoría
type CategoryResponse struct {
	ID    uint   `json:"id"`
	Nombre string `json:"nombre"`
	Slug  string `json:"slug"`
	Icono string `json:"icono,omitempty"`
}

// ContactoResponse - Información de contacto (solo para autenticados)
type ContactoResponse struct {
	Nombre          string `json:"nombre"`
	Telefono        string `json:"telefono"`
	WhatsApp        string `json:"whatsapp"`
	Email           string `json:"email,omitempty"`
	MostrarTelefono bool   `json:"mostrar_telefono"`
	MostrarWhatsApp bool   `json:"mostrar_whatsapp"`
	MostrarEmail    bool   `json:"mostrar_email"`
}

// PropertyStatsResponse - Estadísticas de propiedad
type PropertyStatsResponse struct {
	NumeroVisitas      int `json:"numero_visitas"`
	NumeroContactos    int `json:"numero_contactos"`
	UltimaVisita       time.Time `json:"ultima_visita,omitempty"`
	FotosCount         int `json:"fotos_count"`
	TienePortada       bool `json:"tiene_portada"`
}

// ========== HELPER METHODS ==========

// ToModel - Convierte CreatePropertyRequest a modelo
func (r *CreatePropertyRequest) ToModel(userID uint) *models.Property {
	moneda := "USD"
	if r.Moneda != "" {
		moneda = r.Moneda
	}

	return &models.Property{
		Titulo:        r.Titulo,
		Descripcion:   r.Descripcion,
		TipoPropiedad: r.TipoPropiedad,
		Operacion:     r.Operacion,
		Estado:        models.StatusBorrador, // Siempre empieza como borrador
		Direccion:     r.Direccion,
		Ciudad:        r.Ciudad,
		Zona:          r.Zona,
		Latitud:       r.Latitud,
		Longitud:      r.Longitud,
		Precio:        *r.Precio,
		Moneda:        moneda,
		MetrajeTotal:  r.MetrajeTotal,
		MetrajeConstruido: r.MetrajeConstruido,
		NumeroHabitaciones: r.NumeroHabitaciones,
		NumeroBanos:   r.NumeroBanos,
		NumeroGarages: r.NumeroGarages,
		AntiguedadAnos: r.AntiguedadAnos,
		TienePiscina:  r.TienePiscina,
		TieneJardin:   r.TieneJardin,
		TieneTerraza:  r.TieneTerraza,
		TieneServicio: r.TieneServicio,
		TieneSeguridad: r.TieneSeguridad,
		EsAmueblado:   r.EsAmueblado,
		AceptaMascotas: r.AceptaMascotas,
		UserID:        userID,
		ContactoNombre: r.ContactoNombre,
		ContactoTelefono: r.ContactoTelefono,
		ContactoEmail: r.ContactoEmail,
		MetaTitulo:    r.MetaTitulo,
		MetaDescripcion: r.MetaDescripcion,
	}
}

// SetDefaults - Establece valores por defecto para los filtros
func (f *PropertyFilter) SetDefaults() {
	if f.Page == 0 {
		f.Page = 1
	}
	if f.Limit == 0 {
		f.Limit = 20
	}
	if f.Estado == "" {
		f.Estado = "activa"
	}
}
