package repository

import (
	"context"
	"fmt"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"gorm.io/gorm"
)

// CategoryRepository - Interface para repositorio de categorías
type CategoryRepository interface {
	// CRUD básico
	Create(ctx context.Context, category *models.Category) error
	GetByID(ctx context.Context, id uint) (*models.Category, error)
	GetBySlug(ctx context.Context, slug string) (*models.Category, error)
	Update(ctx context.Context, category *models.Category) error
	Delete(ctx context.Context, id uint) error

	// Listados
	List(ctx context.Context, onlyActive bool) ([]*models.Category, error)
	ListWithPropertiesCount(ctx context.Context) ([]*CategoryWithCount, error)

	// Utilidades
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
	ExistsByName(ctx context.Context, name string) (bool, error)
}

// CategoryWithCount - Categoría con contador de propiedades
type CategoryWithCount struct {
	models.Category
	PropertiesCount int64 `json:"properties_count"`
}

// categoryRepository - Implementación del repositorio
type categoryRepository struct {
	db *gorm.DB
}

// NewCategoryRepository - Crea una nueva instancia del repositorio
func NewCategoryRepository(db *gorm.DB) CategoryRepository {
	return &categoryRepository{
		db: db,
	}
}

// ========== CRUD BÁSICO ==========

// Create - Crea una nueva categoría
func (r *categoryRepository) Create(ctx context.Context, category *models.Category) error {
	return r.db.WithContext(ctx).Create(category).Error
}

// GetByID - Obtiene una categoría por ID
func (r *categoryRepository) GetByID(ctx context.Context, id uint) (*models.Category, error) {
	var category models.Category

	err := r.db.WithContext(ctx).
		First(&category, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("error obteniendo categoría: %w", err)
	}

	return &category, nil
}

// GetBySlug - Obtiene una categoría por slug
func (r *categoryRepository) GetBySlug(ctx context.Context, slug string) (*models.Category, error) {
	var category models.Category

	err := r.db.WithContext(ctx).
		Where("slug = ?", slug).
		First(&category).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("error obteniendo categoría: %w", err)
	}

	return &category, nil
}

// Update - Actualiza una categoría
func (r *categoryRepository) Update(ctx context.Context, category *models.Category) error {
	err := r.db.WithContext(ctx).
		Model(category).
		Updates(category).Error

	if err != nil {
		return fmt.Errorf("error actualizando categoría: %w", err)
	}

	return nil
}

// Delete - Soft delete de una categoría
func (r *categoryRepository) Delete(ctx context.Context, id uint) error {
	err := r.db.WithContext(ctx).
		Delete(&models.Category{}, id).Error

	if err != nil {
		return fmt.Errorf("error eliminando categoría: %w", err)
	}

	return nil
}

// ========== LISTADOS ==========

// List - Lista todas las categorías
func (r *categoryRepository) List(ctx context.Context, onlyActive bool) ([]*models.Category, error) {
	var categories []*models.Category

	query := r.db.WithContext(ctx).Model(&models.Category{})

	if onlyActive {
		query = query.Where("activo = ?", true)
	}

	err := query.
		Order("orden ASC, nombre ASC").
		Find(&categories).Error

	if err != nil {
		return nil, fmt.Errorf("error listando categorías: %w", err)
	}

	return categories, nil
}

// ListWithPropertiesCount - Lista categorías con contador de propiedades
func (r *categoryRepository) ListWithPropertiesCount(ctx context.Context) ([]*CategoryWithCount, error) {
	var results []*CategoryWithCount

	err := r.db.WithContext(ctx).
		Model(&models.Category{}).
		Select("categorias.*, COALESCE COUNT(propiedades.id), 0) as properties_count").
		Joins("LEFT JOIN propiedades_categorias ON categorias.id = propiedades_categorias.category_id").
		Joins("LEFT JOIN propiedades ON propiedades_categorias.property_id = propiedades.id AND propiedades.deleted_at IS NULL").
		Group("categorias.id").
		Order("categorias.orden ASC, categorias.nombre ASC").
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("error listando categorías con conteo: %w", err)
	}

	return results, nil
}

// ========== UTILIDADES ==========

// ExistsBySlug - Verifica si existe una categoría por slug
func (r *categoryRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&models.Category{}).
		Where("slug = ?", slug).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// ExistsByName - Verifica si existe una categoría por nombre
func (r *categoryRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&models.Category{}).
		Where("nombre = ?", name).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// ========== ERRORES CUSTOM ==========

var (
	ErrCategoryNotFound = fmt.Errorf("categoría no encontrada")
)
