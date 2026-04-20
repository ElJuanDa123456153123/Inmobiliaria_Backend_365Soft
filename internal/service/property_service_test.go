package service

import (
	stderrors "errors"
	"testing"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/repository"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPropertyRepository - Mock del repositorio de propiedades
type MockPropertyRepository struct {
	mock.Mock
}

func (m *MockPropertyRepository) Create(property *models.Property) error {
	args := m.Called(property)
	return args.Error(0)
}

func (m *MockPropertyRepository) GetByID(id uint) (*models.Property, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Property), args.Error(1)
}

func (m *MockPropertyRepository) List(params repository.ListPropertyParams) ([]models.Property, int64, error) {
	args := m.Called(params)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.Property), args.Get(1).(int64), args.Error(2)
}

func (m *MockPropertyRepository) Update(property *models.Property) error {
	args := m.Called(property)
	return args.Error(0)
}

func (m *MockPropertyRepository) Delete(property *models.Property) error {
	args := m.Called(property)
	return args.Error(0)
}

func (m *MockPropertyRepository) List(params ListPropertyParams) ([]models.Property, int64, error) {
	args := m.Called(params)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.Property), args.Get(1).(int64), args.Error(2)
}

func (m *MockPropertyRepository) IncrementViewCount(propertyID uint) error {
	args := m.Called(propertyID)
	return args.Error(0)
}

func (m *MockPropertyRepository) GetByUserID(userID uint) ([]models.Property, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Property), args.Error(1)
}

func (m *MockPropertyRepository) CountByUserID(userID uint) (int64, error) {
	args := m.Called(userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPropertyRepository) ExistsByID(id uint) (bool, error) {
	args := m.Called(id)
	return args.Bool(0), args.Error(1)
}

// MockPropertyImageRepository - Mock del repositorio de imágenes
type MockPropertyImageRepository struct {
	mock.Mock
}

func (m *MockPropertyImageRepository) Create(image *models.PropertyImage) error {
	args := m.Called(image)
	return args.Error(0)
}

func (m *MockPropertyImageRepository) CreateBatch(images []models.PropertyImage) error {
	args := m.Called(images)
	return args.Error(0)
}

func (m *MockPropertyImageRepository) GetByID(id uint) (*models.PropertyImage, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PropertyImage), args.Error(1)
}

func (m *MockPropertyImageRepository) GetByPropertyID(propertyID uint) ([]models.PropertyImage, error) {
	args := m.Called(propertyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.PropertyImage), args.Error(1)
}

func (m *MockPropertyImageRepository) Update(image *models.PropertyImage) error {
	args := m.Called(image)
	return args.Error(0)
}

func (m *MockPropertyImageRepository) Delete(image *models.PropertyImage) error {
	args := m.Called(image)
	return args.Error(0)
}

func (m *MockPropertyImageRepository) DeleteByPropertyID(propertyID uint) error {
	args := m.Called(propertyID)
	return args.Error(0)
}

func (m *MockPropertyImageRepository) GetPrincipalImage(propertyID uint) (*models.PropertyImage, error) {
	args := m.Called(propertyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PropertyImage), args.Error(1)
}

func (m *MockPropertyImageRepository) SetPrincipalImage(propertyID uint, imageID uint) error {
	args := m.Called(propertyID, imageID)
	return args.Error(0)
}

func (m *MockPropertyImageRepository) CountByPropertyID(propertyID uint) (int64, error) {
	args := m.Called(propertyID)
	return args.Get(0).(int64), args.Error(1)
}

// MockUserRepository - Mock del repositorio de usuarios
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(id uint) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByGoogleID(googleID string) (*models.User, error) {
	args := m.Called(googleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Update(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) List(limit, offset int) ([]models.User, int64, error) {
	args := m.Called(limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) ExistsByEmail(email string) (bool, error) {
	args := m.Called(email)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) ToggleActive(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

// MockAuditRepository - Mock del repositorio de auditoría
type MockAuditRepository struct {
	mock.Mock
}

func (m *MockAuditRepository) Create(auditLog *models.AuditLog) error {
	args := m.Called(auditLog)
	return args.Error(0)
}

func (m *MockAuditRepository) CreateAsync(auditLog *models.AuditLog) {
	m.Called(auditLog)
}

func (m *MockAuditRepository) GetByID(id uint) (*models.AuditLog, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AuditLog), args.Error(1)
}

func (m *MockAuditRepository) GetByEntityID(entity string, entityID uint, limit int) ([]models.AuditLog, error) {
	args := m.Called(entity, entityID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.AuditLog), args.Error(1)
}

func (m *MockAuditRepository) GetByUserID(userID uint, limit int) ([]models.AuditLog, error) {
	args := m.Called(userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.AuditLog), args.Error(1)
}

func (m *MockAuditRepository) GetByAction(action string, limit int) ([]models.AuditLog, error) {
	args := m.Called(action, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.AuditLog), args.Error(1)
}

// TestPropertyService_CreateProperty_Success - Test creación exitosa de propiedad
func TestPropertyService_CreateProperty_Success(t *testing.T) {
	// Setup
	mockPropertyRepo := new(MockPropertyRepository)
	mockImageRepo := new(MockPropertyImageRepository)
	mockUserRepo := new(MockUserRepository)
	mockAuditRepo := new(MockAuditRepository)

	service := NewPropertyService(mockPropertyRepo, mockImageRepo, mockUserRepo, mockAuditRepo)

	// Mock user
	mockUser := &models.User{
		ID:       1,
		Nombre:   "Juan",
		Apellido: "Perez",
		Email:    "juan@example.com",
		Telefono: "123456789",
	}

	mockUserRepo.On("GetByID", uint(1)).Return(mockUser, nil)

	// Mock property creation
	mockPropertyRepo.On("Create", mock.AnythingOfType("*models.Property")).Return(nil)

	// Mock image creation
	mockImageRepo.On("CreateBatch", mock.AnythingOfType("[]models.PropertyImage")).Return(nil)

	// Test request
	req := &CreatePropertyRequest{
		UserID:           1,
		Titulo:           "Casa bonita en centro",
		TipoOperacion:    "venta",
		Precio:           150000,
		Moneda:           "USD",
		AceptaPermuta:    false,
		Financia:         true,
		TipoPropiedad:    "casa",
		Direccion:        "Av. Principal 123",
		Zona:             "Centro",
		TipoMarcador:     "punto-exacto",
		Latitud:          ptrFloat(-17.78),
		Longitud:         ptrFloat(-63.18),
		MostrarDireccion: true,
		Dormitorios:      "3",
		Banos:            "2",
		Garaje:           "1",
		MetrosEdificados: 150,
		MetrosTerreno:    200,
		Estado:           "Excelente",
		AnoConstruccion:  2020,
		AptoOficina:      false,
		ViviendaSocial:   false,
		Condominio:       false,
		CantidadPisos:    1,
		Descripcion:      "Hermosa casa en el centro con amplios espacios",
		TerminosCondiciones: "Términos estándar",
		Imagenes: []CreatePropertyImageRequest{
			{URL: "/api/imagenes/prop-1.jpg", Orden: 0},
		},
	}

	// Execute
	property, err := service.CreateProperty(req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, property)
	assert.Equal(t, "Casa bonita en centro", property.Titulo)
	assert.Equal(t, 150000.0, property.Precio)
	assert.Equal(t, "USD", property.Moneda)
	assert.Equal(t, models.OperationType("venta"), property.TipoOperacion)

	// Verify mocks were called
	mockUserRepo.AssertExpectations(t)
	mockPropertyRepo.AssertExpectations(t)
	mockImageRepo.AssertExpectations(t)
}

// TestPropertyService_CreateProperty_ValidationError_TituloCorto - Test validación título corto
func TestPropertyService_CreateProperty_ValidationError_TituloCorto(t *testing.T) {
	// Setup
	mockPropertyRepo := new(MockPropertyRepository)
	mockImageRepo := new(MockPropertyImageRepository)
	mockUserRepo := new(MockUserRepository)
	mockAuditRepo := new(MockAuditRepository)

	service := NewPropertyService(mockPropertyRepo, mockImageRepo, mockUserRepo, mockAuditRepo)

	// Mock user
	mockUser := &models.User{ID: 1, Nombre: "Juan", Apellido: "Perez"}
	mockUserRepo.On("GetByID", uint(1)).Return(mockUser, nil)

	// Test request con título muy corto
	req := &CreatePropertyRequest{
		UserID:        1,
		Titulo:        "Casa", // Solo 4 caracteres
		TipoOperacion: "venta",
		Precio:        150000,
		Moneda:        "USD",
		Dormitorios:   "3",
		Banos:         "2",
		MetrosEdificados: 150,
		Estado:        "Excelente",
		Descripcion:   "Descripción válida de la propiedad",
	}

	// Execute
	property, err := service.CreateProperty(req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, property)
	assert.IsType(t, &errors.AppError{}, err)

	appErr := err.(*errors.AppError)
	assert.Equal(t, errors.ErrCodeValidation, appErr.Code)
	assert.Contains(t, appErr.Message, "titulo")
}

// TestPropertyService_CreateProperty_ValidationError_PrecioInvalido - Test validación precio inválido
func TestPropertyService_CreateProperty_ValidationError_PrecioInvalido(t *testing.T) {
	// Setup
	mockPropertyRepo := new(MockPropertyRepository)
	mockImageRepo := new(MockPropertyImageRepository)
	mockUserRepo := new(MockUserRepository)
	mockAuditRepo := new(MockAuditRepository)

	service := NewPropertyService(mockPropertyRepo, mockImageRepo, mockUserRepo, mockAuditRepo)

	// Mock user
	mockUser := &models.User{ID: 1, Nombre: "Juan", Apellido: "Perez"}
	mockUserRepo.On("GetByID", uint(1)).Return(mockUser, nil)

	tests := []struct {
		name        string
		moneda      string
		precio      float64
		expectedErr string
	}{
		{
			name:        "USD muy bajo",
			moneda:      "USD",
			precio:      50,
			expectedErr: "precio",
		},
		{
			name:        "USD muy alto",
			moneda:      "USD",
			precio:      200000000,
			expectedErr: "precio",
		},
		{
			name:        "BOB muy bajo",
			moneda:      "BOB",
			precio:      500,
			expectedErr: "precio",
		},
		{
			name:        "BOB muy alto",
			moneda:      "BOB",
			precio:      600000000,
			expectedErr: "precio",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &CreatePropertyRequest{
				UserID:           1,
				Titulo:           "Título válido de la propiedad",
				TipoOperacion:    "venta",
				Precio:           tt.precio,
				Moneda:           tt.moneda,
				Dormitorios:      "3",
				Banos:            "2",
				MetrosEdificados: 150,
				Estado:           "Excelente",
				Descripcion:      "Descripción válida de la propiedad",
			}

			property, err := service.CreateProperty(req)

			assert.Error(t, err)
			assert.Nil(t, property)
			assert.IsType(t, &errors.AppError{}, err)

			appErr := err.(*errors.AppError)
			assert.Equal(t, errors.ErrCodeValidation, appErr.Code)
			assert.Contains(t, appErr.Message, tt.expectedErr)
		})
	}
}

// TestPropertyService_CreateProperty_ValidationError_CoordenadasInvalidas - Test validación coordenadas
func TestPropertyService_CreateProperty_ValidationError_CoordenadasInvalidas(t *testing.T) {
	// Setup
	mockPropertyRepo := new(MockPropertyRepository)
	mockImageRepo := new(MockPropertyImageRepository)
	mockUserRepo := new(MockUserRepository)
	mockAuditRepo := new(MockAuditRepository)

	service := NewPropertyService(mockPropertyRepo, mockImageRepo, mockUserRepo, mockAuditRepo)

	// Mock user
	mockUser := &models.User{ID: 1, Nombre: "Juan", Apellido: "Perez"}
	mockUserRepo.On("GetByID", uint(1)).Return(mockUser, nil)

	tests := []struct {
		name        string
		latitud     *float64
		longitud    *float64
		expectedErr string
	}{
		{
			name:        "Latitud fuera de rango (norte)",
			latitud:     ptrFloat(-5),
			longitud:    ptrFloat(-63),
			expectedErr: "latitud",
		},
		{
			name:        "Latitud fuera de rango (sur)",
			latitud:     ptrFloat(-25),
			longitud:    ptrFloat(-63),
			expectedErr: "latitud",
		},
		{
			name:        "Longitud fuera de rango (este)",
			latitud:     ptrFloat(-17),
			longitud:    ptrFloat(-50),
			expectedErr: "longitud",
		},
		{
			name:        "Longitud fuera de rango (oeste)",
			latitud:     ptrFloat(-17),
			longitud:    ptrFloat(-75),
			expectedErr: "longitud",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &CreatePropertyRequest{
				UserID:           1,
				Titulo:           "Título válido de la propiedad",
				TipoOperacion:    "venta",
				Precio:           150000,
				Moneda:           "USD",
				Latitud:          tt.latitud,
				Longitud:         tt.longitud,
				Dormitorios:      "3",
				Banos:            "2",
				MetrosEdificados: 150,
				Estado:           "Excelente",
				Descripcion:      "Descripción válida de la propiedad",
			}

			property, err := service.CreateProperty(req)

			assert.Error(t, err)
			assert.Nil(t, property)
			assert.IsType(t, &errors.AppError{}, err)

			appErr := err.(*errors.AppError)
			assert.Equal(t, errors.ErrCodeValidation, appErr.Code)
			assert.Contains(t, appErr.Message, tt.expectedErr)
		})
	}
}

// TestPropertyService_GetPropertyByID_NotFound - Test propiedad no encontrada
func TestPropertyService_GetPropertyByID_NotFound(t *testing.T) {
	// Setup
	mockPropertyRepo := new(MockPropertyRepository)
	mockImageRepo := new(MockPropertyImageRepository)
	mockUserRepo := new(MockUserRepository)
	mockAuditRepo := new(MockAuditRepository)

	service := NewPropertyService(mockPropertyRepo, mockImageRepo, mockUserRepo, mockAuditRepo)

	// Mock repositorio retorna error (no encontrado)
	mockPropertyRepo.On("GetByID", uint(999)).Return(nil, stderrors.New("not found"))

	// Execute
	property, err := service.GetPropertyByID(999)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, property)
	assert.IsType(t, &errors.AppError{}, err)

	appErr := err.(*errors.AppError)
	assert.Equal(t, errors.ErrCodeNotFound, appErr.Code)
	assert.Contains(t, appErr.Message, "Propiedad no encontrada")

	mockPropertyRepo.AssertExpectations(t)
}

// TestPropertyService_UpdatePropertyStatus_Success - Test actualización de estado exitosa
func TestPropertyService_UpdatePropertyStatus_Success(t *testing.T) {
	// Setup
	mockPropertyRepo := new(MockPropertyRepository)
	mockImageRepo := new(MockPropertyImageRepository)
	mockUserRepo := new(MockUserRepository)
	mockAuditRepo := new(MockAuditRepository)

	service := NewPropertyService(mockPropertyRepo, mockImageRepo, mockUserRepo, mockAuditRepo)

	// Mock propiedad existente
	existingProperty := &models.Property{
		ID:             1,
		UserID:         1,
		Titulo:         "Casa bonita",
		EstadoPropiedad: models.PropertyStatusActivo,
	}

	mockPropertyRepo.On("GetByID", uint(1)).Return(existingProperty, nil)
	mockPropertyRepo.On("Update", mock.AnythingOfType("*models.Property")).Return(nil)

	// Test request
	req := &UpdatePropertyStatusRequest{
		PropertyID: 1,
		UserID:     1,
		NewStatus:  models.PropertyStatusVendido,
	}

	// Execute
	property, err := service.UpdatePropertyStatus(req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, property)
	assert.Equal(t, models.PropertyStatusVendido, property.EstadoPropiedad)

	mockPropertyRepo.AssertExpectations(t)
}

// TestPropertyService_UpdatePropertyStatus_Forbidden - Test usuario no es dueño
func TestPropertyService_UpdatePropertyStatus_Forbidden(t *testing.T) {
	// Setup
	mockPropertyRepo := new(MockPropertyRepository)
	mockImageRepo := new(MockPropertyImageRepository)
	mockUserRepo := new(MockUserRepository)
	mockAuditRepo := new(MockAuditRepository)

	service := NewPropertyService(mockPropertyRepo, mockImageRepo, mockUserRepo, mockAuditRepo)

	// Mock propiedad de otro usuario
	existingProperty := &models.Property{
		ID:     1,
		UserID: 999, // Diferente usuario
		Titulo: "Casa bonita",
	}

	mockPropertyRepo.On("GetByID", uint(1)).Return(existingProperty, nil)

	// Test request
	req := &UpdatePropertyStatusRequest{
		PropertyID: 1,
		UserID:     1, // Usuario diferente
		NewStatus:  models.PropertyStatusVendido,
	}

	// Execute
	property, err := service.UpdatePropertyStatus(req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, property)
	assert.IsType(t, &errors.AppError{}, err)

	appErr := err.(*errors.AppError)
	assert.Equal(t, errors.ErrCodeForbidden, appErr.Code)
	assert.Contains(t, appErr.Message, "permiso")

	mockPropertyRepo.AssertExpectations(t)
}

// TestPropertyService_DeleteProperty_Success - Test eliminación exitosa
func TestPropertyService_DeleteProperty_Success(t *testing.T) {
	// Setup
	mockPropertyRepo := new(MockPropertyRepository)
	mockImageRepo := new(MockPropertyImageRepository)
	mockUserRepo := new(MockUserRepository)
	mockAuditRepo := new(MockAuditRepository)

	service := NewPropertyService(mockPropertyRepo, mockImageRepo, mockUserRepo, mockAuditRepo)

	// Mock propiedad existente
	existingProperty := &models.Property{
		ID:     1,
		UserID: 1,
		Titulo: "Casa bonita",
	}

	mockPropertyRepo.On("GetByID", uint(1)).Return(existingProperty, nil)
	mockPropertyRepo.On("Delete", mock.AnythingOfType("*models.Property")).Return(nil)

	// Test request
	req := &DeletePropertyRequest{
		PropertyID: 1,
		UserID:     1,
	}

	// Execute
	err := service.DeleteProperty(req)

	// Assert
	assert.NoError(t, err)

	mockPropertyRepo.AssertExpectations(t)
}

// Helper function para crear punteros a float64
func ptrFloat(f float64) *float64 {
	return &f
}
