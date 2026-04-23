package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/handlers"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/middleware"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// PropertyAPIIntegrationTestSuite - Suite de tests de integración para la API de propiedades
type PropertyAPIIntegrationTestSuite struct {
	suite.Suite
	app       *fiber.App
	db        *config.Database
	testUser  *models.User
	testToken string
}

// SetupSuite - Configuración inicial de la suite
func (s *PropertyAPIIntegrationTestSuite) SetupSuite() {
	// Crear app de Fiber
	s.app = fiber.New(fiber.Config{
		AppName: "Test API",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Conectar a BD de test
	s.db = getConfiguredTestDB()
	assert.NotNil(s.T(), s.db)

	// Crear usuario de prueba
	passwordHash, _ := hashPassword("Test123*")
	s.testUser = &models.User{
		Nombre:   "Test",
		Apellido: "User",
		Email:    "test@example.com",
		Password: passwordHash,
		Telefono: "123456789",
		Rol:      models.RoleUsuario,
		Activo:   true,
	}
	s.db.DB.Create(s.testUser)

	// Generar token JWT
	token, _ := generateJWT(s.testUser.ID, s.testUser.Email, string(s.testUser.Rol))
	s.testToken = token

	// Inicializar handlers
	propertyHandler, err := handlers.NewPropertyHandler()
	assert.NoError(s.T(), err)

	// Configurar rutas
	api := s.app.Group("/api")

	// Rutas de propiedades
	api.Get("/properties", propertyHandler.ListProperties)
	api.Get("/properties/:id", propertyHandler.GetProperty)
	api.Post("/properties", middleware.AuthMiddleware(), propertyHandler.CreateProperty)
	api.Get("/properties/my", middleware.AuthMiddleware(), propertyHandler.GetUserProperties)
	api.Patch("/properties/:id/status", middleware.AuthMiddleware(), propertyHandler.UpdatePropertyStatus)
	api.Delete("/properties/:id", middleware.AuthMiddleware(), propertyHandler.DeleteProperty)
}

// TearDownSuite - Limpieza después de todos los tests
func (s *PropertyAPIIntegrationTestSuite) TearDownSuite() {
	// Limpiar datos de test
	s.db.DB.Where("email LIKE ?", "%@example.com").Delete(&models.User{})
	s.db.DB.Where("titulo LIKE ?", "Test Property%").Delete(&models.Property{})
	s.db.Close()
}

// SetupTest - Configuración antes de cada test
func (s *PropertyAPIIntegrationTestSuite) SetupTest() {
	// Limpiar propiedades de test antes de cada test
	s.db.DB.Where("titulo LIKE ?", "Test Property%").Delete(&models.Property{})
}

// TestListProperties_Empty - Test listar propiedades cuando no hay ninguna
func (s *PropertyAPIIntegrationTestSuite) TestListProperties_Empty() {
	req := httptest.NewRequest("GET", "/api/properties", nil)
	resp, err := s.app.Test(req)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), float64(0), result["total"])
	assert.Equal(s.T(), float64(0), result["page"])
	assert.Equal(s.T(), float64(20), result["limit"])
}

// TestListProperties_WithFilters - Test listar propiedades con filtros
func (s *PropertyAPIIntegrationTestSuite) TestListProperties_WithFilters() {
	// Crear propiedades de prueba
	property1 := models.Property{
		UserID:         s.testUser.ID,
		Titulo:         "Test Property 1 - Venta",
		TipoOperacion:  models.OperationTypeVenta,
		Precio:         150000,
		Moneda:         "USD",
		TipoPropiedad:  models.PropertyTypeCasa,
		Zona:           "Centro",
		Dormitorios:    "3",
		Banos:          "2",
		MetrosEdificados: 150,
		Estado:         "Excelente",
		Descripcion:    "Test description",
		EstadoPropiedad: models.PropertyStatusActivo,
	}

	property2 := models.Property{
		UserID:         s.testUser.ID,
		Titulo:         "Test Property 2 - Alquiler",
		TipoOperacion:  models.OperationTypeAlquiler,
		Precio:         5000,
		Moneda:         "USD",
		TipoPropiedad:  models.PropertyTypeDepartamento,
		Zona:           "Sur",
		Dormitorios:    "2",
		Banos:          "1",
		MetrosEdificados: 80,
		Estado:         "Bueno",
		Descripcion:    "Test description 2",
		EstadoPropiedad: models.PropertyStatusActivo,
	}

	s.db.DB.Create(&property1)
	s.db.DB.Create(&property2)

	tests := []struct {
		name           string
		query          string
		expectedCount  int
		expectedStatus int
	}{
		{
			name:           "Sin filtros",
			query:          "/api/properties",
			expectedCount:  2,
			expectedStatus: 200,
		},
		{
			name:           "Filtro por tipo de operación - venta",
			query:          "/api/properties?tipo_operacion=venta",
			expectedCount:  1,
			expectedStatus: 200,
		},
		{
			name:           "Filtro por tipo de operación - alquiler",
			query:          "/api/properties?tipo_operacion=alquiler",
			expectedCount:  1,
			expectedStatus: 200,
		},
		{
			name:           "Filtro por moneda - USD",
			query:          "/api/properties?moneda=USD",
			expectedCount:  2,
			expectedStatus: 200,
		},
		{
			name:           "Filtro por zona - Centro",
			query:          "/api/properties?zona=Centro",
			expectedCount:  1,
			expectedStatus: 200,
		},
		{
			name:           "Filtro por precio mínimo",
			query:          "/api/properties?precio_min=10000",
			expectedCount:  1,
			expectedStatus: 200,
		},
		{
			name:           "Paginación - página 1",
			query:          "/api/properties?page=1&limit=1",
			expectedCount:  1,
			expectedStatus: 200,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.query, nil)
			resp, err := s.app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			assert.NoError(t, err)

			propiedades := result["propiedades"].([]interface{})
			assert.Equal(t, tt.expectedCount, len(propiedades))
		})
	}
}

// TestCreateProperty_Success - Test crear propiedad exitosamente
func (s *PropertyAPIIntegrationTestSuite) TestCreateProperty_Success() {
	requestBody := map[string]interface{}{
		"titulo":           "Test Property - Create",
		"tipo_operacion":    "venta",
		"precio":           200000,
		"moneda":           "USD",
		"acepta_permuta":   false,
		"financia":         true,
		"tipo_propiedad":   "casa",
		"direccion":        "Test Address 123",
		"zona":             "Norte",
		"tipo_marcador":    "punto-exacto",
		"dormitorios":      "4",
		"banos":            "3",
		"garaje":           "2",
		"metros_edificados": 200,
		"metros_terreno":   300,
		"estado":           "Excelente",
		"ano_construccion": 2022,
		"apto_oficina":     false,
		"vivienda_social":  false,
		"condominio":       false,
		"cantidad_pisos":   2,
		"descripcion":      "Test property description with enough characters",
		"terminos_condiciones": "Test terms",
		"imagenes": []map[string]interface{}{
			{"url": "/api/imagenes/test.jpg", "orden": 0},
		},
	}

	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/properties", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 201, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), "Propiedad creada exitosamente", result["message"])
	assert.NotNil(s.T(), result["property"])

	property := result["property"].(map[string]interface{})
	assert.Equal(s.T(), "Test Property - Create", property["titulo"])
	assert.Equal(s.T(), 200000.0, property["precio"])
	assert.Equal(s.T(), "USD", property["moneda"])
}

// TestCreateProperty_ValidationError - Test crear propiedad con errores de validación
func (s *PropertyAPIIntegrationTestSuite) TestCreateProperty_ValidationError() {
	tests := []struct {
		name          string
		requestBody   map[string]interface{}
		expectedError string
	}{
		{
			name: "Título muy corto",
			requestBody: map[string]interface{}{
				"titulo":        "Casa",
				"tipo_operacion": "venta",
				"precio":        150000,
				"moneda":        "USD",
				"dormitorios":   "3",
				"banos":         "2",
				"metros_edificados": 150,
				"estado":        "Excelente",
				"descripcion":   "Descripción válida de la propiedad",
			},
			expectedError: "título",
		},
		{
			name: "Precio inválido",
			requestBody: map[string]interface{}{
				"titulo":        "Título válido de la propiedad",
				"tipo_operacion": "venta",
				"precio":        50, // Muy bajo para USD
				"moneda":        "USD",
				"dormitorios":   "3",
				"banos":         "2",
				"metros_edificados": 150,
				"estado":        "Excelente",
				"descripcion":   "Descripción válida de la propiedad",
			},
			expectedError: "precio",
		},
		{
			name: "Descripción muy corta",
			requestBody: map[string]interface{}{
				"titulo":        "Título válido de la propiedad",
				"tipo_operacion": "venta",
				"precio":        150000,
				"moneda":        "USD",
				"dormitorios":   "3",
				"banos":         "2",
				"metros_edificados": 150,
				"estado":        "Excelente",
				"descripcion":   "Corta",
			},
			expectedError: "descripción",
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/properties", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+s.testToken)

			resp, err := s.app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, 400, resp.StatusCode)

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			assert.NoError(t, err)

			assert.Contains(t, result["error"], tt.expectedError)
		})
	}
}

// TestGetProperty_Success - Test obtener propiedad por ID
func (s *PropertyAPIIntegrationTestSuite) TestGetProperty_Success() {
	// Crear propiedad de prueba
	property := models.Property{
		UserID:         s.testUser.ID,
		Titulo:         "Test Property - Get",
		TipoOperacion:  models.OperationTypeVenta,
		Precio:         180000,
		Moneda:         "USD",
		TipoPropiedad:  models.PropertyTypeCasa,
		Zona:           "Centro",
		Dormitorios:    "3",
		Banos:          "2",
		MetrosEdificados: 180,
		Estado:         "Excelente",
		Descripcion:    "Test description for get property",
		EstadoPropiedad: models.PropertyStatusActivo,
		VistasContador: 0,
	}
	s.db.DB.Create(&property)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/properties/%d", property.ID), nil)
	resp, err := s.app.Test(req)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), "Test Property - Get", result["titulo"])
	assert.Equal(s.T(), float64(180000), result["precio"])
	assert.Equal(s.T(), "USD", result["moneda"])
}

// TestGetProperty_NotFound - Test obtener propiedad inexistente
func (s *PropertyAPIIntegrationTestSuite) TestGetProperty_NotFound() {
	req := httptest.NewRequest("GET", "/api/properties/999999", nil)
	resp, err := s.app.Test(req)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 404, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(s.T(), err)

	assert.Contains(s.T(), result["error"], "no encontrada")
}

// TestUpdatePropertyStatus_Success - Test actualizar estado de propiedad
func (s *PropertyAPIIntegrationTestSuite) TestUpdatePropertyStatus_Success() {
	// Crear propiedad de prueba
	property := models.Property{
		UserID:         s.testUser.ID,
		Titulo:         "Test Property - Update Status",
		TipoOperacion:  models.OperationTypeVenta,
		Precio:         200000,
		Moneda:         "USD",
		TipoPropiedad:  models.PropertyTypeCasa,
		Zona:           "Centro",
		Dormitorios:    "3",
		Banos:          "2",
		MetrosEdificados: 200,
		Estado:         "Excelente",
		Descripcion:    "Test description",
		EstadoPropiedad: models.PropertyStatusActivo,
	}
	s.db.DB.Create(&property)

	requestBody := map[string]interface{}{
		"estado": "vendido",
	}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/properties/%d/status", property.ID), bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(s.T(), err)

	propiedad := result["property"].(map[string]interface{})
	assert.Equal(s.T(), "vendido", propiedad["estado_propiedad"])
}

// TestDeleteProperty_Success - Test eliminar propiedad
func (s *PropertyAPIIntegrationTestSuite) TestDeleteProperty_Success() {
	// Crear propiedad de prueba
	property := models.Property{
		UserID:         s.testUser.ID,
		Titulo:         "Test Property - Delete",
		TipoOperacion:  models.OperationTypeVenta,
		Precio:         250000,
		Moneda:         "USD",
		TipoPropiedad:  models.PropertyTypeCasa,
		Zona:           "Norte",
		Dormitorios:    "4",
		Banos:          "3",
		MetrosEdificados: 250,
		Estado:         "Excelente",
		Descripcion:    "Test description for delete",
		EstadoPropiedad: models.PropertyStatusActivo,
	}
	s.db.DB.Create(&property)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/properties/%d", property.ID), nil)
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(s.T(), err)

	assert.Contains(s.T(), result["message"], "eliminada")

	// Verificar que fue soft delete (ya no debería aparecer en listas)
	req2 := httptest.NewRequest("GET", "/api/properties", nil)
	resp2, err := s.app.Test(req2)
	assert.NoError(s.T(), err)

	var result2 map[string]interface{}
	err = json.NewDecoder(resp2.Body).Decode(&result2)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), float64(0), result2["total"])
}

// TestGetUserProperties_Success - Test obtener propiedades del usuario
func (s *PropertyAPIIntegrationTestSuite) TestGetUserProperties_Success() {
	// Crear propiedades del usuario
	property1 := models.Property{
		UserID:         s.testUser.ID,
		Titulo:         "Test Property - User 1",
		TipoOperacion:  models.OperationTypeVenta,
		Precio:         150000,
		Moneda:         "USD",
		TipoPropiedad:  models.PropertyTypeCasa,
		Zona:           "Centro",
		Dormitorios:    "3",
		Banos:          "2",
		MetrosEdificados: 150,
		Estado:         "Excelente",
		Descripcion:    "Test description",
		EstadoPropiedad: models.PropertyStatusActivo,
	}

	property2 := models.Property{
		UserID:         s.testUser.ID,
		Titulo:         "Test Property - User 2",
		TipoOperacion:  models.OperationTypeAlquiler,
		Precio:         5000,
		Moneda:         "USD",
		TipoPropiedad:  models.PropertyTypeDepartamento,
		Zona:           "Sur",
		Dormitorios:    "2",
		Banos:          "1",
		MetrosEdificados: 80,
		Estado:         "Bueno",
		Descripcion:    "Test description 2",
		EstadoPropiedad: models.PropertyStatusActivo,
	}

	s.db.DB.Create(&property1)
	s.db.DB.Create(&property2)

	req := httptest.NewRequest("GET", "/api/properties/my", nil)
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 200, resp.StatusCode)

	var result []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), 2, len(result))
	assert.Equal(s.T(), "Test Property - User 1", result[0]["titulo"])
}

// TestPropertyAPIIntegrationTestSuite - Ejecutar la suite
func TestPropertyAPIIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PropertyAPIIntegrationTestSuite))
}

// Helper functions

func getConfiguredTestDB() *config.Database {
	// Usar DATABASE_URL de test o crear una
	dsn := "host=localhost port=5432 user=postgres password=password dbname=inmobiliaria_test sslmode=disable"
	db, err := config.NewDatabaseFromDSN(dsn)
	if err != nil {
		panic("Error conectando a BD de test: " + err.Error())
	}
	return db
}

func hashPassword(password string) (string, error) {
	// Simplificado - en producción usar bcrypt
	return password + "_hashed", nil
}

func generateJWT(userID uint, email, rol string) (string, error) {
	// Simplificado - en producción usar jwt real
	return "test_token_" + fmt.Sprint(userID), nil
}
