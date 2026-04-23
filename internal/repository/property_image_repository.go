package repository

import (
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"gorm.io/gorm"
)

// PropertyImageRepository - Repositorio para operaciones de base de datos de imágenes
type PropertyImageRepository struct {
	DB *config.Database
}

// NewPropertyImageRepository - Crea un nuevo repositorio de imágenes
func NewPropertyImageRepository(db *config.Database) *PropertyImageRepository {
	return &PropertyImageRepository{DB: db}
}

// Create - Crea una nueva imagen
func (r *PropertyImageRepository) Create(image *models.PropertyImage) error {
	return r.DB.DB.Create(image).Error
}

// CreateBatch - Crea múltiples imágenes en batch
func (r *PropertyImageRepository) CreateBatch(images []models.PropertyImage) error {
	if len(images) == 0 {
		return nil
	}
	return r.DB.DB.Create(&images).Error
}

// GetByID - Obtiene una imagen por ID
func (r *PropertyImageRepository) GetByID(id uint) (*models.PropertyImage, error) {
	var image models.PropertyImage
	err := r.DB.DB.First(&image, id).Error
	if err != nil {
		return nil, err
	}
	return &image, nil
}

// GetByPropertyID - Obtiene todas las imágenes de una propiedad
func (r *PropertyImageRepository) GetByPropertyID(propertyID uint) ([]models.PropertyImage, error) {
	var images []models.PropertyImage
	err := r.DB.DB.
		Where("property_id = ?", propertyID).
		Order("orden ASC").
		Find(&images).Error
	if err != nil {
		return nil, err
	}
	return images, nil
}

// Update - Actualiza una imagen
func (r *PropertyImageRepository) Update(image *models.PropertyImage) error {
	return r.DB.DB.Save(image).Error
}

// Delete - Elimina una imagen
func (r *PropertyImageRepository) Delete(image *models.PropertyImage) error {
	return r.DB.DB.Delete(image).Error
}

// DeleteByPropertyID - Elimina todas las imágenes de una propiedad
func (r *PropertyImageRepository) DeleteByPropertyID(propertyID uint) error {
	return r.DB.DB.
		Where("property_id = ?", propertyID).
		Delete(&models.PropertyImage{}).Error
}

// GetPrincipalImage - Obtiene la imagen principal de una propiedad
func (r *PropertyImageRepository) GetPrincipalImage(propertyID uint) (*models.PropertyImage, error) {
	var image models.PropertyImage
	err := r.DB.DB.
		Where("property_id = ? AND principal = ?", propertyID, true).
		First(&image).Error
	if err != nil {
		return nil, err
	}
	return &image, nil
}

// SetPrincipalImage - Establece una imagen como principal (y desmarca las demás)
func (r *PropertyImageRepository) SetPrincipalImage(propertyID uint, imageID uint) error {
	// Transacción para desmarcar todas y marcar la nueva
	return r.DB.DB.Transaction(func(tx *gorm.DB) error {
		// Desmarcar todas como principales
		if err := tx.Model(&models.PropertyImage{}).
			Where("property_id = ?", propertyID).
			Update("principal", false).Error; err != nil {
			return err
		}

		// Marcar la nueva como principal
		if err := tx.Model(&models.PropertyImage{}).
			Where("id = ? AND property_id = ?", imageID, propertyID).
			Update("principal", true).Error; err != nil {
			return err
		}

		return nil
	})
}

// CountByPropertyID - Cuenta las imágenes de una propiedad
func (r *PropertyImageRepository) CountByPropertyID(propertyID uint) (int64, error) {
	var count int64
	err := r.DB.DB.Model(&models.PropertyImage{}).
		Where("property_id = ?", propertyID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
