package repository

import (
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
)

// PropertyRepositoryInterface - Interfaz para repositorio de propiedades
type PropertyRepositoryInterface interface {
	Create(property *models.Property) error
	GetByID(id uint) (*models.Property, error)
	GetByIDWithUser(id uint) (*models.Property, error)
	GetByIDWithRelations(id uint) (*models.Property, error)
	Update(property *models.Property) error
	Delete(property *models.Property) error
	List(params ListPropertyParams) ([]models.Property, int64, error)
	IncrementViewCount(propertyID uint) error
	GetByUserID(userID uint) ([]models.Property, error)
	CountByUserID(userID uint) (int64, error)
	ExistsByID(id uint) (bool, error)
	FullTextSearch(query string, limit, offset int) ([]models.Property, int64, error)
	FullTextSearchWithFilters(query string, filters ListSearchParams, limit, offset int) ([]models.Property, int64, error)
}

// PropertyImageRepositoryInterface - Interfaz para repositorio de imágenes
type PropertyImageRepositoryInterface interface {
	Create(image *models.PropertyImage) error
	CreateBatch(images []models.PropertyImage) error
	GetByID(id uint) (*models.PropertyImage, error)
	GetByPropertyID(propertyID uint) ([]models.PropertyImage, error)
	Update(image *models.PropertyImage) error
	Delete(image *models.PropertyImage) error
	DeleteByPropertyID(propertyID uint) error
	GetPrincipalImage(propertyID uint) (*models.PropertyImage, error)
	SetPrincipalImage(propertyID uint, imageID uint) error
	CountByPropertyID(propertyID uint) (int64, error)
}

// UserRepositoryInterface - Interfaz para repositorio de usuarios
type UserRepositoryInterface interface {
	Create(user *models.User) error
	GetByID(id uint) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	GetByGoogleID(googleID string) (*models.User, error)
	Update(user *models.User) error
	Delete(user *models.User) error
	List(limit, offset int) ([]models.User, int64, error)
	ExistsByEmail(email string) (bool, error)
	ToggleActive(user *models.User) error
}

// AuditRepositoryInterface - Interfaz para repositorio de auditoría
type AuditRepositoryInterface interface {
	Create(auditLog *models.AuditLog) error
	CreateAsync(auditLog *models.AuditLog)
	GetByID(id uint) (*models.AuditLog, error)
	GetByEntityID(entity string, entityID uint, limit int) ([]models.AuditLog, error)
	GetByUserID(userID uint, limit int) ([]models.AuditLog, error)
	GetByAction(action string, limit int) ([]models.AuditLog, error)
}

// AdvisorRepositoryInterface - Interfaz para repositorio de asesores
type AdvisorRepositoryInterface interface {
	GetAllActive() ([]models.Advisor, error)
	CountActivePropertiesByAdvisorID(advisorID uint) (int64, error)
	GetByID(id uint) (*models.Advisor, error)
	GetPropertiesByAdvisorID(advisorID uint, status models.PropertyStatus) ([]models.Property, error)
}

// NotificationRepositoryInterface - Interfaz para repositorio de notificaciones
type NotificationRepositoryInterface interface {
	Create(notification *models.Notification) error
}
