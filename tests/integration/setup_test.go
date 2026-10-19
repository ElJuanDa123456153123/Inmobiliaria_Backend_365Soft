//go:build integration

// Suite de pruebas de integración de la API.
// Se ejecuta únicamente con el build tag "integration":
//
//	go test -tags=integration ./tests/integration/ -v
//
// Usa una base de datos PostgreSQL de prueba dedicada (por defecto
// "inmobiliaria_test" en el mismo servidor del docker-compose) que se crea
// automáticamente si no existe. El esquema se migra con las mismas migraciones
// de la aplicación (internal/database.AutoMigrate) y las tablas se limpian
// antes de cada test.
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/database"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/handlers"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/middleware"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/password"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

const testJWTSecret = "jwt_secret_pruebas_de_integracion"

var (
	testApp *fiber.App
	testDB  *config.Database
)

// TestMain - Configuración global: entorno aislado, BD de prueba, esquema y app real.
func TestMain(m *testing.M) {
	// 1. Entorno aislado para las pruebas
	os.Setenv("JWT_SECRET", testJWTSecret)
	// Redis inaccesible a propósito: los handlers se inicializan sin caché
	os.Setenv("REDIS_URL", "redis://localhost:6390")

	// 2. Garantizar que exista la base de datos de prueba
	if err := ensureTestDatabase(); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: no se pudo preparar la base de datos de prueba: %v\n", err)
		os.Exit(1)
	}

	// 3. Apuntar la app a la BD de prueba y migrar el esquema (mismas migraciones que producción)
	os.Setenv("DATABASE_URL", testDatabaseURL())
	if err := database.AutoMigrate(); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: error migrando el esquema de prueba: %v\n", err)
		os.Exit(1)
	}

	// 4. Construir la app de Fiber con los handlers y middlewares REALES del proyecto
	if err := buildTestApp(); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: error construyendo la app de prueba: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// buildTestApp - Registra las rutas bajo prueba replicando cmd/api/main.go
func buildTestApp() error {
	db, err := config.GetDatabase()
	if err != nil {
		return err
	}
	testDB = db

	authHandler, err := handlers.NewAuthHandler()
	if err != nil {
		return err
	}
	propertyHandler, err := handlers.NewPropertyHandler()
	if err != nil {
		return err
	}
	favoriteHandler, err := handlers.NewFavoriteHandler()
	if err != nil {
		return err
	}
	visitHandler, err := handlers.NewVisitHandler()
	if err != nil {
		return err
	}
	paymentHandler, err := handlers.NewPaymentHandler()
	if err != nil {
		return err
	}
	contractHandler, err := handlers.NewContractHandler()
	if err != nil {
		return err
	}
	rentPaymentHandler, err := handlers.NewRentPaymentHandler()
	if err != nil {
		return err
	}
	rentalRequestHandler, err := handlers.NewRentalRequestHandler()
	if err != nil {
		return err
	}

	testApp = fiber.New(fiber.Config{AppName: "API Inmobiliaria - Tests de Integración"})

	// Rutas públicas de autenticación
	testApp.Post("/api/auth/register", authHandler.Register)

	api := testApp.Group("/api")

	// Rutas de propiedades (mismo orden y middlewares que cmd/api/main.go)
	api.Get("/properties/my", middleware.AuthMiddleware(), propertyHandler.GetUserProperties)
	api.Patch("/properties/:id/status", middleware.AuthMiddleware(), propertyHandler.UpdatePropertyStatus)
	api.Get("/properties", propertyHandler.ListProperties)
	api.Get("/properties/:id", propertyHandler.GetProperty)
	api.Post("/properties", middleware.AuthMiddleware(), middleware.SubscriptionMiddleware(), propertyHandler.CreateProperty)
	api.Delete("/properties/:id", middleware.AuthMiddleware(), propertyHandler.DeleteProperty)

	// Módulo comercial (segunda iteración): webhook del banco (público) y cancelación de QR
	api.Post("/payments/webhook", paymentHandler.WebhookPayment)
	api.Delete("/payments/:id", middleware.AuthMiddleware(), paymentHandler.CancelPayment)

	// Favoritos
	api.Post("/favorites", middleware.AuthMiddleware(), favoriteHandler.AddFavorite)
	api.Get("/favorites", middleware.AuthMiddleware(), favoriteHandler.GetFavorites)
	api.Delete("/favorites/:id", middleware.AuthMiddleware(), favoriteHandler.RemoveFavorite)
	api.Get("/favorites/check/:propertyId", middleware.AuthMiddleware(), favoriteHandler.CheckFavorite)

	// Visitas (panel)
	panel := testApp.Group("/api/panel", middleware.AuthMiddleware())
	panel.Post("/visits", visitHandler.ScheduleVisit)
	panel.Patch("/visits/:id", visitHandler.UpdateVisitStatus)
	panel.Patch("/visits/:id/confirm", visitHandler.ConfirmVisit)
	panel.Patch("/visits/:id/reprogram", visitHandler.ReprogramarVisit)

	// Módulo de alquiler/anticrético: contratos, cuotas y solicitudes
	contracts := testApp.Group("/api/contracts", middleware.AuthMiddleware())
	contracts.Post("/", contractHandler.CreateContract)
	contracts.Get("/my", contractHandler.GetMyContracts)
	contracts.Get("/:id", contractHandler.GetContract)
	contracts.Patch("/:id", contractHandler.UpdateContract)
	contracts.Get("/:id/pdf", contractHandler.GeneratePDF)
	contracts.Post("/:id/terminate", contractHandler.TerminateContract)
	contracts.Post("/:id/payments", rentPaymentHandler.RegisterPayment)
	contracts.Get("/:id/payments", rentPaymentHandler.GetContractPayments)

	rentPayments := testApp.Group("/api/rent-payments", middleware.AuthMiddleware())
	rentPayments.Patch("/:id", rentPaymentHandler.UpdatePaymentStatus)
	rentPayments.Get("/history", rentPaymentHandler.GetPaymentHistory)
	rentPayments.Post("/:id/comprobante", rentPaymentHandler.SubmitComprobante)
	rentPayments.Post("/:id/confirm-receipt", rentPaymentHandler.ConfirmReceipt)

	api.Post("/properties/:id/rental-request", middleware.AuthMiddleware(), rentalRequestHandler.CreateRentalRequest)
	rentalRequests := testApp.Group("/api/rental-requests", middleware.AuthMiddleware())
	rentalRequests.Get("/my", rentalRequestHandler.GetMyRentalRequests)
	rentalRequests.Patch("/:id/accept", rentalRequestHandler.AcceptRentalRequest)
	rentalRequests.Patch("/:id/reject", rentalRequestHandler.RejectRentalRequest)

	panel.Get("/rental-requests", rentalRequestHandler.GetRentalRequests)
	panel.Get("/tenants", rentPaymentHandler.GetPanelTenants)

	return nil
}

// ========== Base de datos de prueba ==========

func testDBName() string {
	if v := os.Getenv("TEST_DB_NAME"); v != "" {
		return v
	}
	return "inmobiliaria_test"
}

// testBaseURL - Conexión a la BD administrativa del servidor PostgreSQL (docker-compose.services.yml)
func testBaseURL() string {
	if v := os.Getenv("TEST_DATABASE_BASE_URL"); v != "" {
		return v
	}
	return "postgres://inmobiliaria_user:password@localhost:5434/postgres?sslmode=disable"
}

func testDatabaseURL() string {
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		return v
	}
	return fmt.Sprintf("postgres://inmobiliaria_user:password@localhost:5434/%s?sslmode=disable", testDBName())
}

// ensureTestDatabase - Crea la base de datos de prueba si no existe
func ensureTestDatabase() error {
	if os.Getenv("TEST_DATABASE_URL") != "" {
		return nil // BD externa provista por el desarrollador
	}

	db, err := config.NewDatabaseFromDSN(testBaseURL())
	if err != nil {
		return fmt.Errorf("no se pudo conectar al servidor PostgreSQL (¿está corriendo docker-compose?): %w", err)
	}
	defer db.Close()

	var count int64
	if err := db.DB.Raw("SELECT COUNT(*) FROM pg_database WHERE datname = ?", testDBName()).Scan(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		if err := db.DB.Exec(fmt.Sprintf(`CREATE DATABASE %q`, testDBName())).Error; err != nil {
			return err
		}
		fmt.Printf("✔ Base de datos de prueba '%s' creada\n", testDBName())
	}
	return nil
}

// cleanupTables - Limpieza total (hard delete) de las tablas involucradas, en orden FK-seguro
func cleanupTables() {
	modelsToClean := []interface{}{
		&models.RentalRequest{},
		&models.RentPayment{},
		&models.RentalContract{},
		&models.Favorite{},
		&models.Visit{},
		&models.QrPago{},
		&models.Notification{},
		&models.AuditLog{},
		&models.PropertyAdvisor{},
		&models.PropertyImage{},
		&models.Property{},
		&models.Subscription{},
		&models.Plan{},
		&models.User{},
	}
	for _, model := range modelsToClean {
		if err := testDB.DB.Unscoped().
			Session(&gorm.Session{AllowGlobalUpdate: true}).
			Delete(model).Error; err != nil {
			panic(fmt.Sprintf("error limpiando tablas de prueba: %v", err))
		}
	}
}

// ========== Helpers de fixtures y requests ==========

// generateTestToken - Genera un JWT firmado igual que el endpoint /auth/login
// (mismo secret, mismos claims que espera middleware.AuthMiddleware)
func generateTestToken(userID uint, email, rol string) string {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"rol":     rol,
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(2 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		panic(err)
	}
	return signed
}

func createTestUser(email string) *models.User {
	hash, err := password.Hash("ClaveTest123*")
	if err != nil {
		panic(err)
	}
	user := &models.User{
		Nombre:   "Usuario",
		Apellido: "De Prueba",
		Email:    email,
		Password: hash,
		Telefono: "70000000",
		Rol:      models.RoleUsuario,
		Activo:   true,
	}
	if err := testDB.DB.Create(user).Error; err != nil {
		panic(err)
	}
	return user
}

func createTestPlan(nombre string, limitePropiedades int) *models.Plan {
	plan := &models.Plan{
		Nombre:                   nombre,
		Precio:                   0,
		DuracionMeses:            1,
		LimitePropiedades:        limitePropiedades,
		DuracionPropiedadMinutos: 43200,
		MetodoPago:               "QR",
		Activo:                   true,
	}
	if err := testDB.DB.Create(plan).Error; err != nil {
		panic(err)
	}
	return plan
}

func createTestSubscription(userID, planID uint, propiedadesUsadas int) *models.Subscription {
	inicio := time.Now().Add(-24 * time.Hour)
	fin := time.Now().AddDate(1, 0, 0)
	sub := &models.Subscription{
		UserID:              userID,
		PlanID:              planID,
		FechaInicio:         inicio,
		FechaFin:            &fin,
		PropiedadesUsadas:   propiedadesUsadas,
		Estado:              "ACTIVA",
		RenovacionAutomatica: false,
		Seleccionada:        true,
	}
	if err := testDB.DB.Create(sub).Error; err != nil {
		panic(err)
	}
	return sub
}

// createTestPropertyDirect - Crea una propiedad directamente en la BD (sin pasar por la API)
func createTestPropertyDirect(userID uint, titulo, tipoOperacion, moneda string, precio float64, tipoPropiedad models.PropertyType, zona, dormitorios string, estado models.PropertyStatus) *models.Property {
	now := time.Now()
	property := &models.Property{
		UserID:            userID,
		Titulo:            titulo,
		TipoOperacion:     models.OperationType(tipoOperacion),
		Precio:            precio,
		Moneda:            moneda,
		TipoPropiedad:     tipoPropiedad,
		Direccion:         "Calle de prueba 123",
		Zona:              zona,
		TipoMarcador:      "punto-exacto",
		Latitud:           func() *float64 { v := -17.7833; return &v }(),
		Longitud:          func() *float64 { v := -63.1821; return &v }(),
		Dormitorios:       dormitorios,
		Banos:             "2",
		Garaje:            "1",
		MetrosEdificados:  120,
		MetrosTerreno:     150,
		Estado:            "Excelente",
		AnoConstruccion:   2015,
		Descripcion:       "Descripción de prueba para la propiedad de integración",
		EstadoPropiedad:   estado,
		VistasContador:    0,
		FechaPublicacion:  &now,
	}
	if err := testDB.DB.Create(property).Error; err != nil {
		panic(err)
	}
	return property
}

// doJSONRequest - Ejecuta un request HTTP contra la app real usando httptest y
// decodifica la respuesta JSON a un mapa. Retorna la respuesta HTTP cruda y el cuerpo.
func doJSONRequest(method, url, token string, body []byte) (*http.Response, map[string]interface{}) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, url, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := testApp.Test(req, 10000)
	if err != nil {
		panic(err)
	}

	result := map[string]interface{}{}
	if resp.Body != nil {
		raw, readErr := io.ReadAll(resp.Body)
		if readErr == nil && len(raw) > 0 {
			_ = json.Unmarshal(raw, &result)
		}
	}
	return resp, result
}
