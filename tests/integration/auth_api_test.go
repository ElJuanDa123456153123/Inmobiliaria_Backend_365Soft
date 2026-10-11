//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/password"
	"github.com/stretchr/testify/suite"
)

// AuthAPISuite - Pruebas de integración de POST /api/auth/register
type AuthAPISuite struct {
	suite.Suite
}

func (s *AuthAPISuite) SetupTest() {
	cleanupTables()
}

// TestRegister_Exitoso - Registro válido: 201 + token + usuario en BD con password hasheado (bcrypt)
func (s *AuthAPISuite) TestRegister_Exitoso() {
	email := "nuevo.usuario@integration.test"
	body, _ := json.Marshal(map[string]string{
		"nombre":   "Juan",
		"apellido": "Perez",
		"email":    email,
		"password": "ClaveSegura123*",
		"telefono": "77712345",
	})

	resp, result := doJSONRequest("POST", "/api/auth/register", "", body)

	s.Equal(201, resp.StatusCode, "el registro exitoso debe retornar 201 Created")
	s.NotEmpty(result["token"], "la respuesta debe incluir el token JWT")

	user := result["user"].(map[string]interface{})
	s.Equal(email, user["email"])
	s.Equal("usuario", user["rol"], "sin rol explícito se asigna el rol por defecto")
	s.NotContains(fmt.Sprint(user), "ClaveSegura123*", "la respuesta nunca debe exponer la contraseña")

	// Verificar en BD: el password se guardó hasheado con bcrypt y verificable
	var dbUser models.User
	s.Require().NoError(testDB.DB.Where("email = ?", email).First(&dbUser).Error)
	s.NotEqual("ClaveSegura123*", dbUser.Password, "en BD no se guarda texto plano")
	s.True(password.Verify(dbUser.Password, "ClaveSegura123*") == nil, "el hash almacenado debe verificar la contraseña")
}

// TestRegister_EmailDuplicado - Email ya registrado: 409 Conflict
func (s *AuthAPISuite) TestRegister_EmailDuplicado() {
	createTestUser("duplicado@integration.test")

	body, _ := json.Marshal(map[string]string{
		"nombre":   "Otro",
		"apellido": "Usuario",
		"email":    "duplicado@integration.test",
		"password": "OtraClave456*",
	})

	resp, result := doJSONRequest("POST", "/api/auth/register", "", body)

	s.Equal(409, resp.StatusCode)
	s.Contains(result["error"], "ya está registrado")
}

// TestRegister_CamposFaltantes - Falta password: 400 Bad Request
func (s *AuthAPISuite) TestRegister_CamposFaltantes() {
	body, _ := json.Marshal(map[string]string{
		"nombre": "Falta",
		"apellido": "Password",
		"email":   "sin.password@integration.test",
	})

	resp, result := doJSONRequest("POST", "/api/auth/register", "", body)

	s.Equal(400, resp.StatusCode)
	s.Contains(result["error"], "requeridos")
}

// TestRegister_RolInvalido - Rol no soportado: 400 Bad Request
func (s *AuthAPISuite) TestRegister_RolInvalido() {
	body, _ := json.Marshal(map[string]string{
		"nombre":   "Rol",
		"apellido": "Malo",
		"email":    "rol.malo@integration.test",
		"password": "Clave123*",
		"rol":      "superadmin",
	})

	resp, _ := doJSONRequest("POST", "/api/auth/register", "", body)

	s.Equal(400, resp.StatusCode)
}

func TestAuthAPISuite(t *testing.T) {
	suite.Run(t, new(AuthAPISuite))
}
