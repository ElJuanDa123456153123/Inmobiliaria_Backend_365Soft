package dto

import (
	"math"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
)

// ========== PAGINATION DTOs ==========

// PaginationRequest - Request de paginación genérico
type PaginationRequest struct {
	Page  int `json:"page" validate:"min=1,max=1000"`
	Limit int `json:"limit" validate:"min=1,max=100"`
}

// PaginationResponse - Respuesta de paginación
type PaginationResponse struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// NewPaginationResponse - Crea una respuesta de paginación
func NewPaginationResponse(page, limit int, total int64) PaginationResponse {
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return PaginationResponse{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// GetOffset - Calcula el offset para la query
func (p *PaginationRequest) GetOffset() int {
	return (p.Page - 1) * p.Limit
}

// GetLimit - Retorna el límite (con default)
func (p *PaginationRequest) GetLimit() int {
	if p.Limit == 0 {
		return 20 // Default
	}
	return p.Limit
}

// GetPage - Retorna la página (con default)
func (p *PaginationRequest) GetPage() int {
	if p.Page == 0 {
		return 1 // Default
	}
	return p.Page
}

// SetDefaults - Establece valores por defecto
func (p *PaginationRequest) SetDefaults() {
	if p.Page == 0 {
		p.Page = 1
	}
	if p.Limit == 0 {
		p.Limit = 20
	}
}

// ========== RESPONSE DTOS ==========

// Response - Respuesta estándar de la API
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorDetail `json:"error,omitempty"`
	Meta    *ResponseMeta `json:"meta,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ErrorDetail - Detalle de error
type ErrorDetail struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []interface{} `json:"details,omitempty"`
}

// ResponseMeta - Metadata de la respuesta
type ResponseMeta struct {
	Timestamp string `json:"timestamp"`
	RequestID string `json:"request_id,omitempty"`
	Version   string `json:"version,omitempty"`
}

// ========== FACTORY FUNCTIONS ==========

// SuccessResponse - Crea respuesta exitosa
func SuccessResponse(data interface{}) Response {
	return Response{
		Success: true,
		Data:    data,
	}
}

// SuccessResponseWithMessage - Crea respuesta exitosa con mensaje
func SuccessResponseWithMessage(data interface{}, message string) Response {
	return Response{
		Success: true,
		Data:    data,
		Message: message,
	}
}

// SuccessResponseWithPagination - Crea respuesta exitosa con paginación
func SuccessResponseWithPagination(data interface{}, pagination PaginationResponse) Response {
	return Response{
		Success: true,
		Data:    data,
		Meta: &ResponseMeta{
			Timestamp: timestampNow(),
		},
	}
}

// ErrorResponse - Crea respuesta de error
func ErrorResponse(code, message string) Response {
	return Response{
		Success: false,
		Error: &ErrorDetail{
			Code:    code,
			Message: message,
		},
	}
}

// ErrorResponseWithDetails - Crea respuesta de error con detalles
func ErrorResponseWithDetails(code, message string, details []interface{}) Response {
	return Response{
		Success: false,
		Error: &ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

// ValidationErrorResponse - Crea respuesta de error de validación
func ValidationErrorResponse(details []interface{}) Response {
	return Response{
		Success: false,
		Error: &ErrorDetail{
			Code:    "VALIDATION_ERROR",
			Message: "Error de validación en los datos enviados",
			Details: details,
		},
	}
}

// NotFoundResponse - Crea respuesta de recurso no encontrado
func NotFoundResponse(resource string) Response {
	return Response{
		Success: false,
		Error: &ErrorDetail{
			Code:    "NOT_FOUND",
			Message: resource + " no encontrado",
		},
	}
}

// UnauthorizedResponse - Crea respuesta de no autorizado
func UnauthorizedResponse() Response {
	return Response{
		Success: false,
		Error: &ErrorDetail{
			Code:    "UNAUTHORIZED",
			Message: "No autorizado para realizar esta acción",
		},
	}
}

// ForbiddenResponse - Crea respuesta de prohibido
func ForbiddenResponse() Response {
	return Response{
		Success: false,
		Error: &ErrorDetail{
			Code:    "FORBIDDEN",
			Message: "No tienes permisos para realizar esta acción",
		},
	}
}

// InternalErrorResponse - Crea respuesta de error interno
func InternalErrorResponse() Response {
	return Response{
		Success: false,
		Error: &ErrorDetail{
			Code:    "INTERNAL_ERROR",
			Message: "Error interno del servidor",
		},
	}
}

// ========== HELPER FUNCTIONS ==========

// timestampNow - Retorna timestamp actual en formato RFC3339
func timestampNow() string {
	return ""
}

// MapPropertyToResponse - Convierte modelo a respuesta completa
func MapPropertyToResponse(p *models.Property, includeContacto bool) *PropertyResponse {
	response := &PropertyResponse{
		ID:        p.ID,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
		Titulo:    p.Titulo,
		Descripcion: p.Descripcion,
		TipoPropiedad: p.TipoPropiedad,
		Operacion:  p.Operacion,
		Estado:     p.Estado,
		Direccion:  p.Direccion,
		Ciudad:     p.Ciudad,
		Zona:       p.Zona,
		Latitud:    p.Latitud,
		Longitud:   p.Longitud,
		GoogleMapsURL: p.GoogleMapsURL,
		Precio:     &p.Precio,
		Moneda:     p.Moneda,
		MetrajeTotal: p.MetrajeTotal,
		MetrajeConstruido: p.MetrajeConstruido,
		NumeroHabitaciones: p.NumeroHabitaciones,
		NumeroBanos: p.NumeroBanos,
		NumeroGarages: p.NumeroGarages,
		AntiguedadAnos: p.AntiguedadAnos,
		TienePiscina: p.TienePiscina,
		TieneJardin:  p.TieneJardin,
		TieneTerraza: p.TieneTerraza,
		TieneServicio: p.TieneServicio,
		TieneSeguridad: p.TieneSeguridad,
		EsAmueblado:   p.EsAmueblado,
		AceptaMascotas: p.AceptaMascotas,
		UserID:        p.UserID,
		NumeroVisitas: p.NumeroVisitas,
		NumeroContactos: p.NumeroContactos,
		Destacado:     p.Destacado,
		Slug:          p.Slug,
	}

	// Mapear fotos
	if len(p.Fotos) > 0 {
		response.Fotos = make([]PhotoResponse, len(p.Fotos))
		for i, foto := range p.Fotos {
			response.Fotos[i] = PhotoResponse{
				ID:           foto.ID,
				URL:          foto.URL,
				URLThumbnail: foto.URLThumbnail,
				Orden:        foto.Orden,
				Descripcion:  foto.Descripcion,
				EsPortada:    foto.EsPortada,
			}
		}
	}

	// Mapear categorías
	if len(p.Categorias) > 0 {
		response.Categorias = make([]CategoryResponse, len(p.Categorias))
		for i, cat := range p.Categorias {
			response.Categorias[i] = CategoryResponse{
				ID:    cat.ID,
				Nombre: cat.Nombre,
				Slug:  cat.Slug,
				Icono: cat.Icono,
			}
		}
	}

	// Contacto (solo si está autorizado)
	if includeContacto && p.User != nil {
		response.Contacto = &ContactoResponse{
			Nombre:   p.User.Nombre,
			Telefono: p.User.Telefono,
			WhatsApp: p.User.Telefono, // Mismo teléfono por ahora
			Email:    p.User.Email,
			MostrarTelefono: true,
			MostrarWhatsApp: true,
			MostrarEmail:    true,
		}
	} else if includeContacto {
		// Usar contacto personalizado si existe
		if p.ContactoNombre != "" {
			response.Contacto = &ContactoResponse{
				Nombre:   p.ContactoNombre,
				Telefono: p.ContactoTelefono,
				WhatsApp: p.ContactoTelefono,
				Email:    p.ContactoEmail,
				MostrarTelefono: p.ContactoTelefono != "",
				MostrarWhatsApp: p.ContactoTelefono != "",
				MostrarEmail:    p.ContactoEmail != "",
			}
		}
	}

	return response
}

// MapPropertyToListItem - Convierte modelo a item de listado
func MapPropertyToListItem(p *models.Property) PropertyListItemResponse {
	item := PropertyListItemResponse{
		ID:          p.ID,
		Titulo:      p.Titulo,
		Descripcion: truncateString(p.Descripcion, 150),
		Precio:      &p.Precio,
		Moneda:      p.Moneda,
		TipoPropiedad: p.TipoPropiedad,
		Operacion:   p.Operacion,
		Ciudad:      p.Ciudad,
		Zona:        p.Zona,
		MetrajeConstruido: p.MetrajeConstruido,
		NumeroHabitaciones: p.NumeroHabitaciones,
		NumeroBanos: p.NumeroBanos,
		Destacado:   p.Destacado,
		NumeroVisitas: p.NumeroVisitas,
		CreatedAt:   p.CreatedAt,
	}

	// Buscar foto de portada
	portada := p.GetPortada()
	if portada != nil {
		item.FotoPortada = &portada.URLThumbnail
	}

	return item
}

// truncateString - Trunca un string a una longitud máxima
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
