package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/repository"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/service"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/microcosm-cc/bluemonday"
)

// PropertyHandler - Handler para propiedades
type PropertyHandler struct {
	PropertyService *service.PropertyService
	Redis           *config.RedisClient
}

// NewPropertyHandler - Crea un nuevo handler de propiedades con Service Layer
func NewPropertyHandler() (*PropertyHandler, error) {
	db, err := config.NewDatabase()
	if err != nil {
		return nil, err
	}

	// Inicializar repositorios
	propertyRepo := repository.NewPropertyRepository(db)
	imageRepo := repository.NewPropertyImageRepository(db)
	userRepo := repository.NewUserRepository(db)
	auditRepo := repository.NewAuditRepository(db)

	// Inicializar servicio
	propertyService := service.NewPropertyService(propertyRepo, imageRepo, userRepo, auditRepo)

	// Inicializar Redis (opcional, si falla continúa sin caché)
	var redis *config.RedisClient
	redis, err = config.NewRedis()
	if err != nil {
		// Redis no está disponible, continuar sin caché
		fmt.Println("⚠️  Redis no disponible, ejecutando sin caché")
		redis = nil
	}

	return &PropertyHandler{
		PropertyService: propertyService,
		Redis:           redis,
	}, nil
}

// CreatePropertyRequest - Estructura para crear propiedad
type CreatePropertyRequest struct {
	// === DATOS DE LA OPERACIÓN (PASO 1) ===
	Titulo        string  `json:"titulo"`
	TipoOperacion string  `json:"tipo_operacion"`
	Precio        float64 `json:"precio"`
	Moneda        string  `json:"moneda"`
	AceptaPermuta bool    `json:"acepta_permuta"`
	Financia      bool    `json:"financia"`

	// === DATOS GENERALES (PASO 2) ===
	TipoPropiedad    string   `json:"tipo_propiedad"`
	Direccion        string   `json:"direccion"`
	Zona             string   `json:"zona"`
	TipoMarcador     string   `json:"tipo_marcador"`
	Latitud          *float64 `json:"latitud"`
	Longitud         *float64 `json:"longitud"`
	MostrarDireccion bool     `json:"mostrar_direccion"`

	// === DETALLES DE LA PROPIEDAD (PASO 3) ===
	Dormitorios     string   `json:"dormitorios"`
	Banos           string   `json:"banos"`
	Garaje          string   `json:"garaje"`
	MetrosEdificados float64 `json:"metros_edificados"`
	MetrosTerreno   float64  `json:"metros_terreno"`
	Estado          string   `json:"estado"`
	AnoConstruccion int      `json:"ano_construccion"`
	AptoOficina     bool     `json:"apto_oficina"`
	ViviendaSocial  bool     `json:"vivienda_social"`
	Condominio      bool     `json:"condominio"`
	CantidadPisos   int      `json:"cantidad_pisos"`

	// === CONFORT Y SEGURIDAD (PASO 4) ===
	Confort        []string `json:"confort"`
	Seguridad      []string `json:"seguridad"`
	NoAplicaConfort bool    `json:"no_aplica_confort"`

	// === IMÁGENES Y VIDEO (PASO 5) ===
	Imagenes []struct {
		URL   string `json:"url"`   // URL de la imagen (ej: /api/imagenes/propiedad-123-1.webp)
		Orden int    `json:"orden"`
	} `json:"imagenes"`
	VideoURL string `json:"video_url"`

	// === DESCRIPCIÓN (PASO 6) ===
	Descripcion         string `json:"descripcion"`
	TerminosCondiciones string `json:"terminos_condiciones"`
}

// CreateProperty - Crea una nueva propiedad
func (h *PropertyHandler) CreateProperty(c *fiber.Ctx) error {
	// Obtener usuario del contexto (inyectado por middleware)
	user := c.Locals("user").(*models.User)

	// Parsear request body
	var req CreatePropertyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	// 🔒 Sanitización HTML para prevenir XSS
	policy := bluemonday.UGCPolicy()
	policy.AllowElements("a", "b", "strong", "em", "i", "u", "ul", "ol", "li", "p", "br", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote", "span", "div")
	policy.AllowAttrs("href").OnElements("a")
	policy.AllowAttrs("class").OnElements("span", "div", "p")
	policy.AllowAttrs("target").OnElements("a")
	policy.RequireNoFollowOnLinks(true)

	// Preparar imágenes para el servicio
	imagenes := make([]service.CreatePropertyImageRequest, len(req.Imagenes))
	for i, img := range req.Imagenes {
		imagenes[i] = service.CreatePropertyImageRequest{
			URL:   img.URL,
			Orden: img.Orden,
		}
	}

	// Preparar request para el servicio
	serviceReq := &service.CreatePropertyRequest{
		UserID:              user.ID,
		Titulo:              req.Titulo,
		TipoOperacion:       req.TipoOperacion,
		Precio:              req.Precio,
		Moneda:              req.Moneda,
		AceptaPermuta:       req.AceptaPermuta,
		Financia:            req.Financia,
		TipoPropiedad:       req.TipoPropiedad,
		Direccion:           req.Direccion,
		Zona:                req.Zona,
		TipoMarcador:        req.TipoMarcador,
		Latitud:             req.Latitud,
		Longitud:            req.Longitud,
		MostrarDireccion:    req.MostrarDireccion,
		Dormitorios:         req.Dormitorios,
		Banos:               req.Banos,
		Garaje:              req.Garaje,
		MetrosEdificados:    req.MetrosEdificados,
		MetrosTerreno:       req.MetrosTerreno,
		Estado:              req.Estado,
		AnoConstruccion:     req.AnoConstruccion,
		AptoOficina:         req.AptoOficina,
		ViviendaSocial:      req.ViviendaSocial,
		Condominio:          req.Condominio,
		CantidadPisos:       req.CantidadPisos,
		Confort:             req.Confort,
		Seguridad:           req.Seguridad,
		NoAplicaConfort:     req.NoAplicaConfort,
		Imagenes:            imagenes,
		VideoURL:            req.VideoURL,
		Descripcion:         policy.Sanitize(strings.TrimSpace(req.Descripcion)),
		TerminosCondiciones: policy.Sanitize(strings.TrimSpace(req.TerminosCondiciones)),
	}

	// Crear propiedad a través del servicio
	property, err := h.PropertyService.CreateProperty(serviceReq)
	if err != nil {
		// Manejar diferentes tipos de errores
		if appErr, ok := err.(*errors.AppError); ok {
			return c.Status(appErr.HTTPStatus).JSON(fiber.Map{
				"error": appErr.Message,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creando la propiedad",
		})
	}

	// Recargar propiedad con imágenes para devolverla completa
	property, err = h.PropertyService.GetPropertyByID(property.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo la propiedad creada",
		})
	}

	// 🔒 Auditoría: Log de creación
	h.LogAudit(c, "create_property", "property", &property.ID, &user.ID, map[string]interface{}{
		"titulo":         property.Titulo,
		"precio":          property.Precio,
		"moneda":          property.Moneda,
		"tipo_operacion":  property.TipoOperacion,
	})

	// 🚀 Redis: Caché y pub/sub para actualizaciones en tiempo real
	if h.Redis != nil {
		h.Redis.CacheProperty(property.ID, property)
		h.Redis.InvalidatePropertyLists()
		h.Redis.InvalidateUserPropertyCount(user.ID)
		h.Redis.PublishPropertyUpdate("created", property.ID, map[string]interface{}{
			"id":        property.ID,
			"titulo":    property.Titulo,
			"precio":    property.Precio,
			"moneda":    property.Moneda,
			"operacion": property.TipoOperacion,
		})
	}

	// Retornar respuesta
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":  "Propiedad creada exitosamente",
		"property": property,
	})
}

// GetProperty - Obtiene una propiedad por ID
func (h *PropertyHandler) GetProperty(c *fiber.Ctx) error {
	// Obtener ID de la propiedad
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de propiedad inválido",
		})
	}

	var property *models.Property

	// 🚀 Redis: Intentar obtener desde caché primero
	if h.Redis != nil {
		cachedProperty := new(models.Property)
		err := h.Redis.GetCachedProperty(uint(id), cachedProperty)
		if err == nil {
			// Found in cache - incrementar vistas
			h.PropertyService.IncrementViewCount(uint(id))
			return c.JSON(cachedProperty)
		}
		// Cache miss, continue to DB
	}

	// Obtener propiedad a través del servicio
	property, err = h.PropertyService.GetPropertyByID(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Propiedad no encontrada",
		})
	}

	// Incrementar contador de vistas
	h.PropertyService.IncrementViewCount(uint(id))

	// 🚀 Redis: Guardar en caché
	if h.Redis != nil {
		h.Redis.CacheProperty(uint(id), property)
	}

	return c.JSON(property)
}

// ListPropertiesRequest - Parámetros para listar propiedades
type ListPropertiesRequest struct {
	// === FILTROS ===
	TipoOperacion  string  `json:"tipo_operacion"`  // venta, alquiler
	TipoPropiedad  string  `json:"tipo_propiedad"`  // casa, departamento, terreno, etc.
	Zona           string  `json:"zona"`            // Búsqueda parcial de zona
	Estado         string  `json:"estado"`          // activo, inactivo, vendido
	Moneda         string  `json:"moneda"`          // USD, BOB
	PrecioMin      float64 `json:"precio_min"`      // Precio mínimo
	PrecioMax      float64 `json:"precio_max"`      // Precio máximo
	MetrosMin      float64 `json:"metros_min"`      // Metros edificados mínimos
	MetrosMax      float64 `json:"metros_max"`      // Metros edificados máximos
	Dormitorios    string  `json:"dormitorios"`     // Número exacto o rango
	Banos          string  `json:"banos"`           // Número exacto o rango
	Garaje         string  `json:"garaje"`          // si, no
	AceptaPermuta  *bool   `json:"acepta_permuta"`  // true/false
	Financia       *bool   `json:"financia"`        // true/false
	Condominio     *bool   `json:"condominio"`      // true/false
	AptoOficina    *bool   `json:"apto_oficina"`    // true/false
	ViviendaSocial *bool   `json:"vivienda_social"` // true/false
	SearchText     string  `json:"search"`          // Búsqueda en título y descripción

	// === ORDENAMIENTO ===
	OrderBy string `json:"order_by"`  // precio, fecha, vistas, metros
	Order   string `json:"order"`     // asc, desc

	// === PAGINACIÓN ===
	Page  int `json:"page"`  // Número de página (empezando en 1)
	Limit int `json:"limit"` // Resultados por página (max 100)
}

// ListProperties - Lista todas las propiedades (con filtros avanzados, ordenamiento y paginación)
func (h *PropertyHandler) ListProperties(c *fiber.Ctx) error {
	// 🚀 Redis: Verificar caché primero
	cacheKey := c.Context().QueryArgs().String()
	if h.Redis != nil {
		var cachedResponse struct {
			Propiedades []models.Property `json:"propiedades"`
			Total       int64             `json:"total"`
			Page        int               `json:"page"`
			Limit       int               `json:"limit"`
			TotalPages  int               `json:"total_pages"`
		}
		if err := h.Redis.GetCachedPropertyList(cacheKey, &cachedResponse); err == nil {
			// Cache hit
			return c.JSON(cachedResponse)
		}
	}

	// Parsear parámetros
	req := ListPropertiesRequest{
		TipoOperacion: c.Query("tipo_operacion"),
		TipoPropiedad: c.Query("tipo_propiedad"),
		Zona:          c.Query("zona"),
		Estado:        c.Query("estado", "activo"),
		Moneda:        c.Query("moneda"),
		Dormitorios:   c.Query("dormitorios"),
		Banos:         c.Query("banos"),
		Garaje:        c.Query("garaje"),
		SearchText:    c.Query("search"),
		OrderBy:       c.Query("order_by", "fecha"),
		Order:         c.Query("order", "desc"),
		Page:          c.QueryInt("page", 1),
		Limit:         c.QueryInt("limit", 20),
	}

	// Parsear parámetros numéricos
	if precioMin := c.Query("precio_min"); precioMin != "" {
		if val, err := strconv.ParseFloat(precioMin, 64); err == nil {
			req.PrecioMin = val
		}
	}
	if precioMax := c.Query("precio_max"); precioMax != "" {
		if val, err := strconv.ParseFloat(precioMax, 64); err == nil {
			req.PrecioMax = val
		}
	}
	if metrosMin := c.Query("metros_min"); metrosMin != "" {
		if val, err := strconv.ParseFloat(metrosMin, 64); err == nil {
			req.MetrosMin = val
		}
	}
	if metrosMax := c.Query("metros_max"); metrosMax != "" {
		if val, err := strconv.ParseFloat(metrosMax, 64); err == nil {
			req.MetrosMax = val
		}
	}

	// Parsear booleanos
	if aceptaPermuta := c.Query("acepta_permuta"); aceptaPermuta != "" {
		if val, err := strconv.ParseBool(aceptaPermuta); err == nil {
			req.AceptaPermuta = &val
		}
	}
	if financia := c.Query("financia"); financia != "" {
		if val, err := strconv.ParseBool(financia); err == nil {
			req.Financia = &val
		}
	}
	if condominio := c.Query("condominio"); condominio != "" {
		if val, err := strconv.ParseBool(condominio); err == nil {
			req.Condominio = &val
		}
	}
	if aptoOficina := c.Query("apto_oficina"); aptoOficina != "" {
		if val, err := strconv.ParseBool(aptoOficina); err == nil {
			req.AptoOficina = &val
		}
	}
	if viviendaSocial := c.Query("vivienda_social"); viviendaSocial != "" {
		if val, err := strconv.ParseBool(viviendaSocial); err == nil {
			req.ViviendaSocial = &val
		}
	}

	// 🔒 Validaciones
	// Validar página
	if req.Page < 1 {
		req.Page = 1
	}

	// Validar límite (máximo 100, mínimo 1)
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20
	}

	// Validar ordenamiento
	validOrderBy := map[string]bool{
		"precio": true, "fecha": true, "vistas": true, "metros": true,
	}
	if !validOrderBy[req.OrderBy] {
		req.OrderBy = "fecha"
	}

	// Validar dirección
	if req.Order != "asc" && req.Order != "desc" {
		req.Order = "desc"
	}

	// Validar estado
	validEstados := map[string]bool{
		"activo": true, "inactivo": true, "vendido": true,
	}
	if req.Estado != "" && !validEstados[req.Estado] {
		req.Estado = "activo"
	}

	// Validar tipo de operación
	if req.TipoOperacion != "" {
		validOps := map[string]bool{
			"venta": true, "alquiler": true,
		}
		if !validOps[req.TipoOperacion] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "tipo_operacion inválido. Debe ser: venta, alquiler",
			})
		}
	}

	// Validar moneda
	if req.Moneda != "" {
		if req.Moneda != "USD" && req.Moneda != "BOB" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "moneda inválida. Debe ser: USD, BOB",
			})
		}
	}

	// Construir parámetros para el servicio
	params := repository.ListPropertyParams{
		Estado:         req.Estado,
		TipoOperacion:  req.TipoOperacion,
		TipoPropiedad:  req.TipoPropiedad,
		Zona:           req.Zona,
		Moneda:         req.Moneda,
		PrecioMin:      req.PrecioMin,
		PrecioMax:      req.PrecioMax,
		MetrosMin:      req.MetrosMin,
		MetrosMax:      req.MetrosMax,
		Dormitorios:    req.Dormitorios,
		Banos:          req.Banos,
		Garaje:         req.Garaje,
		AceptaPermuta:  req.AceptaPermuta,
		Financia:       req.Financia,
		Condominio:     req.Condominio,
		AptoOficina:    req.AptoOficina,
		ViviendaSocial: req.ViviendaSocial,
		SearchText:     req.SearchText,
		OrderBy:        req.OrderBy,
		Order:          req.Order,
		Limit:          req.Limit,
		Offset:         (req.Page - 1) * req.Limit,
	}

	// Obtener propiedades a través del servicio
	properties, total, err := h.PropertyService.ListProperties(params)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo propiedades",
		})
	}

	// Calcular total de páginas
	totalPages := int(total) / req.Limit
	if int(total)%req.Limit > 0 {
		totalPages++
	}

	// Construir respuesta
	response := fiber.Map{
		"propiedades": properties,
		"total":       total,
		"page":        req.Page,
		"limit":       req.Limit,
		"total_pages": totalPages,
		"has_next":    req.Page < totalPages,
		"has_prev":    req.Page > 1,
	}

	// 🚀 Redis: Guardar en caché
	if h.Redis != nil {
		h.Redis.CachePropertyList(cacheKey, response)
	}

	return c.JSON(response)
}

// GetUserProperties - Lista todas las propiedades del usuario autenticado
func (h *PropertyHandler) GetUserProperties(c *fiber.Ctx) error {
	// Obtener usuario del contexto
	user := c.Locals("user").(*models.User)

	// Obtener propiedades del usuario a través del servicio
	properties, err := h.PropertyService.GetUserProperties(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo propiedades del usuario",
		})
	}

	return c.JSON(properties)
}

// SearchProperties - Búsqueda full-text en propiedades con filtros opcionales
func (h *PropertyHandler) SearchProperties(c *fiber.Ctx) error {
	// Obtener query de búsqueda
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "El parámetro 'q' es requerido para la búsqueda",
		})
	}

	// Parsear parámetros de filtros
	req := &service.FullTextSearchRequest{
		Query:         query,
		TipoOperacion: c.Query("tipo_operacion"),
		TipoPropiedad: c.Query("tipo_propiedad"),
		Moneda:        c.Query("moneda"),
		Page:          c.QueryInt("page", 1),
		Limit:         c.QueryInt("limit", 20),
	}

	// Parsear parámetros numéricos
	if precioMin := c.Query("precio_min"); precioMin != "" {
		if val, err := strconv.ParseFloat(precioMin, 64); err == nil {
			req.PrecioMin = val
		}
	}
	if precioMax := c.Query("precio_max"); precioMax != "" {
		if val, err := strconv.ParseFloat(precioMax, 64); err == nil {
			req.PrecioMax = val
		}
	}
	if metrosMin := c.Query("metros_min"); metrosMin != "" {
		if val, err := strconv.ParseFloat(metrosMin, 64); err == nil {
			req.MetrosMin = val
		}
	}
	if metrosMax := c.Query("metros_max"); metrosMax != "" {
		if val, err := strconv.ParseFloat(metrosMax, 64); err == nil {
			req.MetrosMax = val
		}
	}

	// Ejecutar búsqueda
	properties, total, err := h.PropertyService.FullTextSearch(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error ejecutando la búsqueda",
		})
	}

	// Si no hay resultados, retornar vacío
	if properties == nil {
		properties = []models.Property{}
	}

	// Calcular total de páginas
	totalPages := int(total) / req.Limit
	if int(total)%req.Limit > 0 {
		totalPages++
	}

	// Construir respuesta
	return c.JSON(fiber.Map{
		"query":        query,
		"propiedades":  properties,
		"total":        total,
		"page":         req.Page,
		"limit":        req.Limit,
		"total_pages":  totalPages,
		"has_next":     req.Page < totalPages,
		"has_prev":     req.Page > 1,
	})
}

// UpdatePropertyStatus - Actualiza el estado de una propiedad
func (h *PropertyHandler) UpdatePropertyStatus(c *fiber.Ctx) error {
	// Obtener usuario del contexto
	user := c.Locals("user").(*models.User)

	// Obtener ID de la propiedad
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de propiedad inválido",
		})
	}

	// Parsear request body
	var req struct {
		Estado string `json:"estado"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	// Validar estado
	nuevoEstado := models.PropertyStatus(req.Estado)
	switch nuevoEstado {
	case models.PropertyStatusActivo, models.PropertyStatusInactivo,
		models.PropertyStatusVendido:
		// Estados válidos
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Estado inválido. Debe ser: activo, inactivo o vendido",
		})
	}

	// Obtener propiedad primero para guardar estado anterior
	property, err := h.PropertyService.GetPropertyByID(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Propiedad no encontrada",
		})
	}

	// Guardar estado anterior para auditoría
	estadoAnterior := property.EstadoPropiedad

	// Actualizar estado a través del servicio
	reqService := &service.UpdatePropertyStatusRequest{
		PropertyID: uint(id),
		UserID:     user.ID,
		NewStatus:  nuevoEstado,
	}

	property, err = h.PropertyService.UpdatePropertyStatus(reqService)
	if err != nil {
		// Manejar diferentes tipos de errores
		if appErr, ok := err.(*errors.AppError); ok {
			if appErr.Code == errors.ErrCodeForbidden {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": appErr.Message,
				})
			}
			return c.Status(appErr.HTTPStatus).JSON(fiber.Map{
				"error": appErr.Message,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error actualizando la propiedad",
		})
	}

	// 🔒 Auditoría: Log de cambio de estado
	h.LogAudit(c, "update_property_status", "property", &property.ID, &user.ID, map[string]interface{}{
		"estado_anterior": string(estadoAnterior),
		"estado_nuevo":    string(nuevoEstado),
	})

	// 🚀 Redis: Invalidar caché y publicar actualización
	if h.Redis != nil {
		h.Redis.InvalidateProperty(property.ID)
		h.Redis.InvalidatePropertyLists()
		h.Redis.PublishPropertyUpdate("status_changed", property.ID, map[string]interface{}{
			"id":             property.ID,
			"estado_anterior": string(estadoAnterior),
			"estado_nuevo":    string(nuevoEstado),
		})
	}

	return c.JSON(fiber.Map{
		"message":  "Estado actualizado exitosamente",
		"property": property,
	})
}

// DeleteProperty - Elimina (soft delete) una propiedad
func (h *PropertyHandler) DeleteProperty(c *fiber.Ctx) error {
	// Obtener usuario del contexto
	user := c.Locals("user").(*models.User)

	// Obtener ID de la propiedad
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de propiedad inválido",
		})
	}

	// Obtener propiedad primero para guardar detalles para auditoría
	property, err := h.PropertyService.GetPropertyByID(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Propiedad no encontrada",
		})
	}

	// Guardar detalles para auditoría
	propertyID := property.ID
	propertyDetails := map[string]interface{}{
		"titulo": property.Titulo,
		"precio": property.Precio,
		"moneda": property.Moneda,
	}

	// Eliminar a través del servicio
	reqService := &service.DeletePropertyRequest{
		PropertyID: uint(id),
		UserID:     user.ID,
	}

	if err := h.PropertyService.DeleteProperty(reqService); err != nil {
		// Manejar diferentes tipos de errores
		if appErr, ok := err.(*errors.AppError); ok {
			if appErr.Code == errors.ErrCodeForbidden {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": appErr.Message,
				})
			}
			return c.Status(appErr.HTTPStatus).JSON(fiber.Map{
				"error": appErr.Message,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error eliminando la propiedad",
		})
	}

	// 🔒 Auditoría: Log de eliminación
	h.LogAudit(c, "delete_property", "property", &propertyID, &user.ID, propertyDetails)

	// 🚀 Redis: Invalidar caché y publicar actualización
	if h.Redis != nil {
		h.Redis.InvalidateProperty(propertyID)
		h.Redis.InvalidatePropertyLists()
		h.Redis.InvalidateUserPropertyCount(user.ID)
		h.Redis.PublishPropertyUpdate("deleted", propertyID, propertyDetails)
	}

	return c.JSON(fiber.Map{
		"message": "Propiedad eliminada exitosamente",
	})
}
