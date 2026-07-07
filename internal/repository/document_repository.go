package repository

import (
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"gorm.io/gorm"
)

// DocumentRepository - Repositorio para operaciones de base de datos de documentos
type DocumentRepository struct {
	DB *gorm.DB
}

// NewDocumentRepository - Crea un nuevo repositorio de documentos
func NewDocumentRepository(db *gorm.DB) *DocumentRepository {
	return &DocumentRepository{DB: db}
}

// Create - Crea un nuevo documento
func (r *DocumentRepository) Create(document *models.Document) error {
	return r.DB.Create(document).Error
}

// GetByID - Obtiene un documento por ID
func (r *DocumentRepository) GetByID(id uint) (*models.Document, error) {
	var document models.Document
	err := r.DB.Preload("Property").
		Preload("User").
		Where("id = ?", id).
		First(&document).Error
	if err != nil {
		return nil, err
	}
	return &document, nil
}

// GetByUserID - Obtiene documentos de un usuario con paginación
func (r *DocumentRepository) GetByUserID(userID uint, page, limit int) ([]models.Document, int64, int, error) {
	var documents []models.Document
	var total int64

	query := r.DB.Model(&models.Document{}).Where("user_id = ?", userID)

	// Contar total para paginación
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, 0, err
	}

	// Calcular offset
	offset := (page - 1) * limit

	// Obtener documentos con paginación y relaciones
	err := query.
		Preload("Property").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&documents).Error

	if err != nil {
		return nil, 0, 0, err
	}

	// Calcular total de páginas
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return documents, total, totalPages, nil
}

// GetByPropertyID - Obtiene documentos de una propiedad (solo públicos)
func (r *DocumentRepository) GetByPropertyID(propertyID uint) ([]models.Document, error) {
	var documents []models.Document
	err := r.DB.Where("property_id = ? AND is_public = ?", propertyID, true).
		Order("created_at DESC").
		Find(&documents).Error
	return documents, err
}

// GetByType - Obtiene documentos de un usuario por tipo
func (r *DocumentRepository) GetByType(userID uint, docType string) ([]models.Document, error) {
	var documents []models.Document
	err := r.DB.Where("user_id = ? AND type = ?", userID, docType).
		Preload("Property").
		Order("created_at DESC").
		Find(&documents).Error
	return documents, err
}

// Update - Actualiza un documento
func (r *DocumentRepository) Update(document *models.Document) error {
	return r.DB.Save(document).Error
}

// Delete - Elimina un documento (soft delete)
func (r *DocumentRepository) Delete(id uint) error {
	return r.DB.Delete(&models.Document{}, id).Error
}

// GetPublicDocuments - Obtiene todos los documentos públicos
func (r *DocumentRepository) GetPublicDocuments(page, limit int) ([]models.Document, int64, int, error) {
	var documents []models.Document
	var total int64

	query := r.DB.Model(&models.Document{}).Where("is_public = ?", true)

	// Contar total para paginación
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, 0, err
	}

	// Calcular offset
	offset := (page - 1) * limit

	// Obtener documentos con paginación y relaciones
	err := query.
		Preload("Property").
		Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&documents).Error

	if err != nil {
		return nil, 0, 0, err
	}

	// Calcular total de páginas
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return documents, total, totalPages, nil
}

// GetAll - Obtiene todos los documentos (admin only)
func (r *DocumentRepository) GetAll(page, limit int, docType string) ([]models.Document, int64, int, error) {
	var documents []models.Document
	var total int64

	query := r.DB.Model(&models.Document{})

	// Filtrar por tipo si se proporciona
	if docType != "" {
		query = query.Where("type = ?", docType)
	}

	// Contar total para paginación
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, 0, err
	}

	// Calcular offset
	offset := (page - 1) * limit

	// Obtener documentos con paginación y relaciones
	err := query.
		Preload("Property").
		Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&documents).Error

	if err != nil {
		return nil, 0, 0, err
	}

	// Calcular total de páginas
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return documents, total, totalPages, nil
}
