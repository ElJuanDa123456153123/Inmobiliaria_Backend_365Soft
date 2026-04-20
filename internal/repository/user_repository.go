package repository

import (
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
)

// UserRepository - Repositorio para operaciones de base de datos de usuarios
type UserRepository struct {
	DB *config.Database
}

// NewUserRepository - Crea un nuevo repositorio de usuarios
func NewUserRepository(db *config.Database) *UserRepository {
	return &UserRepository{DB: db}
}

// Create - Crea un nuevo usuario
func (r *UserRepository) Create(user *models.User) error {
	return r.DB.DB.Create(user).Error
}

// GetByID - Obtiene un usuario por ID
func (r *UserRepository) GetByID(id uint) (*models.User, error) {
	var user models.User
	err := r.DB.DB.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail - Obtiene un usuario por email
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.DB.DB.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByGoogleID - Obtiene un usuario por Google ID
func (r *UserRepository) GetByGoogleID(googleID string) (*models.User, error) {
	var user models.User
	err := r.DB.DB.Where("google_id = ?", googleID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update - Actualiza un usuario
func (r *UserRepository) Update(user *models.User) error {
	return r.DB.DB.Save(user).Error
}

// Delete - Soft delete de un usuario
func (r *UserRepository) Delete(user *models.User) error {
	return r.DB.DB.Delete(user).Error
}

// List - Lista todos los usuarios con paginación
func (r *UserRepository) List(limit, offset int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	// Contar total
	if err := r.DB.DB.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Obtener usuarios
	err := r.DB.DB.
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&users).Error

	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// ExistsByEmail - Verifica si existe un usuario con el email dado
func (r *UserRepository) ExistsByEmail(email string) (bool, error) {
	var count int64
	err := r.DB.DB.Model(&models.User{}).
		Where("email = ?", email).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ToggleActive - Alterna el estado activo de un usuario
func (r *UserRepository) ToggleActive(user *models.User) error {
	user.Activo = !user.Activo
	return r.DB.DB.Save(user).Error
}
