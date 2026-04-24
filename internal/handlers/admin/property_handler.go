package admin

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/repository"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/service"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/microcosm-cc/bluemonday"
	"gorm.io/gorm"
)

// PropertyHandler - Handler para administración de propiedades
type PropertyHandler struct {
	PropertyService *service.PropertyService
	Redis           *config.RedisClient
}

// NewPropertyHandler - Crea un nuevo handler de propiedades para admin
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

	// Inicializar Redis (opcional)
	var redis *config.RedisClient
	redis, err = config.NewRedis()
	if err != nil {
		fmt.Println("⚠️  Redis no disponible, ejecutando sin caché")
		redis = nil
	}

	return &PropertyHandler{
		PropertyService: propertyService,
		Redis:           redis,
	}, nil
}

// ListProperties - Lista todas las propiedades con filtros avanzados
func (h *PropertyHandler) ListProperties(c *fiber.Ctx) error {
	db, err := config.NewDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	// Parsear query params
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	estado := c.Query("estado", "")
	tipoOperacion := c.Query("tipo_operacion", "")
	tipoPropiedad := c.Query("tipo_propiedad", "")
	zona := c.Query("zona", "")
	userID := c.Query("user_id", "")
	advisorID := c.Query("advisor_id", "")
	search := c.Query("search", "")

	offset := (page - 1) * limit

	// Construir query con preload de relaciones
	query := db.DB.Model(&models.Property{}).
		Preload("User").
		Preload("PropertyImages").
		Preload("Advisor").
		Preload("PropertyAdvisors.Advisor") // Incluir múltiples asesores asignados

	// Filtros
	if estado != "" {
		query = query.Where("estado_propiedad = ?", estado)
	}
	if tipoOperacion != "" {
		query = query.Where("tipo_operacion = ?", tipoOperacion)
	}
	if tipoPropiedad != "" {
		query = query.Where("tipo_propiedad = ?", tipoPropiedad)
	}
	if zona != "" {
		query = query.Where("zona ILIKE ?", "%"+zona+"%")
	}
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if advisorID != "" {
		query = query.Where("advisor_id = ?", advisorID)
	}
	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("titulo ILIKE ? OR zona ILIKE ? OR vendedor_nombre ILIKE ?",
			searchTerm, searchTerm, searchTerm)
	}

	// Contar total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		fmt.Printf("❌ ERROR contando propiedades: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error contando propiedades",
		})
	}

	// Obtener propiedades
	var properties []models.Property
	if err := query.Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&properties).Error; err != nil {
		fmt.Printf("❌ ERROR obteniendo propiedades: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo propiedades",
		})
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return c.JSON(fiber.Map{
		"properties":  properties,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}

// GetProperty - Obtiene una propiedad con todas sus relaciones
func (h *PropertyHandler) GetProperty(c *fiber.Ctx) error {
	id := c.Params("id")

	idUint, _ := strconv.ParseUint(id, 10, 32)
	property, err := h.PropertyService.GetPropertyByIDWithRelations(uint(idUint))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Propiedad no encontrada",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo propiedad",
		})
	}

	return c.JSON(property)
}

// CreatePropertyRequest - Request para crear propiedad como admin
type CreatePropertyRequest struct {
	UserID *uint `json:"user_id"` // Opcional, si no se usa el del admin
	// Resto igual que CreatePropertyRequest
	Titulo          string   `json:"titulo"`
	TipoOperacion   string   `json:"tipo_operacion"`
	Precio          float64  `json:"precio"`
	Moneda          string   `json:"moneda"`
	AceptaPermuta   bool     `json:"acepta_permuta"`
	Financia        bool     `json:"financia"`
	TipoPropiedad   string   `json:"tipo_propiedad"`
	Direccion       string   `json:"direccion"`
	Zona            string   `json:"zona"`
	TipoMarcador    string   `json:"tipo_marcador"`
	Latitud         *float64 `json:"latitud"`
	Longitud        *float64 `json:"longitud"`
	MostrarDireccion bool    `json:"mostrar_direccion"`
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
	Confort         []string `json:"confort"`
	Seguridad       []string `json:"seguridad"`
	NoAplicaConfort bool     `json:"no_aplica_confort"`
	Imagenes        []struct {
		URL   string `json:"url"`
		Orden int    `json:"orden"`
	} `json:"imagenes"`
	VideoURL            string  `json:"video_url"`
	Descripcion         string  `json:"descripcion"`
	TerminosCondiciones string  `json:"terminos_condiciones"`
	AdvisorID           *uint   `json:"advisor_id"`
}

// CreateProperty - Crea una propiedad como admin
func (h *PropertyHandler) CreateProperty(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	var req CreatePropertyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	// Sanitización HTML
	policy := bluemonday.UGCPolicy()
	policy.AllowElements("a", "b", "strong", "em", "i", "u", "ul", "ol", "li", "p", "br", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote", "span", "div")
	policy.AllowAttrs("href").OnElements("a")
	policy.AllowAttrs("class").OnElements("span", "div", "p")
	policy.AllowAttrs("target").OnElements("a")
	policy.RequireNoFollowOnLinks(true)

	// Preparar imágenes
	imagenes := make([]service.CreatePropertyImageRequest, len(req.Imagenes))
	for i, img := range req.Imagenes {
		imagenes[i] = service.CreatePropertyImageRequest{
			URL:   img.URL,
			Orden: img.Orden,
		}
	}

	// Usar ID del usuario especificado o el del admin
	userID := user.ID
	if req.UserID != nil {
		userID = *req.UserID
	}

	// Preparar request para el servicio
	serviceReq := &service.CreatePropertyRequest{
		UserID:              userID,
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
		AdvisorID:           req.AdvisorID,
	}

	// Crear propiedad directamente activa (como admin)
	property, err := h.PropertyService.CreatePropertyAsAdmin(serviceReq)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			return c.Status(appErr.HTTPStatus).JSON(fiber.Map{
				"error": appErr.Message,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creando la propiedad",
		})
	}

	// Recargar con relaciones
	property, err = h.PropertyService.GetPropertyByIDWithRelations(property.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo la propiedad creada",
		})
	}

	h.LogAudit(c, "admin_create_property", "property", &property.ID, &user.ID, map[string]interface{}{
		"titulo":         property.Titulo,
		"precio":         property.Precio,
		"moneda":         property.Moneda,
		"tipo_operacion": property.TipoOperacion,
	})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":  "Propiedad creada exitosamente",
		"property": property,
	})
}

// UpdatePropertyRequest - Request para actualizar propiedad
type UpdatePropertyRequest struct {
	Titulo              *string   `json:"titulo"`
	TipoOperacion       *string   `json:"tipo_operacion"`
	Precio              *float64  `json:"precio"`
	Moneda              *string   `json:"moneda"`
	AceptaPermuta       *bool     `json:"acepta_permuta"`
	Financia            *bool     `json:"financia"`
	TipoPropiedad       *string   `json:"tipo_propiedad"`
	Direccion           *string   `json:"direccion"`
	Zona                *string   `json:"zona"`
	TipoMarcador        *string   `json:"tipo_marcador"`
	Latitud             *float64  `json:"latitud"`
	Longitud            *float64  `json:"longitud"`
	MostrarDireccion    *bool     `json:"mostrar_direccion"`
	Dormitorios         *string   `json:"dormitorios"`
	Banos               *string   `json:"banos"`
	Garaje              *string   `json:"garaje"`
	MetrosEdificados    *float64  `json:"metros_edificados"`
	MetrosTerreno       *float64  `json:"metros_terreno"`
	Estado              *string   `json:"estado"`
	AnoConstruccion     *int      `json:"ano_construccion"`
	AptoOficina         *bool     `json:"apto_oficina"`
	ViviendaSocial      *bool     `json:"vivienda_social"`
	Condominio          *bool     `json:"condominio"`
	CantidadPisos       *int      `json:"cantidad_pisos"`
	Confort             *[]string `json:"confort"`
	Seguridad           *[]string `json:"seguridad"`
	NoAplicaConfort     *bool     `json:"no_aplica_confort"`
	VideoURL            *string   `json:"video_url"`
	Descripcion         *string   `json:"descripcion"`
	TerminosCondiciones *string   `json:"terminos_condiciones"`
	AdvisorID           *uint     `json:"advisor_id"`
}

// UpdateProperty - Actualiza una propiedad
func (h *PropertyHandler) UpdateProperty(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)
	id := c.Params("id")

	var req UpdatePropertyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	// Sanitización HTML
	policy := bluemonday.UGCPolicy()
	policy.AllowElements("a", "b", "strong", "em", "i", "u", "ul", "ol", "li", "p", "br", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote", "span", "div")
	policy.AllowAttrs("href").OnElements("a")
	policy.AllowAttrs("class").OnElements("span", "div", "p")

	updateData := make(map[string]interface{})

	if req.Titulo != nil {
		updateData["titulo"] = *req.Titulo
	}
	if req.TipoOperacion != nil {
		updateData["tipo_operacion"] = *req.TipoOperacion
	}
	if req.Precio != nil {
		updateData["precio"] = *req.Precio
	}
	if req.Moneda != nil {
		updateData["moneda"] = *req.Moneda
	}
	if req.AceptaPermuta != nil {
		updateData["acepta_permuta"] = *req.AceptaPermuta
	}
	if req.Financia != nil {
		updateData["financia"] = *req.Financia
	}
	if req.TipoPropiedad != nil {
		updateData["tipo_propiedad"] = *req.TipoPropiedad
	}
	if req.Direccion != nil {
		updateData["direccion"] = *req.Direccion
	}
	if req.Zona != nil {
		updateData["zona"] = *req.Zona
	}
	if req.TipoMarcador != nil {
		updateData["tipo_marcador"] = *req.TipoMarcador
	}
	if req.Latitud != nil {
		updateData["latitud"] = *req.Latitud
	}
	if req.Longitud != nil {
		updateData["longitud"] = *req.Longitud
	}
	if req.MostrarDireccion != nil {
		updateData["mostrar_direccion"] = *req.MostrarDireccion
	}
	if req.Dormitorios != nil {
		updateData["dormitorios"] = *req.Dormitorios
	}
	if req.Banos != nil {
		updateData["banos"] = *req.Banos
	}
	if req.Garaje != nil {
		updateData["garaje"] = *req.Garaje
	}
	if req.MetrosEdificados != nil {
		updateData["metros_edificados"] = *req.MetrosEdificados
	}
	if req.MetrosTerreno != nil {
		updateData["metros_terreno"] = *req.MetrosTerreno
	}
	if req.Estado != nil {
		updateData["estado"] = *req.Estado
	}
	if req.AnoConstruccion != nil {
		updateData["ano_construccion"] = *req.AnoConstruccion
	}
	if req.AptoOficina != nil {
		updateData["apto_oficina"] = *req.AptoOficina
	}
	if req.ViviendaSocial != nil {
		updateData["vivienda_social"] = *req.ViviendaSocial
	}
	if req.Condominio != nil {
		updateData["condominio"] = *req.Condominio
	}
	if req.CantidadPisos != nil {
		updateData["cantidad_pisos"] = *req.CantidadPisos
	}
	if req.NoAplicaConfort != nil {
		updateData["no_aplica_confort"] = *req.NoAplicaConfort
	}
	if req.VideoURL != nil {
		updateData["video_url"] = *req.VideoURL
	}
	if req.Descripcion != nil {
		updateData["descripcion"] = policy.Sanitize(strings.TrimSpace(*req.Descripcion))
	}
	if req.TerminosCondiciones != nil {
		updateData["terminos_condiciones"] = policy.Sanitize(strings.TrimSpace(*req.TerminosCondiciones))
	}
	if req.AdvisorID != nil {
		updateData["advisor_id"] = *req.AdvisorID
	}

	idUint, _ := strconv.ParseUint(id, 10, 32)
	property, err := h.PropertyService.UpdateProperty(uint(idUint), updateData)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			return c.Status(appErr.HTTPStatus).JSON(fiber.Map{
				"error": appErr.Message,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error actualizando la propiedad",
		})
	}

	h.LogAudit(c, "admin_update_property", "property", &property.ID, &user.ID, updateData)

	return c.JSON(fiber.Map{
		"message":  "Propiedad actualizada exitosamente",
		"property": property,
	})
}

// UpdatePropertyStatusRequest - Request para cambiar estado
type UpdatePropertyStatusRequest struct {
	Estado string `json:"estado"`
}

// UpdatePropertyStatus - Actualiza el estado de una propiedad
func (h *PropertyHandler) UpdatePropertyStatus(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)
	id := c.Params("id")

	var req UpdatePropertyStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	// Validar estado
	validEstados := map[string]bool{
		"activo":    true,
		"inactivo":  true,
		"pendiente": true,
		"vendido":   true,
		"rechazado": true,
	}

	if !validEstados[req.Estado] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Estado no válido",
		})
	}

	idUint, _ := strconv.ParseUint(id, 10, 32)
	property, err := h.PropertyService.UpdatePropertyStatusAdmin(uint(idUint), req.Estado)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			return c.Status(appErr.HTTPStatus).JSON(fiber.Map{
				"error": appErr.Message,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error actualizando estado",
		})
	}

	h.LogAudit(c, "admin_update_property_status", "property", &property.ID, &user.ID, map[string]interface{}{
		"estado": req.Estado,
	})

	return c.JSON(fiber.Map{
		"message":  "Estado actualizado exitosamente",
		"property": property,
	})
}

// ToggleFeaturedRequest - Request para destacar
type ToggleFeaturedRequest struct {
	Destacado bool `json:"destacado"`
}

// ToggleFeatured - Marca/quita destacado
func (h *PropertyHandler) ToggleFeatured(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)
	id := c.Params("id")

	var req ToggleFeaturedRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	idUint, _ := strconv.ParseUint(id, 10, 32)
	property, err := h.PropertyService.ToggleFeatured(uint(idUint), req.Destacado)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			return c.Status(appErr.HTTPStatus).JSON(fiber.Map{
				"error": appErr.Message,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error actualizando destacado",
		})
	}

	h.LogAudit(c, "admin_toggle_featured", "property", &property.ID, &user.ID, map[string]interface{}{
		"destacado": req.Destacado,
	})

	return c.JSON(fiber.Map{
		"message":  "Propiedad actualizada exitosamente",
		"property": property,
	})
}

// DeleteProperty - Elimina una propiedad (soft delete)
func (h *PropertyHandler) DeleteProperty(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)
	id := c.Params("id")

	idUint, _ := strconv.ParseUint(id, 10, 32)
	err := h.PropertyService.DeletePropertyAdmin(uint(idUint))
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			return c.Status(appErr.HTTPStatus).JSON(fiber.Map{
				"error": appErr.Message,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error eliminando la propiedad",
		})
	}

	h.LogAudit(c, "admin_delete_property", "property", &id, &user.ID, nil)

	return c.JSON(fiber.Map{
		"message": "Propiedad eliminada exitosamente",
	})
}

// AssignAdvisorRequest - Request para asignar asesor
type AssignAdvisorRequest struct {
	AdvisorID *uint `json:"advisor_id"`
}

// AssignAdvisor - Asigna o remueve asesor de una propiedad
func (h *PropertyHandler) AssignAdvisor(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)
	id := c.Params("id")

	var req AssignAdvisorRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	idUint, _ := strconv.ParseUint(id, 10, 32)
	property, err := h.PropertyService.AssignAdvisor(uint(idUint), req.AdvisorID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			return c.Status(appErr.HTTPStatus).JSON(fiber.Map{
				"error": appErr.Message,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error asignando asesor",
		})
	}

	h.LogAudit(c, "admin_assign_advisor", "property", &property.ID, &user.ID, map[string]interface{}{
		"advisor_id": req.AdvisorID,
	})

	return c.JSON(fiber.Map{
		"message":  "Asesor asignado exitosamente",
		"property": property,
	})
}

// GetPropertiesStats - Obtiene estadísticas de propiedades
func (h *PropertyHandler) GetPropertiesStats(c *fiber.Ctx) error {
	db, err := config.NewDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	periodo := c.Query("periodo", "30d")

	_ = periodo // Se usará en el futuro para filtrar por fecha
	now := time.Now()

	// Stats
	var total, activos, pendientes, vendidos, rechazados, publicadosHoy, ventasMes int64

	db.DB.Model(&models.Property{}).Count(&total)
	db.DB.Model(&models.Property{}).Where("estado_propiedad = ?", "activo").Count(&activos)
	db.DB.Model(&models.Property{}).Where("estado_propiedad = ?", "pendiente").Count(&pendientes)
	db.DB.Model(&models.Property{}).Where("estado_propiedad = ?", "vendido").Count(&vendidos)
	db.DB.Model(&models.Property{}).Where("estado_propiedad = ?", "rechazado").Count(&rechazados)

	// Publicadas hoy
	hoyInicio := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	db.DB.Model(&models.Property{}).Where("created_at >= ?", hoyInicio).Count(&publicadosHoy)

	// Vendidas este mes
	mesInicio := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	db.DB.Model(&models.Property{}).
		Where("estado_propiedad = ? AND updated_at >= ?", "vendido", mesInicio).
		Count(&ventasMes)

	return c.JSON(fiber.Map{
		"total":          total,
		"activos":        activos,
		"pendientes":     pendientes,
		"vendidos":       vendidos,
		"rechazados":     rechazados,
		"publicados_hoy": publicadosHoy,
		"ventas_mes":     ventasMes,
	})
}

// LogAudit - Helper para registrar auditoría
func (h *PropertyHandler) LogAudit(c *fiber.Ctx, action, entityType string, entityID, userID interface{}, metadata map[string]interface{}) {
	db, err := config.NewDatabase()
	if err != nil {
		return
	}

	auditRepo := repository.NewAuditRepository(db)

	userIDPtr, ok := userID.(*uint)
	if !ok {
		return
	}

	entityIDPtr, ok := entityID.(*uint)
	if !ok {
		return
	}

	// Convertir metadata a JSON string
	detailsJSON := ""
	if metadata != nil {
		jsonBytes, err := json.Marshal(metadata)
		if err == nil {
			detailsJSON = string(jsonBytes)
		}
	}

	audit := &models.AuditLog{
		UserID:     userIDPtr,
		Action:     action,
		EntityID:   entityIDPtr,
		Entity:     entityType,
		Details:    detailsJSON,
	}

	_ = auditRepo.Create(audit)
}
