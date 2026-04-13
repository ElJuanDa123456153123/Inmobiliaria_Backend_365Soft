package handlers

import (
	"strconv"
	"strings"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/dto"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/services"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/validators"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/response"
	apperrors "github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/errors"
	"github.com/gofiber/fiber/v2"
)

// PropertyHandler - Handler para propiedades
type PropertyHandler struct {
	propertyService  services.PropertyService
	searchService    services.SearchService
	storageService   services.StorageService
	cacheService     services.CacheService
	validator        *validators.PropertyValidator
}

// NewPropertyHandler - Crea una nueva instancia del handler
func NewPropertyHandler(
	propertyService services.PropertyService,
	searchService services.SearchService,
	storageService services.StorageService,
	cacheService services.CacheService,
) *PropertyHandler {
	return &PropertyHandler{
		propertyService: propertyService,
		searchService:   searchService,
		storageService:  storageService,
		cacheService:    cacheService,
		validator:       validators.NewPropertyValidator(),
	}
}

// RegisterRoutes - Registra las rutas de propiedades
func (h *PropertyHandler) RegisterRoutes(app *fiber.App) {
	// Grupo de rutas de propiedades
	api := app.Group("/api/propiedades")

	// Rutas públicas (sin autenticación)
	api.Get("/", h.ListProperties)
	api.Get("/destacadas", h.GetFeaturedProperties)
	api.Get("/relacionadas/:id", h.GetRelatedProperties)
	api.Get("/:id", h.GetProperty)

	// Rutas con autenticación requerida
	api.Post("/", h.AuthMiddleware(), h.CreateProperty)
	api.Put("/:id", h.AuthMiddleware(), h.UpdateProperty)
	api.Delete("/:id", h.AuthMiddleware(), h.DeleteProperty)

	// Rutas de fotos (requieren autenticación)
	api.Post("/:id/fotos", h.AuthMiddleware(), h.UploadPhotos)
	api.Put("/:id/fotos/:foto_id/portada", h.AuthMiddleware(), h.SetPhotoAsPortada)
	api.Put("/:id/fotos/reordenar", h.AuthMiddleware(), h.ReorderPhotos)
	api.Delete("/:id/fotos/:foto_id", h.AuthMiddleware(), h.DeletePhoto)

	// Rutas de búsqueda (sin autenticación)
	api.Get("/buscar", h.SearchProperties)

	// Rutas de categorías
	api.Post("/:id/categorias/:category_id", h.AuthMiddleware(), h.AddCategory)
	api.Delete("/:id/categorias/:category_id", h.AuthMiddleware(), h.RemoveCategory)

	// Rutas de usuario
	api.Get("/mis-propiedades", h.AuthMiddleware(), h.GetMyProperties)
}

// ========== ENDPOINTS PÚBLICOS ==========

// ListProperties - GET /api/propiedades - Lista propiedades con filtros
func (h *PropertyHandler) ListProperties(c *fiber.Ctx) error {
	// 1. Parsear filtros
	filter := h.parsePropertyFilter(c)


	// 3. Listar propiedades
	list, err := h.propertyService.ListProperties(c.Context(), filter)
	if err != nil {
		return h.handleError(c, err)
	}


	return c.JSON(list)
}

// GetProperty - GET /api/propiedades/:id - Obtiene detalle de propiedad
func (h *PropertyHandler) GetProperty(c *fiber.Ctx) error {
	// 1. Obtener ID
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(response.ErrorWithMessage(apperrors.ErrCodeValidation, "ID inválido"))
	}

	// 2. Verificar si hay usuario autenticado
	user := c.Locals("user")
	includeContacto := user != nil

	// 3. Obtener propiedad
	response, err := h.propertyService.GetProperty(c.Context(), uint(id), includeContacto)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(response)
}

// GetFeaturedProperties - GET /api/propiedades/destacadas - Propiedades destacadas
func (h *PropertyHandler) GetFeaturedProperties(c *fiber.Ctx) error {
	// 1. Obtener límite (default 10)
	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	// 2. Obtener destacadas
	properties, err := h.propertyService.GetFeaturedProperties(c.Context(), limit)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(properties)
}

// GetRelatedProperties - GET /api/propiedades/relacionadas/:id - Propiedades relacionadas
func (h *PropertyHandler) GetRelatedProperties(c *fiber.Ctx) error {
	// 1. Obtener ID
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(response.ErrorWithMessage(apperrors.ErrCodeValidation, "ID inválido"))
	}

	// 2. Obtener límite (default 4)
	limit := 4
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 10 {
			limit = l
		}
	}

	// 3. Obtener relacionadas
	properties, err := h.propertyService.GetRelatedProperties(c.Context(), uint(id), limit)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(properties)
}

// SearchProperties - GET /api/propiedades/buscar - Búsqueda de propiedades
func (h *PropertyHandler) SearchProperties(c *fiber.Ctx) error {
	// 1. Obtener query
	query := c.Query("q")

	// 2. Parsear filtros
	filter := h.parsePropertyFilter(c)

	// 3. Buscar
	results, err := h.searchService.Search(c.Context(), query, filter)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(results)
}

// ========== ENDPOINTS PRIVADOS (Requieren Auth) ==========

// CreateProperty - POST /api/propiedades - Crea nueva propiedad
func (h *PropertyHandler) CreateProperty(c *fiber.Ctx) error {
	// 1. Obtener usuario autenticado
	user := c.Locals("user").(*models.User)
	if user == nil {
		return c.Status(401).JSON(dto.UnauthorizedResponse())
	}

	// 2. Parsear request body
	var req dto.CreatePropertyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(response.ErrorWithMessage(apperrors.ErrCodeValidation, "Error parseando request"))
	}

	// 3. Crear propiedad
	response, err := h.propertyService.CreateProperty(c.Context(), &req, user.ID)
	if err != nil {
		return h.handleError(c, err)
	}

	// 4. Invalidar caché
	_ = h.cacheService.InvalidatePropertyList(c.Context())

	return c.Status(201).JSON(dto.SuccessResponseWithMessage(
		response,
		"Propiedad creada exitosamente",
	))
}

// UpdateProperty - PUT /api/propiedades/:id - Actualiza propiedad
func (h *PropertyHandler) UpdateProperty(c *fiber.Ctx) error {
	// 1. Obtener usuario
	user := c.Locals("user").(*models.User)
	if user == nil {
		return c.Status(401).JSON(dto.UnauthorizedResponse())
	}

	// 2. Obtener ID
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(response.ErrorWithMessage(apperrors.ErrCodeValidation, "ID inválido"))
	}

	// 3. Parsear request body
	var req dto.UpdatePropertyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(response.ErrorWithMessage(apperrors.ErrCodeValidation, "Error parseando request"))
	}

	// 4. Actualizar propiedad
	response, err := h.propertyService.UpdateProperty(c.Context(), uint(id), &req, user.ID)
	if err != nil {
		return h.handleError(c, err)
	}

	// 5. Invalidar caché
	_ = h.cacheService.InvalidateProperty(c.Context(), uint(id))
	_ = h.cacheService.InvalidatePropertyList(c.Context())

	return c.JSON(dto.SuccessResponseWithMessage(
		response,
		"Propiedad actualizada exitosamente",
	))
}

// DeleteProperty - DELETE /api/propiedades/:id - Elimina propiedad
func (h *PropertyHandler) DeleteProperty(c *fiber.Ctx) error {
	// 1. Obtener usuario
	user := c.Locals("user").(*models.User)
	if user == nil {
		return c.Status(401).JSON(dto.UnauthorizedResponse())
	}

	// 2. Obtener ID
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(response.ErrorWithMessage(apperrors.ErrCodeValidation, "ID inválido"))
	}

	// 3. Eliminar propiedad
	if err := h.propertyService.DeleteProperty(c.Context(), uint(id), user.ID); err != nil {
		return h.handleError(c, err)
	}

	// 4. Invalidar caché
	_ = h.cacheService.InvalidateProperty(c.Context(), uint(id))
	_ = h.cacheService.InvalidatePropertyList(c.Context())

	return c.JSON(dto.SuccessResponseWithMessage(
		nil,
		"Propiedad eliminada exitosamente",
	))
}

// GetMyProperties - GET /api/propiedades/mis-propiedades - Propiedades del usuario
func (h *PropertyHandler) GetMyProperties(c *fiber.Ctx) error {
	// 1. Obtener usuario
	user := c.Locals("user").(*models.User)
	if user == nil {
		return c.Status(401).JSON(dto.UnauthorizedResponse())
	}

	// 2. Parsear paginación
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	// 3. Listar propiedades
	list, err := h.propertyService.ListUserProperties(c.Context(), user.ID, page, limit)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(list)
}

// ========== ENDPOINTS DE FOTOS ==========

// UploadPhotos - POST /api/propiedades/:id/fotos - Sube fotos
func (h *PropertyHandler) UploadPhotos(c *fiber.Ctx) error {
	// 1. Obtener usuario
	user := c.Locals("user").(*models.User)
	if user == nil {
		return c.Status(401).JSON(dto.UnauthorizedResponse())
	}

	// 2. Obtener ID
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(response.ErrorWithMessage(apperrors.ErrCodeValidation, "ID inválido"))
	}

	// 3. Parsear formulario multipart
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).JSON(response.ErrorWithMessage(apperrors.ErrCodeValidation, "Error parseando formulario"))
	}

	// 4. Obtener archivos
	files := form.File["files"]
	if len(files) == 0 {
		return c.Status(400).JSON(response.ErrorWithMessage(apperrors.ErrCodeValidation, "No se enviaron archivos"))
	}

	// 5. Validar cantidad (3-7)
	if len(files) < 3 {
		// Warning, pero continuar
	} else if len(files) > 7 {
		return c.Status(400).JSON(response.ErrorWithMessage(apperrors.ErrCodeValidation, "Máximo 7 fotos permitidas"))
	}

	// 6. Procesar archivos
	photoFiles := make([]dto.PhotoFile, 0, len(files))
	for _, fileHeader := range files {
		// Abrir archivo
		file, err := fileHeader.Open()
		if err != nil {
			return c.Status(500).JSON(dto.InternalErrorResponse())
		}
		defer file.Close()

		// Leer contenido
		content := make([]byte, fileHeader.Size)
		if _, err := file.Read(content); err != nil {
			return c.Status(500).JSON(dto.InternalErrorResponse())
		}

		// Detectar MIME type
		mimeType := fileHeader.Header.Get("Content-Type")
		if mimeType == "" {
			ext := strings.ToLower(fileHeader.Filename)
			switch ext {
			case ".jpg", ".jpeg":
				mimeType = "image/jpeg"
			case ".png":
				mimeType = "image/png"
			case ".webp":
				mimeType = "image/webp"
			default:
				return c.Status(400).JSON(response.ErrorWithMessage(apperrors.ErrCodeValidation, "Formato de archivo inválido"))
			}
		}

		// Crear PhotoFile
		photoFiles = append(photoFiles, dto.PhotoFile{
			Filename: fileHeader.Filename,
			Size:     fileHeader.Size,
			Content:  content,
			MimeType: mimeType,
		})
	}

	// 7. Crear request
	req := &dto.UploadPhotosRequest{
		Files: photoFiles,
	}

	// 8. Subir fotos
	response, err := h.propertyService.UploadPhotos(c.Context(), uint(id), req, user.ID)
	if err != nil {
		return h.handleError(c, err)
	}

	// 9. Invalidar caché
	_ = h.cacheService.InvalidateProperty(c.Context(), uint(id))

	return c.JSON(dto.SuccessResponseWithMessage(
		response,
		"Fotos subidas exitosamente",
	))
}

// SetPhotoAsPortada - PUT /api/propiedades/:id/fotos/:foto_id/portada - Marca foto como portada
func (h *PropertyHandler) SetPhotoAsPortada(c *fiber.Ctx) error {
	// 1. Obtener usuario
	user := c.Locals("user").(*models.User)
	if user == nil {
		return c.Status(401).JSON(dto.UnauthorizedResponse())
	}

	// 2. Obtener IDs
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(response.ErrorWithMessage(apperrors.ErrCodeValidation, "ID de propiedad inválido"))
	}

	fotoID, err := strconv.ParseUint(c.Params("foto_id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(response.ErrorWithMessage(apperrors.ErrCodeValidation, "ID de foto inválido"))
	}

	// 3. Marcar portada
	if err := h.propertyService.SetPhotoAsPortada(c.Context(), uint(id), uint(fotoID), user.ID); err != nil {
		return h.handleError(c, err)
	}

	// 4. Invalidar caché
	_ = h.cacheService.InvalidateProperty(c.Context(), uint(id))

	return c.JSON(dto.SuccessResponseWithMessage(
		nil,
		"Foto marcada como portada",
	))
}

// ReorderPhotos - PUT /api/propiedades/:id/fotos/reordenar - Reordena fotos
func (h *PropertyHandler) ReorderPhotos(c *fiber.Ctx) error {
	// 1. Obtener usuario
	user := c.Locals("user").(*models.User)
	if user == nil {
		return c.Status(401).JSON(dto.UnauthorizedResponse())
	}

	// 2. Obtener ID
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(response.ErrorWithMessage(apperrors.ErrCodeValidation, "ID inválido"))
	}

	// 3. Parsear request
	var req dto.ReorderPhotosRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(response.ErrorWithMessage(apperrors.ErrCodeValidation, "Error parseando request"))
	}

	// 4. Convertir a uint slice
	orden := make([]uint, len(req.Orden))
	for i, id := range req.Orden {
		orden[i] = uint(id)
	}

	// 5. Reordenar
	if err := h.propertyService.ReorderPhotos(c.Context(), uint(id), orden, user.ID); err != nil {
		return h.handleError(c, err)
	}

	// 6. Invalidar caché
	_ = h.cacheService.InvalidateProperty(c.Context(), uint(id))

	return c.JSON(dto.SuccessResponseWithMessage(
		nil,
		"Fotos reordenadas exitosamente",
	))
}

// DeletePhoto - DELETE /api/propiedades/:id/fotos/:foto_id - Elimina foto
func (h *PropertyHandler) DeletePhoto(c *fiber.Ctx) error {
	// 1. Obtener usuario
	user := c.Locals("user").(*models.User)
	if user == nil {
		return c.Status(401).JSON(dto.UnauthorizedResponse())
	}

	// 2. Obtener IDs
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(response.ErrorWithMessage(apperrors.ErrCodeValidation, "ID de propiedad inválido"))
	}

	fotoID, err := strconv.ParseUint(c.Params("foto_id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(response.ErrorWithMessage(apperrors.ErrCodeValidation, "ID de foto inválido"))
	}

	// 3. Eliminar foto
	if err := h.propertyService.DeletePhoto(c.Context(), uint(id), uint(fotoID), user.ID); err != nil {
		return h.handleError(c, err)
	}

	// 4. Invalidar caché
	_ = h.cacheService.InvalidateProperty(c.Context(), uint(id))

	return c.JSON(dto.SuccessResponseWithMessage(
		nil,
		"Foto eliminada exitosamente",
	))
}

// ========== ENDPOINTS DE CATEGORÍAS ==========

// AddCategory - POST /api/propiedades/:id/categorias/:category_id - Añade categoría
func (h *PropertyHandler) AddCategory(c *fiber.Ctx) error {
	// 1. Obtener usuario
	user := c.Locals("user").(*models.User)
	if user == nil {
		return c.Status(401).JSON(dto.UnauthorizedResponse())
	}

	// 2. Obtener IDs
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(response.ErrorWithMessage(apperrors.ErrCodeValidation, "ID de propiedad inválido"))
	}

	categoryID, err := strconv.ParseUint(c.Params("category_id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(response.ErrorWithMessage(apperrors.ErrCodeValidation, "ID de categoría inválido"))
	}

	// 3. Añadir categoría
	if err := h.propertyService.AddCategory(c.Context(), uint(id), uint(categoryID), user.ID); err != nil {
		return h.handleError(c, err)
	}

	// 4. Invalidar caché
	_ = h.cacheService.InvalidateProperty(c.Context(), uint(id))
	_ = h.cacheService.InvalidateCategories(c.Context())

	return c.JSON(dto.SuccessResponseWithMessage(
		nil,
		"Categoría añadida exitosamente",
	))
}

// RemoveCategory - DELETE /api/propiedades/:id/categorias/:category_id - Elimina categoría
func (h *PropertyHandler) RemoveCategory(c *fiber.Ctx) error {
	// 1. Obtener usuario
	user := c.Locals("user").(*models.User)
	if user == nil {
		return c.Status(401).JSON(dto.UnauthorizedResponse())
	}

	// 2. Obtener IDs
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(response.ErrorWithMessage(apperrors.ErrCodeValidation, "ID de propiedad inválido"))
	}

	categoryID, err := strconv.ParseUint(c.Params("category_id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(response.ErrorWithMessage(apperrors.ErrCodeValidation, "ID de categoría inválido"))
	}

	// 3. Eliminar categoría
	if err := h.propertyService.RemoveCategory(c.Context(), uint(id), uint(categoryID), user.ID); err != nil {
		return h.handleError(c, err)
	}

	// 4. Invalidar caché
	_ = h.cacheService.InvalidateProperty(c.Context(), uint(id))
	_ = h.cacheService.InvalidateCategories(c.Context())

	return c.JSON(dto.SuccessResponseWithMessage(
		nil,
		"Categoría eliminada exitosamente",
	))
}

// ========== HELPER METHODS ==========

// parsePropertyFilter - Parsea filtros desde query params
func (h *PropertyHandler) parsePropertyFilter(c *fiber.Ctx) *dto.PropertyFilter {
	filter := &dto.PropertyFilter{}

	// Paginación
	filter.Page, _ = strconv.Atoi(c.Query("page", "1"))
	filter.Limit, _ = strconv.Atoi(c.Query("limit", "20"))

	// Filtros básicos
	tipoPropiedadStr := c.Query("tipo")
	if tipoPropiedadStr != "" {
		filter.TipoPropiedad = strings.Split(tipoPropiedadStr, ",")
	}

	operacionStr := c.Query("operacion")
	if operacionStr != "" {
		filter.Operacion = strings.Split(operacionStr, ",")
	}

	filter.Estado = c.Query("estado")
	filter.Ciudad = c.Query("ciudad")
	filter.Zona = c.Query("zona")

	// Filtros de rango
	if precioMin := c.Query("precio_min"); precioMin != "" {
		if p, err := strconv.ParseFloat(precioMin, 64); err == nil {
			filter.PrecioMin = &p
		}
	}

	if precioMax := c.Query("precio_max"); precioMax != "" {
		if p, err := strconv.ParseFloat(precioMax, 64); err == nil {
			filter.PrecioMax = &p
		}
	}

	// Amenities
	if piscina := c.Query("tiene_piscina"); piscina != "" {
		if p, err := strconv.ParseBool(piscina); err == nil {
			filter.TienePiscina = &p
		}
	}

	if jardin := c.Query("tiene_jardin"); jardin != "" {
		if j, err := strconv.ParseBool(jardin); err == nil {
			filter.TieneJardin = &j
		}
	}

	// Ordenamiento
	filter.OrderBy = c.Query("ordenar", "relevancia")

	return filter
}

// handleError - Maneja errores y retorna respuesta apropiada
func (h *PropertyHandler) handleError(c *fiber.Ctx, err error) error {
	// Verificar si es AppError
	if appErr, ok := err.(*apperrors.AppError); ok {
		// Usar el HTTP status del AppError
		status := 200
		if appErr.HTTPStatus != 0 {
			status = appErr.HTTPStatus
		} else {
			status = 500
		}

		return c.Status(status).JSON(dto.ErrorResponseWithDetails(
			string(appErr.Code),
			appErr.Message,
			appErr.Details,
		))
	}

	// Error genérico
	return c.Status(500).JSON(dto.InternalErrorResponse())
}

// AuthMiddleware - Middleware de autenticación
func (h *PropertyHandler) AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Obtener token del header
		token := c.Get("Authorization")
		if token == "" {
			return c.Status(401).JSON(response.Unauthorized())
		}

		// Extraer Bearer prefix si existe
		token = strings.TrimPrefix(token, "Bearer ")

		// Validar token y obtener usuario
		// (Esto se implementa en el auth handler)
		userID, userRole, err := h.validateTokenAndGetUser(token)
		if err != nil {
			return c.Status(401).JSON(response.Unauthorized())
		}

		// Crear usuario mínimo (en producción, obtener de BD)
		user := &models.User{
			ID:  userID,
			Rol:  models.UserRole(userRole),
		}

		// Guardar usuario en contexto
		c.Locals("user", user)

		return c.Next()
	}
}

// validateTokenAndGetUser - Valida token y retorna user info
func (h *PropertyHandler) validateTokenAndGetUser(token string) (uint, string, error) {
	// TODO: Implementar validación real de JWT
	// Por ahora retornar un usuario dummy
	return 1, "usuario", nil
}
