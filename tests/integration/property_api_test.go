//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// PropertyAPISuite - Pruebas de integración de los endpoints de propiedades
type PropertyAPISuite struct {
	suite.Suite
	owner      *models.User
	other      *models.User
	plan       *models.Plan
	ownerToken string
	otherToken string
}

func (s *PropertyAPISuite) SetupTest() {
	cleanupTables()

	s.plan = createTestPlan("Plan Gratuito Prueba", 2) // límite: 2 publicaciones
	s.owner = createTestUser("propietario@integration.test")
	s.other = createTestUser("intruso@integration.test")
	createTestSubscription(s.owner.ID, s.plan.ID, 0) // 0 de 2 usadas

	s.ownerToken = generateTestToken(s.owner.ID, s.owner.Email, string(s.owner.Rol))
	s.otherToken = generateTestToken(s.other.ID, s.other.Email, string(s.other.Rol))
}

// validPropertyBody - Cuerpo válido para POST /api/properties
func validPropertyBody(titulo string) []byte {
	body, _ := json.Marshal(map[string]interface{}{
		"titulo":            titulo,
		"tipo_operacion":    "venta",
		"precio":            120000,
		"moneda":            "USD",
		"tipo_propiedad":    "casa",
		"direccion":         "Av. Siempre Viva 742",
		"zona":              "ZonaFiltrosTest",
		"tipo_marcador":     "punto-exacto",
		"latitud":           -17.7833,
		"longitud":          -63.1821,
		"mostrar_direccion": true,
		"dormitorios":       "3",
		"banos":             "2",
		"garaje":            "1",
		"metros_edificados": 150,
		"metros_terreno":    200,
		"estado":            "Excelente",
		"ano_construccion":  2018,
		"descripcion":       "Hermosa casa familiar con jardín y parrillero",
		"terminos_condiciones": "Términos de prueba",
	})
	return body
}

// TestCreateProperty_UsuarioDentroDelLimite - Plan con 2 publicaciones, 0 usadas → 201
// y el contador de la suscripción se incrementa
func (s *PropertyAPISuite) TestCreateProperty_UsuarioDentroDelLimite() {
	resp, result := doJSONRequest("POST", "/api/properties", s.ownerToken, validPropertyBody("Casa de prueba dentro del limite"))

	s.Equal(201, resp.StatusCode)
	s.Equal("Propiedad creada exitosamente", result["message"])

	property := result["property"].(map[string]interface{})
	s.Equal("Casa de prueba dentro del limite", property["titulo"])
	s.Equal("activo", property["estado_propiedad"], "la propiedad se publica activa automáticamente")

	// El contador de la suscripción debe pasar de 0 a 1
	var sub models.Subscription
	s.Require().NoError(testDB.DB.Where("user_id = ?", s.owner.ID).First(&sub).Error)
	s.Equal(1, sub.PropiedadesUsadas, "la publicación consumió 1 cupo del plan")
}

// TestCreateProperty_UsuarioQueSuperoElLimite - Plan con 2 publicaciones, 2 usadas → 403
// con error_code LIMIT_REACHED y sin crear la propiedad
func (s *PropertyAPISuite) TestCreateProperty_UsuarioQueSuperoElLimite() {
	// Agotar el cupo del plan gratuito (2 de 2)
	s.Require().NoError(testDB.DB.Model(&models.Subscription{}).
		Where("user_id = ?", s.owner.ID).
		Update("propiedades_usadas", 2).Error)

	resp, result := doJSONRequest("POST", "/api/properties", s.ownerToken, validPropertyBody("Casa que no debe crearse"))

	s.Equal(403, resp.StatusCode, "superar el límite del plan debe retornar 403 Forbidden")
	s.Equal("LIMIT_REACHED", result["error_code"])
	s.Contains(result["error"], "límite")
	s.Equal(2.0, result["limit"], "la respuesta informa el límite del plan")
	s.Equal(2.0, result["used"], "la respuesta informa el uso actual")

	// Verificar que NO se creó ninguna propiedad
	var count int64
	testDB.DB.Model(&models.Property{}).Where("user_id = ?", s.owner.ID).Count(&count)
	s.Equal(int64(0), count)
}

// TestCreateProperty_SinAutenticacion - Sin token: 401
func (s *PropertyAPISuite) TestCreateProperty_SinAutenticacion() {
	resp, _ := doJSONRequest("POST", "/api/properties", "", validPropertyBody("Casa sin token"))

	s.Equal(401, resp.StatusCode)
}

// TestUpdateProperty_UsuarioNoPropietario - PATCH /properties/:id/status de otro dueño → 403
// (equivalente de "PUT /propiedades/:id usuario no propietario": la actualización de
// propiedades del dueño se expone como PATCH .../status en cmd/api/main.go)
func (s *PropertyAPISuite) TestUpdateProperty_UsuarioNoPropietario() {
	property := createTestPropertyDirect(s.owner.ID, "Casa del propietario", "venta", "USD", 150000, models.PropertyTypeCasa, "ZonaFiltrosTest", "3", models.PropertyStatusActivo)

	body, _ := json.Marshal(map[string]string{"estado": "vendido"})
	resp, result := doJSONRequest("PATCH", fmt.Sprintf("/api/properties/%d/status", property.ID), s.otherToken, body)

	s.Equal(403, resp.StatusCode, "un usuario que no es el dueño recibe 403 Forbidden")
	s.Contains(result["error"], "permiso")

	// El estado de la propiedad no cambió
	var current models.Property
	s.Require().NoError(testDB.DB.First(&current, property.ID).Error)
	s.Equal(models.PropertyStatusActivo, current.EstadoPropiedad)
}

// TestDeleteProperty_EliminacionLogica - DELETE hace soft delete: desaparece de la API
// pero el registro persiste en BD con deleted_at (no es borrado físico)
func (s *PropertyAPISuite) TestDeleteProperty_EliminacionLogica() {
	property := createTestPropertyDirect(s.owner.ID, "Casa para eliminar logicamente", "venta", "USD", 90000, models.PropertyTypeCasa, "ZonaFiltrosTest", "2", models.PropertyStatusActivo)

	resp, result := doJSONRequest("DELETE", fmt.Sprintf("/api/properties/%d", property.ID), s.ownerToken, nil)

	s.Equal(200, resp.StatusCode)
	s.Contains(result["message"], "eliminada")

	// 1) La API ya no la expone (GET → 404)
	respGet, _ := doJSONRequest("GET", fmt.Sprintf("/api/properties/%d", property.ID), "", nil)
	s.Equal(404, respGet.StatusCode)

	// 2) El registro SIGUE en la BD (eliminación lógica, no física) con deleted_at asignado
	var deleted models.Property
	s.Require().NoError(testDB.DB.Unscoped().First(&deleted, property.ID).Error,
		"el registro debe seguir existiendo físicamente en la base de datos")
	s.True(deleted.DeletedAt.Valid, "deleted_at debe estar asignado (soft delete de GORM)")
	s.NotZero(deleted.DeletedAt.Time)
}

// TestDeleteProperty_UsuarioNoPropietario - DELETE de propiedad ajena → 403
func (s *PropertyAPISuite) TestDeleteProperty_UsuarioNoPropietario() {
	property := createTestPropertyDirect(s.owner.ID, "Casa ajena intocable", "venta", "USD", 95000, models.PropertyTypeCasa, "ZonaFiltrosTest", "2", models.PropertyStatusActivo)

	resp, _ := doJSONRequest("DELETE", fmt.Sprintf("/api/properties/%d", property.ID), s.otherToken, nil)

	s.Equal(403, resp.StatusCode)

	var count int64
	testDB.DB.Unscoped().Model(&models.Property{}).Where("id = ?", property.ID).Count(&count)
	s.Equal(int64(1), count, "la propiedad no debe eliminarse")
}

// TestListProperties_FiltrosCombinados - GET /api/properties combinando varios filtros
func (s *PropertyAPISuite) TestListProperties_FiltrosCombinados() {
	// Fixtures: 3 activas + 1 inactiva (el listado público por defecto solo muestra activas)
	propA := createTestPropertyDirect(s.owner.ID, "Casa cara zona test", "venta", "USD", 120000, models.PropertyTypeCasa, "ZonaFiltrosTest", "3", models.PropertyStatusActivo)
	propB := createTestPropertyDirect(s.owner.ID, "Casa economica zona test", "venta", "USD", 80000, models.PropertyTypeCasa, "ZonaFiltrosTest", "2", models.PropertyStatusActivo)
	propC := createTestPropertyDirect(s.owner.ID, "Departamento alquiler otra zona", "alquiler", "BOB", 3500, models.PropertyTypeDepartamento, "OtraZonaTest", "1", models.PropertyStatusActivo)
	createTestPropertyDirect(s.owner.ID, "Casa inactiva zona test", "venta", "USD", 95000, models.PropertyTypeCasa, "ZonaFiltrosTest", "3", models.PropertyStatusInactivo)

	tests := []struct {
		name        string
		query       string
		wantTitles  []string
	}{
		{
			name:       "sin_filtros_muestra_solo_activas",
			query:      "/api/properties",
			wantTitles: []string{propA.Titulo, propB.Titulo, propC.Titulo},
		},
		{
			name:       "tipo_operacion_venta",
			query:      "/api/properties?tipo_operacion=venta",
			wantTitles: []string{propA.Titulo, propB.Titulo},
		},
		{
			name:       "tipo_operacion_y_moneda",
			query:      "/api/properties?tipo_operacion=alquiler&moneda=BOB",
			wantTitles: []string{propC.Titulo},
		},
		{
			name:       "rango_de_precio",
			query:      "/api/properties?tipo_operacion=venta&moneda=USD&precio_min=100000&precio_max=130000",
			wantTitles: []string{propA.Titulo},
		},
		{
			name:       "zona_y_dormitorios",
			query:      "/api/properties?zona=ZonaFiltrosTest&dormitorios=2",
			wantTitles: []string{propB.Titulo},
		},
		{
			name:       "todos_combinados",
			query:      "/api/properties?tipo_operacion=venta&moneda=USD&precio_min=50000&precio_max=100000&zona=ZonaFiltrosTest&dormitorios=2&banos=2",
			wantTitles: []string{propB.Titulo},
		},
		{
			name:       "sin_resultados",
			query:      "/api/properties?tipo_operacion=venta&precio_min=1000000",
			wantTitles: []string{},
		},
	}

	for _, tc := range tests {
		s.T().Run(tc.name, func(t *testing.T) {
			resp, result := doJSONRequest("GET", tc.query, "", nil)

			if !assert.Equal(t, 200, resp.StatusCode) {
				return
			}

			list, _ := result["propiedades"].([]interface{})
			gotTitles := make([]string, 0, len(list))
			for _, item := range list {
				p := item.(map[string]interface{})
				gotTitles = append(gotTitles, p["titulo"].(string))
			}

			assert.ElementsMatch(t, tc.wantTitles, gotTitles)
			assert.Equal(t, float64(len(tc.wantTitles)), result["total"], "el total debe coincidir con los resultados")
		})
	}
}

// TestGetPropertyDetalle - GET /api/properties/:id expone el detalle de una propiedad
func (s *PropertyAPISuite) TestGetPropertyDetalle() {
	property := createTestPropertyDirect(s.owner.ID, "Casa visible en detalle", "venta", "USD", 175000, models.PropertyTypeCasa, "ZonaFiltrosTest", "4", models.PropertyStatusActivo)

	resp, result := doJSONRequest("GET", fmt.Sprintf("/api/properties/%d", property.ID), "", nil)

	s.Equal(200, resp.StatusCode)
	s.Equal(property.Titulo, result["titulo"])
	s.Equal(175000.0, result["precio"])
	s.Equal("USD", result["moneda"])
}

func TestPropertyAPISuite(t *testing.T) {
	suite.Run(t, new(PropertyAPISuite))
}
