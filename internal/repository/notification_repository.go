package repository

import (
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"gorm.io/gorm"
)

// NotificationRepository - Repositorio para operaciones de base de datos de notificaciones
type NotificationRepository struct {
	DB *gorm.DB
}

// NewNotificationRepository - Crea un nuevo repositorio de notificaciones
func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{DB: db}
}

// Create - Crea una nueva notificación
func (r *NotificationRepository) Create(notification *models.Notification) error {
	return r.DB.Create(notification).Error
}

// GetByUserID - Obtiene notificaciones de un usuario con paginación y filtros
func (r *NotificationRepository) GetByUserID(userID uint, page, limit int, isRead *bool) ([]models.Notification, int64, int, error) {
	var notifications []models.Notification
	var total int64

	query := r.DB.Model(&models.Notification{}).Where("user_id = ?", userID)

	// Aplicar filtro de leído/no leído si se proporciona
	if isRead != nil {
		query = query.Where("is_read = ?", *isRead)
	}

	// Contar total para paginación
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, 0, err
	}

	// Calcular offset
	offset := (page - 1) * limit

	// Obtener notificaciones con paginación y relaciones
	err := query.
		Preload("Property.PropertyImages").
		Preload("SavedSearch").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error

	if err != nil {
		return nil, 0, 0, err
	}

	// Calcular total de páginas
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return notifications, total, totalPages, nil
}

// GetUnreadCount - Obtiene el conteo de notificaciones no leídas de un usuario
func (r *NotificationRepository) GetUnreadCount(userID uint) (int64, error) {
	var count int64
	err := r.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

// MarkAsRead - Marca una notificación específica como leída
func (r *NotificationRepository) MarkAsRead(notificationID, userID uint) error {
	return r.DB.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Update("is_read", true).Error
}

// MarkAllAsRead - Marca todas las notificaciones de un usuario como leídas
func (r *NotificationRepository) MarkAllAsRead(userID uint) error {
	return r.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error
}

// GetByID - Obtiene una notificación por ID
func (r *NotificationRepository) GetByID(notificationID uint) (*models.Notification, error) {
	var notification models.Notification
	err := r.DB.Where("id = ?", notificationID).First(&notification).Error
	if err != nil {
		return nil, err
	}
	return &notification, nil
}

// Delete - Elimina una notificación (soft delete)
func (r *NotificationRepository) Delete(notificationID uint) error {
	return r.DB.Delete(&models.Notification{}, notificationID).Error
}
