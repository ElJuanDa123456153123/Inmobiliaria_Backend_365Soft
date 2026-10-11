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

// ========== MOCKS (implementan las interfaces actuales de repository) ==========

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

func (m *MockPropertyRepository) GetByIDWithUser(id uint) (*models.Property, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Property), args.Error(1)
}

func (m *MockPropertyRepository) GetByIDWithRelations(id uint) (*models.Property, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Property), args.Error(1)
}

func (m *MockPropertyRepository) Update(property *models.Property) error {
	args := m.Called(property)
	return args.Error(0)
}

func (m *MockPropertyRepository) Delete(property *models.Property) error {
	args := m.Called(property)
	return args.Error(0)
}

func (m *MockPropertyRepository) List(params repository.ListPropertyParams) ([]models.Property, int64, error) {
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

func (m *MockPropertyRepository) FullTextSearch(query string, limit, offset int) ([]models.Property, int64, error) {
	args := m.Called(query, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.Property), args.Get(1).(int64), args.Error(2)
}

func (m *MockPropertyRepository) FullTextSearchWithFilters(query string, filters repository.ListSearchParams, limit, offset int) ([]models.Property, int64, error) {
	args := m.Called(query, filters, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.Property), args.Get(1).(int64), args.Error(2)
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

// ========== FIXTURES ==========

// newServiceWithMocks - Crea el servicio con mocks (db nil: los tests usan rutas que no tocan la BD)
func newServiceWithMocks() (*PropertyService, *MockPropertyRepository, *MockPropertyImageRepository, *MockUserRepository, *MockAuditRepository) {
	mockPropertyRepo := new(MockPropertyRepository)
	mockImageRepo := new(MockPropertyImageRepository)
	mockUserRepo := new(MockUserRepository)
	mockAuditRepo := new(MockAuditRepository)

	svc := NewPropertyService(mockPropertyRepo, mockImageRepo, mockUserRepo, mockAuditRepo, nil, nil)
	return svc, mockPropertyRepo, mockImageRepo, mockUserRepo, mockAuditRepo
}

// validCreateRequestBase - Request válido base (UserID=0 usa el admin harcodeado y no requiere BD)
func validCreateRequestBase() *CreatePropertyRequest {
	return &CreatePropertyRequest{
		UserID:           0,
		Titulo:           "Casa bonita en zona centro",
		TipoOperacion:    "venta",
		Precio:           150000,
		Moneda:           "USD",
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
		Descripcion:      "Hermosa casa en el centro con amplios espacios",
	}
}

// ========== TESTS ==========

// TestPropertyService_CreateProperty_Success - Test creación exitosa (ruta admin harcodeado, sin BD)
func TestPropertyService_CreateProperty_Success(t *testing.T) {
	svc, mockPropertyRepo, _, _, _ := newServiceWithMocks()

	mockPropertyRepo.On("Create", mock.AnythingOfType("*models.Property")).Return(nil).Run(func(args mock.Arguments) {
		// Simular que GORM asigna el ID al crear
		args.Get(0).(*models.Property).ID = 1
	})

	property, err := svc.CreateProperty(validCreateRequestBase())

	assert.NoError(t, err)
	assert.NotNil(t, property)
	assert.Equal(t, "Casa bonita en zona centro", property.Titulo)
	assert.Equal(t, 150000.0, property.Precio)
	assert.Equal(t, "USD", property.Moneda)
	assert.Equal(t, models.OperationType("venta"), property.TipoOperacion)
	assert.Equal(t, models.PropertyStatusActivo, property.EstadoPropiedad, "la propiedad se publica activa (aprobación automática)")
	assert.Equal(t, 0, property.VistasContador, "el contador de vistas inicia en 0")
	assert.Equal(t, "Administrador Sistema", property.VendedorNombre, "se guarda el snapshot del vendedor")
	assert.NotNil(t, property.FechaPublicacion)

	mockPropertyRepo.AssertExpectations(t)
}

// TestPropertyService_CreateProperty_UserNotFound - El usuario no existe en la BD
func TestPropertyService_CreateProperty_UserNotFound(t *testing.T) {
	svc, _, _, mockUserRepo, _ := newServiceWithMocks()

	mockUserRepo.On("GetByID", uint(77)).Return(nil, stderrors.New("record not found"))

	req := validCreateRequestBase()
	req.UserID = 77

	property, err := svc.CreateProperty(req)

	assert.Error(t, err)
	assert.Nil(t, property)

	appErr, ok := err.(*errors.AppError)
	assert.True(t, ok)
	assert.Equal(t, errors.ErrCodeUserNotFound, appErr.Code)

	mockUserRepo.AssertExpectations(t)
}

// TestPropertyService_CreateProperty_ValidationErrors - Errores de validación del request
func TestPropertyService_CreateProperty_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(req *CreatePropertyRequest)
		expectedErr string
	}{
		{
			name:        "titulo_muy_corto",
			mutate:      func(req *CreatePropertyRequest) { req.Titulo = "Casa" },
			expectedErr: "titulo",
		},
		{
			name:        "tipo_operacion_invalido",
			mutate:      func(req *CreatePropertyRequest) { req.TipoOperacion = "permuta" },
			expectedErr: "tipo_operacion",
		},
		{
			name:        "moneda_invalida",
			mutate:      func(req *CreatePropertyRequest) { req.Moneda = "EUR" },
			expectedErr: "moneda",
		},
		{
			name:        "precio_venta_usd_bajo_el_minimo",
			mutate:      func(req *CreatePropertyRequest) { req.Precio = 50 },
			expectedErr: "precio",
		},
		{
			name:        "precio_venta_bob_sobre_el_maximo",
			mutate:      func(req *CreatePropertyRequest) { req.Precio = 600000000; req.Moneda = "BOB" },
			expectedErr: "precio",
		},
		{
			name:        "coordenadas_fuera_de_bolivia",
			mutate:      func(req *CreatePropertyRequest) { req.Latitud = ptrFloat(10.0); req.Longitud = ptrFloat(-63.18) },
			expectedErr: "latitud",
		},
		{
			name:        "metros_edificados_invalidos",
			mutate:      func(req *CreatePropertyRequest) { req.MetrosEdificados = 0 },
			expectedErr: "metros_edificados",
		},
		{
			name:        "estado_vacio",
			mutate:      func(req *CreatePropertyRequest) { req.Estado = "" },
			expectedErr: "estado",
		},
		{
			name:        "descripcion_muy_corta",
			mutate:      func(req *CreatePropertyRequest) { req.Descripcion = "Corta" },
			expectedErr: "descripcion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _, _, _ := newServiceWithMocks()

			req := validCreateRequestBase()
			tt.mutate(req)

			property, err := svc.CreateProperty(req)

			assert.Error(t, err)
			assert.Nil(t, property)

			appErr, ok := err.(*errors.AppError)
			assert.True(t, ok)
			assert.Equal(t, errors.ErrCodeValidation, appErr.Code)
			assert.Contains(t, appErr.Message, tt.expectedErr)
		})
	}
}

// TestPropertyService_GetPropertyByID_NotFound - Propiedad inexistente
func TestPropertyService_GetPropertyByID_NotFound(t *testing.T) {
	svc, mockPropertyRepo, _, _, _ := newServiceWithMocks()

	mockPropertyRepo.On("GetByID", uint(999)).Return(nil, stderrors.New("record not found"))

	property, err := svc.GetPropertyByID(999)

	assert.Error(t, err)
	assert.Nil(t, property)

	appErr, ok := err.(*errors.AppError)
	assert.True(t, ok)
	assert.Equal(t, errors.ErrCodePropertyNotFound, appErr.Code)

	mockPropertyRepo.AssertExpectations(t)
}

// TestPropertyService_UpdatePropertyStatus_Success - Actualización de estado por el dueño
func TestPropertyService_UpdatePropertyStatus_Success(t *testing.T) {
	svc, mockPropertyRepo, _, _, _ := newServiceWithMocks()

	existing := &models.Property{ID: 1, UserID: 1, Titulo: "Casa bonita", EstadoPropiedad: models.PropertyStatusActivo}
	mockPropertyRepo.On("GetByID", uint(1)).Return(existing, nil)
	mockPropertyRepo.On("Update", mock.AnythingOfType("*models.Property")).Return(nil)

	property, err := svc.UpdatePropertyStatus(&UpdatePropertyStatusRequest{
		PropertyID: 1,
		UserID:     1,
		NewStatus:  models.PropertyStatusVendido,
	})

	assert.NoError(t, err)
	assert.NotNil(t, property)
	assert.Equal(t, models.PropertyStatusVendido, property.EstadoPropiedad)

	mockPropertyRepo.AssertExpectations(t)
}

// TestPropertyService_UpdatePropertyStatus_Forbidden - Un usuario que no es dueño no puede modificar
func TestPropertyService_UpdatePropertyStatus_Forbidden(t *testing.T) {
	svc, mockPropertyRepo, _, _, _ := newServiceWithMocks()

	existing := &models.Property{ID: 1, UserID: 999, Titulo: "Casa de otro usuario"}
	mockPropertyRepo.On("GetByID", uint(1)).Return(existing, nil)

	property, err := svc.UpdatePropertyStatus(&UpdatePropertyStatusRequest{
		PropertyID: 1,
		UserID:     1, // distinto del dueño (999)
		NewStatus:  models.PropertyStatusVendido,
	})

	assert.Error(t, err)
	assert.Nil(t, property)

	appErr, ok := err.(*errors.AppError)
	assert.True(t, ok)
	assert.Equal(t, errors.ErrCodeForbidden, appErr.Code)
	assert.Contains(t, appErr.Message, "permiso")

	mockPropertyRepo.AssertExpectations(t)
}

// TestPropertyService_UpdatePropertyStatus_EstadoInvalido - Estado no permitido
func TestPropertyService_UpdatePropertyStatus_EstadoInvalido(t *testing.T) {
	svc, mockPropertyRepo, _, _, _ := newServiceWithMocks()

	existing := &models.Property{ID: 1, UserID: 1, Titulo: "Casa bonita"}
	mockPropertyRepo.On("GetByID", uint(1)).Return(existing, nil)

	property, err := svc.UpdatePropertyStatus(&UpdatePropertyStatusRequest{
		PropertyID: 1,
		UserID:     1,
		NewStatus:  models.PropertyStatus("pendiente"),
	})

	assert.Error(t, err)
	assert.Nil(t, property)

	appErr, ok := err.(*errors.AppError)
	assert.True(t, ok)
	assert.Equal(t, errors.ErrCodeValidation, appErr.Code)

	mockPropertyRepo.AssertExpectations(t)
}

// TestPropertyService_DeleteProperty_Success - Eliminación lógica por el dueño
func TestPropertyService_DeleteProperty_Success(t *testing.T) {
	svc, mockPropertyRepo, _, _, _ := newServiceWithMocks()

	// UserID=0 evita el acceso a la BD para el contador de la suscripción
	existing := &models.Property{ID: 1, UserID: 0, Titulo: "Casa bonita"}
	mockPropertyRepo.On("GetByID", uint(1)).Return(existing, nil)
	mockPropertyRepo.On("Delete", mock.AnythingOfType("*models.Property")).Return(nil)

	err := svc.DeleteProperty(&DeletePropertyRequest{PropertyID: 1, UserID: 0})

	assert.NoError(t, err)

	mockPropertyRepo.AssertExpectations(t)
}

// TestPropertyService_DeleteProperty_Forbidden - Un usuario que no es dueño no puede eliminar
func TestPropertyService_DeleteProperty_Forbidden(t *testing.T) {
	svc, mockPropertyRepo, _, _, _ := newServiceWithMocks()

	existing := &models.Property{ID: 1, UserID: 999, Titulo: "Casa de otro usuario"}
	mockPropertyRepo.On("GetByID", uint(1)).Return(existing, nil)

	err := svc.DeleteProperty(&DeletePropertyRequest{PropertyID: 1, UserID: 1})

	assert.Error(t, err)

	appErr, ok := err.(*errors.AppError)
	assert.True(t, ok)
	assert.Equal(t, errors.ErrCodeForbidden, appErr.Code)
	assert.Contains(t, appErr.Message, "permiso")

	mockPropertyRepo.AssertExpectations(t)
}

// Helper function para crear punteros a float64
func ptrFloat(f float64) *float64 {
	return &f
}
