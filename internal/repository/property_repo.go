package repository

import (
	"context"
	"fmt"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/dto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PropertyRepository - Interface para repositorio de propiedades
type PropertyRepository interface {
	// CRUD básico
	Create(ctx context.Context, property *models.Property) error
	GetByID(ctx context.Context, id uint) (*models.Property, error)
	Update(ctx context.Context, property *models.Property) error
	Delete(ctx context.Context, id uint) error

	// Listados y búsqueda
	List(ctx context.Context, filter *dto.PropertyFilter) ([]*models.Property, int64, error)
	ListByUser(ctx context.Context, userID uint, page, limit int) ([]*models.Property, int64, error)
	GetFeatured(ctx context.Context, limit int) ([]*models.Property, error)
	GetRelated(ctx context.Context, propertyID uint, limit int) ([]*models.Property, error)

	// Búsqueda y filtros
	Search(ctx context.Context, query string, filter *dto.PropertyFilter) ([]*models.Property, int64, error)

	// Fotos
	AddPhoto(ctx context.Context, photo *models.PropertyPhoto) error
	UpdatePhoto(ctx context.Context, photo *models.PropertyPhoto) error
	DeletePhoto(ctx context.Context, photoID uint) error
	GetPhotos(ctx context.Context, propertyID uint) ([]models.PropertyPhoto, error)
	SetPortada(ctx context.Context, propertyID, photoID uint) error
	ReorderPhotos(ctx context.Context, propertyID uint, orden []uint) error

	// Categorías
	AddCategory(ctx context.Context, propertyID, categoryID uint) error
	RemoveCategory(ctx context.Context, propertyID, categoryID uint) error
	GetCategories(ctx context.Context, propertyID uint) ([]models.Category, error)

	// Métricas
	IncrementVisits(ctx context.Context, propertyID uint) error
	IncrementContacts(ctx context.Context, propertyID uint) error

	// Utilidades
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
	ExistsByID(ctx context.Context, id uint) (bool, error)
	IsOwner(ctx context.Context, propertyID, userID uint) (bool, error)
}

// propertyRepository - Implementación del repositorio
type propertyRepository struct {
	db *gorm.DB
}

// NewPropertyRepository - Crea una nueva instancia del repositorio
func NewPropertyRepository(db *gorm.DB) PropertyRepository {
	return &propertyRepository{
		db: db,
	}
}

// ========== CRUD BÁSICO ==========

// Create - Crea una nueva propiedad
func (r *propertyRepository) Create(ctx context.Context, property *models.Property) error {
	return r.db.WithContext(ctx).Create(property).Error
}

// GetByID - Obtiene una propiedad por ID con eager loading
func (r *propertyRepository) GetByID(ctx context.Context, id uint) (*models.Property, error) {
	var property models.Property

	err := r.db.WithContext(ctx).
		Preload("Fotos", func(db *gorm.DB) *gorm.DB {
			return db.Order("orden ASC")
		}).
		Preload("Categorias").
		Preload("User").
		First(&property, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrPropertyNotFound
		}
		return nil, fmt.Errorf("error obteniendo propiedad: %w", err)
	}

	return &property, nil
}

// Update - Actualiza una propiedad
func (r *propertyRepository) Update(ctx context.Context, property *models.Property) error {
	err := r.db.WithContext(ctx).
		Model(property).
		Updates(property).Error

	if err != nil {
		return fmt.Errorf("error actualizando propiedad: %w", err)
	}

	// Recargar para actualizar relaciones
	return r.db.WithContext(ctx).
		Preload("Fotos").
		Preload("Categorias").
		First(property, property.ID).Error
}

// Delete - Soft delete de una propiedad
func (r *propertyRepository) Delete(ctx context.Context, id uint) error {
	err := r.db.WithContext(ctx).
		Delete(&models.Property{}, id).Error

	if err != nil {
		return fmt.Errorf("error eliminando propiedad: %w", err)
	}

	return nil
}

// ========== LISTADOS Y BÚSQUEDA ==========

// List - Lista propiedades con filtros y paginación
func (r *propertyRepository) List(ctx context.Context, filter *dto.PropertyFilter) ([]*models.Property, int64, error) {
	var properties []*models.Property
	var total int64

	// Construir query base
	query := r.db.WithContext(ctx).Model(&models.Property{})

	// Aplicar filtros
	query = r.applyFilters(query, filter)

	// Contar total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("error contando propiedades: %w", err)
	}

	// Aplicar paginación
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 20
	}

	offset := (filter.Page - 1) * filter.Limit

	// Aplicar ordenamiento
	query = r.applyOrderBy(query, filter.OrderBy)

	// Eager loading
	query = query.Preload("Fotos", func(db *gorm.DB) *gorm.DB {
		return db.Where("es_portada = ? OR orden = 0", true).
			Order("es_portada DESC, orden ASC")
	})

	// Ejecutar query
	if err := query.
		Offset(offset).
		Limit(filter.Limit).
		Find(&properties).Error; err != nil {
		return nil, 0, fmt.Errorf("error listando propiedades: %w", err)
	}

	return properties, total, nil
}

// ListByUser - Lista propiedades de un usuario
func (r *propertyRepository) ListByUser(ctx context.Context, userID uint, page, limit int) ([]*models.Property, int64, error) {
	var properties []*models.Property
	var total int64

	offset := (page - 1) * limit

	// Contar
	if err := r.db.WithContext(ctx).
		Model(&models.Property{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Listar
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&properties).Error; err != nil {
		return nil, 0, err
	}

	return properties, total, nil
}

// GetFeatured - Obtiene propiedades destacadas
func (r *propertyRepository) GetFeatured(ctx context.Context, limit int) ([]*models.Property, error) {
	var properties []*models.Property

	err := r.db.WithContext(ctx).
		Where("destacado = ? AND estado = ?", true, models.StatusActiva).
		Order("numero_visitas DESC, created_at DESC").
		Limit(limit).
		Preload("Fotos", func(db *gorm.DB) *gorm.DB {
			return db.Where("es_portada = ?", true)
		}).
		Find(&properties).Error

	if err != nil {
		return nil, fmt.Errorf("error obteniendo propiedades destacadas: %w", err)
	}

	return properties, nil
}

// GetRelated - Obtiene propiedades relacionadas
func (r *propertyRepository) GetRelated(ctx context.Context, propertyID uint, limit int) ([]*models.Property, error) {
	// Primero obtener la propiedad actual
	var currentProperty models.Property
	if err := r.db.WithContext(ctx).First(&currentProperty, propertyID).Error; err != nil {
		return nil, err
	}

	var properties []*models.Property

	// Buscar propiedades con:
	// - Mismo tipo_propiedad
	// - Misma operación
	// - Misma ciudad
	// - Precio ±20%
	// - ID diferente

	minPrecio := currentProperty.Precio * 0.8
	maxPrecio := currentProperty.Precio * 1.2

	err := r.db.WithContext(ctx).
		Where("tipo_propiedad = ?", currentProperty.TipoPropiedad).
		Where("operacion = ?", currentProperty.Operacion).
		Where("ciudad = ?", currentProperty.Ciudad).
		Where("precio BETWEEN ? AND ?", minPrecio, maxPrecio).
		Where("id != ?", propertyID).
		Where("estado = ?", models.StatusActiva).
		Order("RANDOM()").
		Limit(limit).
		Preload("Fotos", func(db *gorm.DB) *gorm.DB {
			return db.Where("es_portada = ?", true)
		}).
		Find(&properties).Error

	if err != nil {
		return nil, fmt.Errorf("error obteniendo propiedades relacionadas: %w", err)
	}

	return properties, nil
}

// ========== BÚSQUEDA Y FILTROS ==========

// Search - Búsqueda full-text de propiedades
func (r *propertyRepository) Search(ctx context.Context, query string, filter *dto.PropertyFilter) ([]*models.Property, int64, error) {
	var properties []*models.Property
	var total int64

	// Construir query base con búsqueda full-text
	dbQuery := r.db.WithContext(ctx).
		Where("estado = ?", models.StatusActiva).
		Where("to_tsvector('spanish', titulo || ' ' || descripcion) @@ to_tsquery('spanish', ?)", query)

	// Aplicar filtros adicionales
	dbQuery = r.applyFilters(dbQuery, filter)

	// Contar
	if err := dbQuery.Model(&models.Property{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Listar
	if err := dbQuery.
		Preload("Fotos").
		Find(&properties).Error; err != nil {
		return nil, 0, err
	}

	return properties, total, nil
}

// ========== FOTOS ==========

// AddPhoto - Añade una foto a una propiedad
func (r *propertyRepository) AddPhoto(ctx context.Context, photo *models.PropertyPhoto) error {
	return r.db.WithContext(ctx).Create(photo).Error
}

// UpdatePhoto - Actualiza una foto
func (r *propertyRepository) UpdatePhoto(ctx context.Context, photo *models.PropertyPhoto) error {
	return r.db.WithContext(ctx).
		Model(photo).
		Updates(photo).Error
}

// DeletePhoto - Elimina una foto
func (r *propertyRepository) DeletePhoto(ctx context.Context, photoID uint) error {
	return r.db.WithContext(ctx).
		Delete(&models.PropertyPhoto{}, photoID).Error
}

// GetPhotos - Obtiene todas las fotos de una propiedad
func (r *propertyRepository) GetPhotos(ctx context.Context, propertyID uint) ([]models.PropertyPhoto, error) {
	var photos []models.PropertyPhoto

	err := r.db.WithContext(ctx).
		Where("property_id = ?", propertyID).
		Order("orden ASC").
		Find(&photos).Error

	if err != nil {
		return nil, fmt.Errorf("error obteniendo fotos: %w", err)
	}

	return photos, nil
}

// SetPortada - Marca una foto como portada
func (r *propertyRepository) SetPortada(ctx context.Context, propertyID, photoID uint) error {
	// Transaction
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Desmarcar todas las fotos de la propiedad
		if err := tx.Model(&models.PropertyPhoto{}).
			Where("property_id = ?", propertyID).
			Update("es_portada", false).Error; err != nil {
			return err
		}

		// Marcar la foto seleccionada
		if err := tx.Model(&models.PropertyPhoto{}).
			Where("id = ? AND property_id = ?", photoID, propertyID).
			Update("es_portada", true).Error; err != nil {
			return err
		}

		return nil
	})
}

// ReorderPhotos - Reordena las fotos de una propiedad
func (r *propertyRepository) ReorderPhotos(ctx context.Context, propertyID uint, orden []uint) error {
	// Transaction
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, photoID := range orden {
			if err := tx.Model(&models.PropertyPhoto{}).
				Where("id = ? AND property_id = ?", photoID, propertyID).
				Update("orden", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ========== CATEGORÍAS ==========

// AddCategory - Añade una categoría a una propiedad
func (r *propertyRepository) AddCategory(ctx context.Context, propertyID, categoryID uint) error {
	propertyCategory := models.PropertyCategory{
		PropertyID: propertyID,
		CategoryID: categoryID,
	}

	// Usar clause.OnConflict para evitar duplicados
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		Create(&propertyCategory).Error
}

// RemoveCategory - Elimina una categoría de una propiedad
func (r *propertyRepository) RemoveCategory(ctx context.Context, propertyID, categoryID uint) error {
	return r.db.WithContext(ctx).
		Where("property_id = ? AND category_id = ?", propertyID, categoryID).
		Delete(&models.PropertyCategory{}).Error
}

// GetCategories - Obtiene todas las categorías de una propiedad
func (r *propertyRepository) GetCategories(ctx context.Context, propertyID uint) ([]models.Category, error) {
	var property models.Property

	err := r.db.WithContext(ctx).
		Preload("Categorias").
		First(&property, propertyID).Error

	if err != nil {
		return nil, fmt.Errorf("error obteniendo categorías: %w", err)
	}

	return property.Categorias, nil
}

// ========== MÉTRICAS ==========

// IncrementVisits - Incrementa el contador de visitas
func (r *propertyRepository) IncrementVisits(ctx context.Context, propertyID uint) error {
	return r.db.WithContext(ctx).
		Model(&models.Property{}).
		Where("id = ?", propertyID).
		UpdateColumn("numero_visitas", gorm.Expr("numero_visitas + 1")).Error
}

// IncrementContacts - Incrementa el contador de contactos
func (r *propertyRepository) IncrementContacts(ctx context.Context, propertyID uint) error {
	return r.db.WithContext(ctx).
		Model(&models.Property{}).
		Where("id = ?", propertyID).
		UpdateColumn("numero_contactos", gorm.Expr("numero_contactos + 1")).Error
}

// ========== UTILIDADES ==========

// ExistsBySlug - Verifica si existe una propiedad por slug
func (r *propertyRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&models.Property{}).
		Where("slug = ?", slug).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// ExistsByID - Verifica si existe una propiedad por ID
func (r *propertyRepository) ExistsByID(ctx context.Context, id uint) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&models.Property{}).
		Where("id = ?", id).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// IsOwner - Verifica si un usuario es dueño de una propiedad
func (r *propertyRepository) IsOwner(ctx context.Context, propertyID, userID uint) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&models.Property{}).
		Where("id = ? AND user_id = ?", propertyID, userID).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// ========== MÉTODOS PRIVADOS ==========

// applyFilters - Aplica filtros a la query
func (r *propertyRepository) applyFilters(query *gorm.DB, filter *dto.PropertyFilter) *gorm.DB {
	if filter == nil {
		return query
	}

	// Estado (default: activa si no se especifica)
	if filter.Estado == "" {
		query = query.Where("estado = ?", models.StatusActiva)
	} else {
		query = query.Where("estado = ?", filter.Estado)
	}

	// Tipo de propiedad
	if len(filter.TipoPropiedad) > 0 {
		query = query.Where("tipo_propiedad IN ?", filter.TipoPropiedad)
	}

	// Operación
	if len(filter.Operacion) > 0 {
		query = query.Where("operacion IN ?", filter.Operacion)
	}

	// Ubicación
	if filter.Ciudad != "" {
		query = query.Where("ciudad ILIKE ?", "%"+filter.Ciudad+"%")
	}

	if filter.Zona != "" {
		query = query.Where("zona ILIKE ?", "%"+filter.Zona+"%")
	}

	// Usuario
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}

	// Rango de precios
	if filter.PrecioMin != nil {
		query = query.Where("precio >= ?", *filter.PrecioMin)
	}

	if filter.PrecioMax != nil {
		query = query.Where("precio <= ?", *filter.PrecioMax)
	}

	// Rango de metraje
	if filter.MetrajeMin != nil {
		query = query.Where("metraje_construido >= ?", *filter.MetrajeMin)
	}

	if filter.MetrajeMax != nil {
		query = query.Where("metraje_construido <= ?", *filter.MetrajeMax)
	}

	// Rango de habitaciones
	if filter.HabitacionesMin != nil {
		query = query.Where("numero_habitaciones >= ?", *filter.HabitacionesMin)
	}

	if filter.HabitacionesMax != nil {
		query = query.Where("numero_habitaciones <= ?", *filter.HabitacionesMax)
	}

	// Rango de baños
	if filter.BanosMin != nil {
		query = query.Where("numero_banos >= ?", *filter.BanosMin)
	}

	if filter.BanosMax != nil {
		query = query.Where("numero_banos <= ?", *filter.BanosMax)
	}

	// Amenities (boolean)
	if filter.TienePiscina != nil {
		query = query.Where("tiene_piscina = ?", *filter.TienePiscina)
	}

	if filter.TieneJardin != nil {
		query = query.Where("tiene_jardin = ?", *filter.TieneJardin)
	}

	if filter.TieneTerraza != nil {
		query = query.Where("tiene_terraza = ?", *filter.TieneTerraza)
	}

	if filter.TieneServicio != nil {
		query = query.Where("tiene_servicio = ?", *filter.TieneServicio)
	}

	if filter.TieneSeguridad != nil {
		query = query.Where("tiene_seguridad = ?", *filter.TieneSeguridad)
	}

	if filter.EsAmueblado != nil {
		query = query.Where("es_amueblado = ?", *filter.EsAmueblado)
	}

	if filter.AceptaMascotas != nil {
		query = query.Where("acepta_mascotas = ?", *filter.AceptaMascotas)
	}


	return query
}

// applyOrderBy - Aplica ordenamiento
func (r *propertyRepository) applyOrderBy(query *gorm.DB, orderBy string) *gorm.DB {
	switch orderBy {
	case "precio_asc":
		return query.Order("precio ASC")
	case "precio_desc":
		return query.Order("precio DESC")
	case "mas_recientes":
		return query.Order("created_at DESC")
	case "mas_visitados":
		return query.Order("numero_visitas DESC")
	case "relevancia", "":
		return query.Order("destacado DESC, created_at DESC")
	default:
		return query.Order("created_at DESC")
	}
}

// ========== ERRORES CUSTOM ==========

var (
	ErrPropertyNotFound = fmt.Errorf("propiedad no encontrada")
	ErrPhotoNotFound    = fmt.Errorf("foto no encontrada")
	ErrNotOwner         = fmt.Errorf("usuario no es dueño de la propiedad")
)
