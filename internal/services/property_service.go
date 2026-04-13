package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/dto"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/repository"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/utils"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/validators"
	apperrors "github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/errors"
)

// PropertyService - Interface para servicio de propiedades
type PropertyService interface {
	// CRUD
	CreateProperty(ctx context.Context, req *dto.CreatePropertyRequest, userID uint) (*dto.PropertyResponse, error)
	GetProperty(ctx context.Context, id uint, includeContacto bool) (*dto.PropertyResponse, error)
	UpdateProperty(ctx context.Context, id uint, req *dto.UpdatePropertyRequest, userID uint) (*dto.PropertyResponse, error)
	DeleteProperty(ctx context.Context, id uint, userID uint) error

	// Listados
	ListProperties(ctx context.Context, filter *dto.PropertyFilter) (*dto.PropertyListResponse, error)
	ListUserProperties(ctx context.Context, userID uint, page, limit int) (*dto.PropertyListResponse, error)
	GetFeaturedProperties(ctx context.Context, limit int) ([]*dto.PropertyResponse, error)
	GetRelatedProperties(ctx context.Context, id uint, limit int) ([]*dto.PropertyResponse, error)

	// Fotos
	UploadPhotos(ctx context.Context, id uint, req *dto.UploadPhotosRequest, userID uint) (*dto.PropertyResponse, error)
	SetPhotoAsPortada(ctx context.Context, id, photoID uint, userID uint) error
	ReorderPhotos(ctx context.Context, id uint, orden []uint, userID uint) error
	DeletePhoto(ctx context.Context, id, photoID uint, userID uint) error

	// Búsqueda
	SearchProperties(ctx context.Context, query string, filter *dto.PropertyFilter) (*dto.PropertyListResponse, error)

	// Categorías
	AddCategory(ctx context.Context, id, categoryID uint, userID uint) error
	RemoveCategory(ctx context.Context, id, categoryID uint, userID uint) error
}

// propertyService - Implementación del servicio
type propertyService struct {
	propertyRepo repository.PropertyRepository
	categoryRepo repository.CategoryRepository
	auditRepo    repository.AuditRepository
	validator    *validators.PropertyValidator
}

// NewPropertyService - Crea una nueva instancia del servicio
func NewPropertyService(
	propertyRepo repository.PropertyRepository,
	categoryRepo repository.CategoryRepository,
	auditRepo repository.AuditRepository,
) PropertyService {
	return &propertyService{
		propertyRepo: propertyRepo,
		categoryRepo: categoryRepo,
		auditRepo:    auditRepo,
		validator:    validators.NewPropertyValidator(),
	}
}

// ========== CRUD ==========

// CreateProperty - Crea una nueva propiedad
func (s *propertyService) CreateProperty(ctx context.Context, req *dto.CreatePropertyRequest, userID uint) (*dto.PropertyResponse, error) {
	// 1. Validar request
	if errs := s.validator.ValidateCreateProperty(req); len(errs) > 0 {
		details := validators.FormatErrorsForResponse(errs)
		return nil, apperrors.ValidationErrors(details)
	}

	// 2. Crear modelo
	property := req.ToModel(userID)

	// 3. Generar slug único
	slugGenerator := utils.NewSlugGenerator()
	property.Slug = slugGenerator.GenerateUnique(property.Titulo, func(slug string) bool {
		exists, _ := s.propertyRepo.ExistsBySlug(ctx, slug)
		return exists
	})

	// 4. Generar Google Maps URL si hay coordenadas
	if property.Latitud != nil && property.Longitud != nil {
		property.GoogleMapsURL = utils.GenerateGoogleMapsURL(*property.Latitud, *property.Longitud)
	}

	// 5. Guardar en BD
	if err := s.propertyRepo.Create(ctx, property); err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	// 6. Añadir categorías si se especificaron
	if len(req.Categorias) > 0 {
		for _, catSlugOrID := range req.Categorias {
			// Intentar obtener por slug primero
			category, err := s.categoryRepo.GetBySlug(ctx, catSlugOrID)
			if err != nil {
				// Si falla, intentar por ID
				// (implementar si necesario)
				continue
			}

			// Añadir categoría
			_ = s.propertyRepo.AddCategory(ctx, property.ID, category.ID)
		}
	}

	// 7. Log de auditoría
	_ = s.auditRepo.LogCreate(ctx, userID, "propiedad", property.ID, property)

	// 8. Retornar response (recargar para obtener relaciones)
	savedProperty, _ := s.propertyRepo.GetByID(ctx, property.ID)
	return dto.MapPropertyToResponse(savedProperty, false), nil
}

// GetProperty - Obtiene una propiedad por ID
func (s *propertyService) GetProperty(ctx context.Context, id uint, includeContacto bool) (*dto.PropertyResponse, error) {
	// 1. Obtener propiedad
	property, err := s.propertyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.PropertyNotFound(id)
	}

	// 2. Incrementar contador de visitas
	_ = s.propertyRepo.IncrementVisits(ctx, id)

	// 3. Mapear a response
	return dto.MapPropertyToResponse(property, includeContacto), nil
}

// UpdateProperty - Actualiza una propiedad
func (s *propertyService) UpdateProperty(ctx context.Context, id uint, req *dto.UpdatePropertyRequest, userID uint) (*dto.PropertyResponse, error) {
	// 1. Verificar que la propiedad existe
	property, err := s.propertyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.PropertyNotFound(id)
	}

	// 2. Verificar que el usuario es dueño
	if !property.IsOwnedBy(userID) {
		return nil, apperrors.NotOwner("propiedad")
	}

	// 3. Validar request
	if errs := s.validator.ValidateUpdateProperty(req); len(errs) > 0 {
		details := validators.FormatErrorsForResponse(errs)
		return nil, apperrors.ValidationErrors(details)
	}

	// 4. Guardar estado anterior para auditoría
	beforeCopy := *property

	// 5. Aplicar actualizaciones
	s.applyUpdates(property, req)

	// 6. Actualizar slug si cambió el título
	if req.Titulo != nil && *req.Titulo != property.Titulo {
		slugGenerator := utils.NewSlugGenerator()
		property.Slug = slugGenerator.GenerateUnique(*req.Titulo, func(slug string) bool {
			exists, _ := s.propertyRepo.ExistsBySlug(ctx, slug)
			return exists
		})
	}

	// 7. Actualizar Google Maps URL si cambiaron coordenadas
	if req.Latitud != nil || req.Longitud != nil {
		if property.Latitud != nil && property.Longitud != nil {
			property.GoogleMapsURL = utils.GenerateGoogleMapsURL(*property.Latitud, *property.Longitud)
		}
	}

	// 8. Guardar cambios
	if err := s.propertyRepo.Update(ctx, property); err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	// 9. Log de auditoría
	_ = s.auditRepo.LogUpdate(ctx, userID, "propiedad", id, beforeCopy, property)

	// 10. Recargar y retornar
	updatedProperty, _ := s.propertyRepo.GetByID(ctx, id)
	return dto.MapPropertyToResponse(updatedProperty, false), nil
}

// DeleteProperty - Elimina una propiedad
func (s *propertyService) DeleteProperty(ctx context.Context, id uint, userID uint) error {
	// 1. Verificar que la propiedad existe
	property, err := s.propertyRepo.GetByID(ctx, id)
	if err != nil {
		return apperrors.PropertyNotFound(id)
	}

	// 2. Verificar que el usuario es dueño
	if !property.IsOwnedBy(userID) {
		return apperrors.NotOwner("propiedad")
	}

	// 3. Log de auditoría antes de eliminar
	_ = s.auditRepo.LogDelete(ctx, userID, "propiedad", id, property)

	// 4. Eliminar (soft delete)
	if err := s.propertyRepo.Delete(ctx, id); err != nil {
		return apperrors.DatabaseError(err)
	}

	return nil
}

// ========== LISTADOS ==========

// ListProperties - Lista propiedades con filtros
func (s *propertyService) ListProperties(ctx context.Context, filter *dto.PropertyFilter) (*dto.PropertyListResponse, error) {
	// 1. Validar filtros
	if errs := s.validator.ValidatePropertyFilter(filter); len(errs) > 0 {
		details := validators.FormatErrorsForResponse(errs)
		return nil, apperrors.ValidationErrors(details)
	}

	// 2. Establecer defaults
	filter.SetDefaults()

	// 3. Listar propiedades
	properties, total, err := s.propertyRepo.List(ctx, filter)
	if err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	// 4. Mapear a responses
	items := make([]dto.PropertyListItemResponse, len(properties))
	for i, property := range properties {
		items[i] = dto.MapPropertyToListItem(property)
	}

	// 5. Crear response
	pagination := dto.NewPaginationResponse(filter.Page, filter.Limit, total)
	return &dto.PropertyListResponse{
		Data:       items,
		Pagination: pagination,
	}, nil
}

// ListUserProperties - Lista propiedades de un usuario
func (s *propertyService) ListUserProperties(ctx context.Context, userID uint, page, limit int) (*dto.PropertyListResponse, error) {
	properties, total, err := s.propertyRepo.ListByUser(ctx, userID, page, limit)
	if err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	items := make([]dto.PropertyListItemResponse, len(properties))
	for i, property := range properties {
		items[i] = dto.MapPropertyToListItem(property)
	}

	pagination := dto.NewPaginationResponse(page, limit, total)
	return &dto.PropertyListResponse{
		Data:       items,
		Pagination: pagination,
	}, nil
}

// GetFeaturedProperties - Obtiene propiedades destacadas
func (s *propertyService) GetFeaturedProperties(ctx context.Context, limit int) ([]*dto.PropertyResponse, error) {
	properties, err := s.propertyRepo.GetFeatured(ctx, limit)
	if err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	responses := make([]*dto.PropertyResponse, len(properties))
	for i, property := range properties {
		// Incluir contacto para destacadas
		prop := dto.MapPropertyToResponse(property, true)
		responses[i] = prop
	}

	return responses, nil
}

// GetRelatedProperties - Obtiene propiedades relacionadas
func (s *propertyService) GetRelatedProperties(ctx context.Context, id uint, limit int) ([]*dto.PropertyResponse, error) {
	properties, err := s.propertyRepo.GetRelated(ctx, id, limit)
	if err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	responses := make([]*dto.PropertyResponse, len(properties))
	for i, property := range properties {
		prop := dto.MapPropertyToResponse(property, false)
		responses[i] = prop
	}

	return responses, nil
}

// ========== FOTOS ==========

// UploadPhotos - Sube fotos a una propiedad
func (s *propertyService) UploadPhotos(ctx context.Context, id uint, req *dto.UploadPhotosRequest, userID uint) (*dto.PropertyResponse, error) {
	// 1. Verificar que la propiedad existe
	property, err := s.propertyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.PropertyNotFound(id)
	}

	// 2. Verificar que el usuario es dueño
	if !property.IsOwnedBy(userID) {
		return nil, apperrors.NotOwner("propiedad")
	}

	// 3. Validar request
	if errs := s.validator.ValidateUploadPhotos(req); len(errs) > 0 {
		// Filtrar warning
		details := make([]interface{}, 0)
		for k, v := range errs {
			if k != "warning" {
				details = append(details, map[string]string{
					"field":   k,
					"message": v,
				})
			}
		}

		if len(details) > 0 {
			return nil, apperrors.ValidationErrors(details)
		}

		// Si solo hay warning, continuar
		if _, hasWarning := errs["warning"]; hasWarning {
			// Log warning pero continuar
		}
	}

	// 4. Procesar cada foto
	for i, file := range req.Files {
		// Generar nombre único
		ext := filepath.Ext(file.Filename)
		filename := fmt.Sprintf("%d_%d_%s%s", id, time.Now().Unix(), i, ext)

		// TODO: Procesar imagen (resize, thumbnail)
		// TODO: Subir a storage (R2 o local)

		// Por ahora, crear registro con URL placeholder
		photo := &models.PropertyPhoto{
			PropertyID: id,
			URL:        fmt.Sprintf("/uploads/propiedades/%d/%s", id, filename),
			Orden:      i,
			FileName:   filename,
			FileSize:   file.Size,
			MimeType:   file.MimeType,
		}

		// Es portada si es la primera foto y no hay portada
		if i == 0 && len(property.Fotos) == 0 {
			photo.EsPortada = true
		}

		if err := s.propertyRepo.AddPhoto(ctx, photo); err != nil {
			return nil, apperrors.DatabaseError(err)
		}
	}

	// 5. Recargar propiedad y retornar
	updatedProperty, _ := s.propertyRepo.GetByID(ctx, id)
	return dto.MapPropertyToResponse(updatedProperty, false), nil
}

// SetPhotoAsPortada - Marca una foto como portada
func (s *propertyService) SetPhotoAsPortada(ctx context.Context, id, photoID uint, userID uint) error {
	// 1. Verificar propiedad
	property, err := s.propertyRepo.GetByID(ctx, id)
	if err != nil {
		return apperrors.PropertyNotFound(id)
	}

	// 2. Verificar dueño
	if !property.IsOwnedBy(userID) {
		return apperrors.NotOwner("propiedad")
	}

	// 3. Marcar portada (usando transacción)
	if err := s.propertyRepo.SetPortada(ctx, id, photoID); err != nil {
		return apperrors.DatabaseError(err)
	}

	return nil
}

// ReorderPhotos - Reordena las fotos
func (s *propertyService) ReorderPhotos(ctx context.Context, id uint, orden []uint, userID uint) error {
	// 1. Verificar propiedad
	property, err := s.propertyRepo.GetByID(ctx, id)
	if err != nil {
		return apperrors.PropertyNotFound(id)
	}

	// 2. Verificar dueño
	if !property.IsOwnedBy(userID) {
		return apperrors.NotOwner("propiedad")
	}

	// 3. Reordenar (usando transacción)
	if err := s.propertyRepo.ReorderPhotos(ctx, id, orden); err != nil {
		return apperrors.DatabaseError(err)
	}

	return nil
}

// DeletePhoto - Elimina una foto
func (s *propertyService) DeletePhoto(ctx context.Context, id, photoID uint, userID uint) error {
	// 1. Verificar propiedad
	property, err := s.propertyRepo.GetByID(ctx, id)
	if err != nil {
		return apperrors.PropertyNotFound(id)
	}

	// 2. Verificar dueño
	if !property.IsOwnedBy(userID) {
		return apperrors.NotOwner("propiedad")
	}

	// 3. Verificar que la foto existe y pertenece a la propiedad
	var photo *models.PropertyPhoto
	for _, p := range property.Fotos {
		if p.ID == photoID {
			photo = &p
			break
		}
	}

	if photo == nil {
		return apperrors.NotFound("foto")
	}

	// 4. Eliminar foto
	if err := s.propertyRepo.DeletePhoto(ctx, photoID); err != nil {
		return apperrors.DatabaseError(err)
	}

	// 5. Si era portada, marcar la siguiente como portada
	if photo.EsPortada && len(property.Fotos) > 1 {
		// Buscar la siguiente foto
		for _, p := range property.Fotos {
			if p.ID != photoID {
				_ = s.propertyRepo.SetPortada(ctx, id, p.ID)
				break
			}
		}
	}

	return nil
}

// ========== BÚSQUEDA ==========

// SearchProperties - Búsqueda full-text de propiedades
func (s *propertyService) SearchProperties(ctx context.Context, query string, filter *dto.PropertyFilter) (*dto.PropertyListResponse, error) {
	// 1. Validar query
	query = strings.TrimSpace(query)
	if query == "" {
		return s.ListProperties(ctx, filter)
	}

	// 2. Buscar
	properties, total, err := s.propertyRepo.Search(ctx, query, filter)
	if err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	// 3. Mapear
	items := make([]dto.PropertyListItemResponse, len(properties))
	for i, property := range properties {
		items[i] = dto.MapPropertyToListItem(property)
	}

	// 4. Response
	filter.SetDefaults()
	pagination := dto.NewPaginationResponse(filter.Page, filter.Limit, total)
	return &dto.PropertyListResponse{
		Data:       items,
		Pagination: pagination,
	}, nil
}

// ========== CATEGORÍAS ==========

// AddCategory - Añade una categoría a una propiedad
func (s *propertyService) AddCategory(ctx context.Context, id, categoryID uint, userID uint) error {
	// 1. Verificar propiedad
	property, err := s.propertyRepo.GetByID(ctx, id)
	if err != nil {
		return apperrors.PropertyNotFound(id)
	}

	// 2. Verificar dueño
	if !property.IsOwnedBy(userID) {
		return apperrors.NotOwner("propiedad")
	}

	// 3. Verificar que la categoría existe
	_, err = s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return apperrors.CategoryNotFound(fmt.Sprintf("%d", categoryID))
	}

	// 4. Añadir categoría
	if err := s.propertyRepo.AddCategory(ctx, id, categoryID); err != nil {
		return apperrors.DatabaseError(err)
	}

	return nil
}

// RemoveCategory - Elimina una categoría de una propiedad
func (s *propertyService) RemoveCategory(ctx context.Context, id, categoryID uint, userID uint) error {
	// 1. Verificar propiedad
	property, err := s.propertyRepo.GetByID(ctx, id)
	if err != nil {
		return apperrors.PropertyNotFound(id)
	}

	// 2. Verificar dueño
	if !property.IsOwnedBy(userID) {
		return apperrors.NotOwner("propiedad")
	}

	// 3. Eliminar categoría
	if err := s.propertyRepo.RemoveCategory(ctx, id, categoryID); err != nil {
		return apperrors.DatabaseError(err)
	}

	return nil
}

// ========== HELPER METHODS ==========

// applyUpdates - Aplica actualizaciones del request al modelo
func (s *propertyService) applyUpdates(property *models.Property, req *dto.UpdatePropertyRequest) {
	if req.Titulo != nil {
		property.Titulo = *req.Titulo
	}

	if req.Descripcion != nil {
		property.Descripcion = *req.Descripcion
	}

	if req.TipoPropiedad != nil {
		property.TipoPropiedad = *req.TipoPropiedad
	}

	if req.Operacion != nil {
		property.Operacion = *req.Operacion
	}

	if req.Estado != nil {
		property.Estado = *req.Estado
	}

	if req.Direccion != nil {
		property.Direccion = *req.Direccion
	}

	if req.Ciudad != nil {
		property.Ciudad = *req.Ciudad
	}

	if req.Zona != nil {
		property.Zona = *req.Zona
	}

	if req.Latitud != nil {
		property.Latitud = req.Latitud
	}

	if req.Longitud != nil {
		property.Longitud = req.Longitud
	}

	if req.Precio != nil {
		property.Precio = *req.Precio
	}

	if req.Moneda != nil {
		property.Moneda = *req.Moneda
	}

	if req.MetrajeTotal != nil {
		property.MetrajeTotal = req.MetrajeTotal
	}

	if req.MetrajeConstruido != nil {
		property.MetrajeConstruido = req.MetrajeConstruido
	}

	if req.NumeroHabitaciones != nil {
		property.NumeroHabitaciones = req.NumeroHabitaciones
	}

	if req.NumeroBanos != nil {
		property.NumeroBanos = req.NumeroBanos
	}

	if req.NumeroGarages != nil {
		property.NumeroGarages = req.NumeroGarages
	}

	if req.AntiguedadAnos != nil {
		property.AntiguedadAnos = req.AntiguedadAnos
	}

	if req.TienePiscina != nil {
		property.TienePiscina = *req.TienePiscina
	}

	if req.TieneJardin != nil {
		property.TieneJardin = *req.TieneJardin
	}

	if req.TieneTerraza != nil {
		property.TieneTerraza = *req.TieneTerraza
	}

	if req.TieneServicio != nil {
		property.TieneServicio = *req.TieneServicio
	}

	if req.TieneSeguridad != nil {
		property.TieneSeguridad = *req.TieneSeguridad
	}

	if req.EsAmueblado != nil {
		property.EsAmueblado = *req.EsAmueblado
	}

	if req.AceptaMascotas != nil {
		property.AceptaMascotas = *req.AceptaMascotas
	}

	if req.ContactoNombre != nil {
		property.ContactoNombre = *req.ContactoNombre
	}

	if req.ContactoTelefono != nil {
		property.ContactoTelefono = *req.ContactoTelefono
	}

	if req.ContactoEmail != nil {
		property.ContactoEmail = *req.ContactoEmail
	}

	if req.Destacado != nil {
		property.Destacado = *req.Destacado
	}

	if req.MetaTitulo != nil {
		property.MetaTitulo = *req.MetaTitulo
	}

	if req.MetaDescripcion != nil {
		property.MetaDescripcion = *req.MetaDescripcion
	}
}
