//go:build integration

// Pruebas de integración del módulo comercial (segunda iteración):
// webhook del banco, ciclo de favoritos y ciclo de visitas.
// Se ejecutan junto al resto de la suite:
//
//	go test -tags=integration ./tests/integration/ -v
package integration

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/stretchr/testify/assert"
)

// createTestQrPago - Crea un QR de pago directamente en la BD (sin pasar por el banco)
func createTestQrPago(userID, planID uint, alias, estado string) *models.QrPago {
	qr := &models.QrPago{
		Alias:            alias,
		Estado:           estado,
		Monto:            50,
		Moneda:           "BOB",
		PropertyID:       nil, // QR de compra de plan, sin propiedad asociada (NULL)
		UserID:           userID,
		PlanID:           planID,
		FechaVencimiento: time.Now().Add(15 * time.Minute),
	}
	if err := testDB.DB.Create(qr).Error; err != nil {
		panic(err)
	}
	return qr
}

func countSubscriptions(userID uint) int64 {
	var n int64
	testDB.DB.Model(&models.Subscription{}).Where("user_id = ?", userID).Count(&n)
	return n
}

// ========== WEBHOOK DEL BANCO ==========

// I14 - Webhook con alias válido: marca PAGADO, crea suscripción y notifica
func TestI14_WebhookValidoActivaSuscripcion(t *testing.T) {
	cleanupTables()
	user := createTestUser("comprador@test.com")
	plan := createTestPlan("Plan Test 1 Mes", 5)
	qr := createTestQrPago(user.ID, plan.ID, "TESTALIAS-I14", "PENDIENTE")

	body, _ := json.Marshal(map[string]string{
		"alias":                  "TESTALIAS-I14",
		"numeroOrdenOriginante":  "ORD-1001",
		"nombreCliente":          "Juan Perez",
		"documentoCliente":       "1234567 SC",
		"cuentaCliente":          "100020003000",
		"fechaproceso":           "2026-08-30T10:00:00",
	})
	resp, result := doJSONRequest("POST", "/api/payments/webhook", "", body)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "0000", result["codigo"])

	// BD: QR queda PAGADO con datos del pagador
	var qrDB models.QrPago
	testDB.DB.First(&qrDB, qr.ID)
	assert.Equal(t, "PAGADO", qrDB.Estado)
	assert.Equal(t, "ORD-1001", qrDB.NumeroOrden)
	assert.Equal(t, "Juan Perez", qrDB.NombreCliente)
	assert.NotNil(t, qrDB.FechaProceso)

	// BD: una única suscripción ACTIVA y seleccionada
	assert.Equal(t, int64(1), countSubscriptions(user.ID))
	var sub models.Subscription
	testDB.DB.Where("user_id = ?", user.ID).First(&sub)
	assert.Equal(t, "ACTIVA", sub.Estado)
	assert.True(t, sub.Seleccionada)
	assert.Equal(t, 0, sub.PropiedadesUsadas)

	// BD: fecha_fin = inicio + duracion_meses (plan de 1 mes)
	finEsperado := time.Now().AddDate(0, 1, 0)
	assert.WithinDuration(t, finEsperado, *sub.FechaFin, 5*time.Minute)

	// BD: notificación de pago confirmado
	var notif models.Notification
	err := testDB.DB.Where("user_id = ? AND tipo = ?", user.ID, "pago_confirmado").First(&notif).Error
	assert.NoError(t, err)
}

// I15 - Webhook idempotente: reenviar el mismo webhook no duplica suscripción
func TestI15_WebhookIdempotente(t *testing.T) {
	cleanupTables()
	user := createTestUser("comprador2@test.com")
	plan := createTestPlan("Plan Test Idempotente", 5)
	createTestQrPago(user.ID, plan.ID, "TESTALIAS-I15", "PENDIENTE")

	body, _ := json.Marshal(map[string]string{"alias": "TESTALIAS-I15"})

	resp, result := doJSONRequest("POST", "/api/payments/webhook", "", body)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "0000", result["codigo"])

	resp, result = doJSONRequest("POST", "/api/payments/webhook", "", body)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "0000", result["codigo"])

	assert.Equal(t, int64(1), countSubscriptions(user.ID), "no debe duplicar la suscripción")
}

// I16 - Webhook con alias inexistente: código 1212 del banco, sin efectos
func TestI16_WebhookAliasInexistente(t *testing.T) {
	cleanupTables()
	body, _ := json.Marshal(map[string]string{"alias": "ALIAS-INEXISTENTE-999"})
	resp, result := doJSONRequest("POST", "/api/payments/webhook", "", body)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "1212", result["codigo"])
	assert.Equal(t, "Alias no encontrado", result["mensaje"])
}

// I17 - Cancelación de un QR ya PAGADO es rechazada
func TestI17_CancelarQrYaPagado(t *testing.T) {
	cleanupTables()
	user := createTestUser("comprador3@test.com")
	plan := createTestPlan("Plan Test Cancelacion", 5)
	qr := createTestQrPago(user.ID, plan.ID, "TESTALIAS-I17", "PAGADO")

	token := generateTestToken(user.ID, "comprador3@test.com", "usuario")
	resp, _ := doJSONRequest("DELETE", fmt.Sprintf("/api/payments/%d", qr.ID), token, nil)
	assert.Equal(t, 500, resp.StatusCode)

	var qrDB models.QrPago
	testDB.DB.First(&qrDB, qr.ID)
	assert.Equal(t, "PAGADO", qrDB.Estado, "el estado no debe cambiar")
}

// ========== FAVORITOS ==========

// I18 - Agregar favorito y verificar con check
func TestI18_AgregarFavoritoYCheck(t *testing.T) {
	cleanupTables()
	duenho := createTestUser("vendedor-fav@test.com")
	cliente := createTestUser("cliente-fav@test.com")
	prop := createTestPropertyDirect(duenho.ID, "Casa favorita", "venta", "USD", 100000, models.PropertyTypeCasa, "Zona Norte", "3", models.PropertyStatusActivo)
	token := generateTestToken(cliente.ID, "cliente-fav@test.com", "usuario")

	body, _ := json.Marshal(map[string]uint{"property_id": prop.ID})
	resp, result := doJSONRequest("POST", "/api/favorites", token, body)
	assert.Equal(t, 201, resp.StatusCode)
	assert.Equal(t, "Added to favorites", result["message"])

	resp, result = doJSONRequest("GET", fmt.Sprintf("/api/favorites/check/%d", prop.ID), token, nil)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, true, result["isFavorite"])
}

// I19 - Favorito duplicado rechazado con 400
func TestI19_FavoritoDuplicadoRechazado(t *testing.T) {
	cleanupTables()
	duenho := createTestUser("vendedor-fav2@test.com")
	cliente := createTestUser("cliente-fav2@test.com")
	prop := createTestPropertyDirect(duenho.ID, "Depto favorito", "alquiler", "BOB", 2500, models.PropertyTypeDepartamento, "Zona Sur", "2", models.PropertyStatusActivo)
	token := generateTestToken(cliente.ID, "cliente-fav2@test.com", "usuario")

	body, _ := json.Marshal(map[string]uint{"property_id": prop.ID})
	resp, _ := doJSONRequest("POST", "/api/favorites", token, body)
	assert.Equal(t, 201, resp.StatusCode)

	resp, result := doJSONRequest("POST", "/api/favorites", token, body)
	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "Already in favorites", result["error"])
}

// I20 - Eliminar y volver a marcar: sin duplicados tras la eliminación
func TestI20_FavoritoRestauradoTrasEliminar(t *testing.T) {
	cleanupTables()
	duenho := createTestUser("vendedor-fav3@test.com")
	cliente := createTestUser("cliente-fav3@test.com")
	prop := createTestPropertyDirect(duenho.ID, "Oficina favorita", "venta", "USD", 80000, models.PropertyTypeOficina, "Centro", "1", models.PropertyStatusActivo)
	token := generateTestToken(cliente.ID, "cliente-fav3@test.com", "usuario")

	body, _ := json.Marshal(map[string]uint{"property_id": prop.ID})
	resp, _ := doJSONRequest("POST", "/api/favorites", token, body)
	assert.Equal(t, 201, resp.StatusCode)

	// Eliminar el favorito
	resp, _ = doJSONRequest("DELETE", fmt.Sprintf("/api/favorites/%d", prop.ID), token, nil)
	assert.Equal(t, 200, resp.StatusCode)

	// Re-marcar: 201 de nuevo y sin duplicados (una sola fila activa para el usuario)
	resp, result := doJSONRequest("POST", "/api/favorites", token, body)
	assert.Equal(t, 201, resp.StatusCode)
	assert.Equal(t, "Added to favorites", result["message"])

	var total int64
	testDB.DB.Model(&models.Favorite{}).Where("user_id = ?", cliente.ID).Count(&total)
	assert.Equal(t, int64(1), total, "no deben existir duplicados tras re-marcar")
}

// I21 - Cada usuario ve únicamente sus propios favoritos
func TestI21_FavoritosSoloPropios(t *testing.T) {
	cleanupTables()
	duenho := createTestUser("vendedor-fav4@test.com")
	clienteA := createTestUser("clientea@test.com")
	clienteB := createTestUser("clienteb@test.com")
	prop1 := createTestPropertyDirect(duenho.ID, "Casa A", "venta", "USD", 100000, models.PropertyTypeCasa, "Zona A", "3", models.PropertyStatusActivo)
	prop2 := createTestPropertyDirect(duenho.ID, "Casa B", "venta", "USD", 120000, models.PropertyTypeCasa, "Zona B", "4", models.PropertyStatusActivo)

	tokenA := generateTestToken(clienteA.ID, "clientea@test.com", "usuario")
	tokenB := generateTestToken(clienteB.ID, "clienteb@test.com", "usuario")

	body1, _ := json.Marshal(map[string]uint{"property_id": prop1.ID})
	resp, _ := doJSONRequest("POST", "/api/favorites", tokenA, body1)
	assert.Equal(t, 201, resp.StatusCode)

	body2, _ := json.Marshal(map[string]uint{"property_id": prop2.ID})
	resp, _ = doJSONRequest("POST", "/api/favorites", tokenB, body2)
	assert.Equal(t, 201, resp.StatusCode)

	resp, result := doJSONRequest("GET", "/api/favorites", tokenA, nil)
	assert.Equal(t, 200, resp.StatusCode)
	favs := result["favorites"].([]interface{})
	assert.Len(t, favs, 1, "A solo ve su favorito")
	favA := favs[0].(map[string]interface{})
	assert.Equal(t, float64(prop1.ID), favA["property_id"])

	resp, result = doJSONRequest("GET", "/api/favorites", tokenB, nil)
	favs = result["favorites"].([]interface{})
	assert.Len(t, favs, 1, "B solo ve su favorito")
	favB := favs[0].(map[string]interface{})
	assert.Equal(t, float64(prop2.ID), favB["property_id"])
}

// ========== VISITAS ==========

// I22 - Visita a propiedad propia bloqueada
func TestI22_VisitaAPropiedadPropiaRechazada(t *testing.T) {
	cleanupTables()
	duenho := createTestUser("vendedor-vis@test.com")
	prop := createTestPropertyDirect(duenho.ID, "Casa propia", "venta", "USD", 90000, models.PropertyTypeCasa, "Zona C", "2", models.PropertyStatusActivo)
	token := generateTestToken(duenho.ID, "vendedor-vis@test.com", "usuario")

	body, _ := json.Marshal(map[string]interface{}{
		"property_id":       prop.ID,
		"nombre_completo":   "Duenho Test",
		"telefono":          "70000001",
		"horario_preferido": "Lunes por la mañana",
	})
	resp, result := doJSONRequest("POST", "/api/panel/visits", token, body)
	assert.Equal(t, 400, resp.StatusCode)
	assert.Contains(t, result["error"], "propia propiedad")
}

// I23 - Agendar visita válida: pendiente + notificación al vendedor
func TestI23_AgendarVisitaCreaNotificacionAlVendedor(t *testing.T) {
	cleanupTables()
	duenho := createTestUser("vendedor-vis2@test.com")
	cliente := createTestUser("cliente-vis@test.com")
	prop := createTestPropertyDirect(duenho.ID, "Casa visitable", "venta", "USD", 95000, models.PropertyTypeCasa, "Zona D", "3", models.PropertyStatusActivo)
	token := generateTestToken(cliente.ID, "cliente-vis@test.com", "usuario")

	body, _ := json.Marshal(map[string]interface{}{
		"property_id":       prop.ID,
		"nombre_completo":   "Cliente Interesado",
		"telefono":          "70000002",
		"horario_preferido": "Sábado 10:00",
		"notas":             "Soy comprador serio",
	})
	resp, result := doJSONRequest("POST", "/api/panel/visits", token, body)
	assert.Equal(t, 201, resp.StatusCode)
	assert.Equal(t, "Solicitud de visita enviada correctamente", result["message"])

	// BD: visita pendiente vinculada a inquilino y vendedor
	var visita models.Visit
	testDB.DB.Where("inquilino_id = ?", cliente.ID).First(&visita)
	assert.Equal(t, prop.ID, visita.PropertyID)
	assert.Equal(t, duenho.ID, visita.VendedorID)
	assert.Equal(t, models.VisitStatusPendiente, visita.Estado)

	// BD: notificación al vendedor
	var notif models.Notification
	err := testDB.DB.Where("user_id = ? AND tipo = ?", duenho.ID, "visita_solicitada").First(&notif).Error
	assert.NoError(t, err)
}

// I24 - Confirmar visita: solo el vendedor, solo si está pendiente
func TestI24_ConfirmarVisitaPorVendedor(t *testing.T) {
	cleanupTables()
	duenho := createTestUser("vendedor-vis3@test.com")
	cliente := createTestUser("cliente-vis2@test.com")
	prop := createTestPropertyDirect(duenho.ID, "Casa confirmable", "venta", "USD", 97000, models.PropertyTypeCasa, "Zona E", "3", models.PropertyStatusActivo)

	visita := &models.Visit{
		PropertyID: prop.ID,
		InquilinoID: cliente.ID,
		VendedorID:  duenho.ID,
		FechaVisita: time.Now(),
		Estado:      models.VisitStatusPendiente,
	}
	if err := testDB.DB.Create(visita).Error; err != nil {
		panic(err)
	}

	tokenVend := generateTestToken(duenho.ID, "vendedor-vis3@test.com", "usuario")
	body, _ := json.Marshal(map[string]string{"fecha_visita": "2026-09-15 15:00"})
	resp, result := doJSONRequest("PATCH", fmt.Sprintf("/api/panel/visits/%d/confirm", visita.ID), tokenVend, body)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "Visita confirmada correctamente", result["message"])

	// BD: estado confirmada con fecha asignada
	var visitaDB models.Visit
	testDB.DB.First(&visitaDB, visita.ID)
	assert.Equal(t, models.VisitStatusConfirmada, visitaDB.Estado)
	fechaEsperada, _ := time.Parse("2006-01-02 15:04", "2026-09-15 15:00")
	assert.WithinDuration(t, fechaEsperada, visitaDB.FechaVisita, time.Second)

	// Confirmar de nuevo debe fallar: solo visitas pendientes
	resp, result = doJSONRequest("PATCH", fmt.Sprintf("/api/panel/visits/%d/confirm", visita.ID), tokenVend, body)
	assert.Equal(t, 400, resp.StatusCode)
	assert.Contains(t, result["error"], "pendientes")
}

// I25 - El inquilino no puede confirmar la visita (solo el vendedor)
func TestI25_InquilinoNoPuedeConfirmar(t *testing.T) {
	cleanupTables()
	duenho := createTestUser("vendedor-vis4@test.com")
	cliente := createTestUser("cliente-vis3@test.com")
	prop := createTestPropertyDirect(duenho.ID, "Casa ajena", "venta", "USD", 98000, models.PropertyTypeCasa, "Zona F", "3", models.PropertyStatusActivo)

	visita := &models.Visit{
		PropertyID: prop.ID,
		InquilinoID: cliente.ID,
		VendedorID:  duenho.ID,
		FechaVisita: time.Now(),
		Estado:      models.VisitStatusPendiente,
	}
	if err := testDB.DB.Create(visita).Error; err != nil {
		panic(err)
	}

	tokenCli := generateTestToken(cliente.ID, "cliente-vis3@test.com", "usuario")
	body, _ := json.Marshal(map[string]string{"fecha_visita": "2026-09-20 10:00"})
	resp, result := doJSONRequest("PATCH", fmt.Sprintf("/api/panel/visits/%d/confirm", visita.ID), tokenCli, body)
	assert.Equal(t, 404, resp.StatusCode)
	assert.Contains(t, result["error"], "permisos")
}

// I26 - Cancelar visita genera notificación al inquilino
func TestI26_CancelarVisitaNotificaAlCliente(t *testing.T) {
	cleanupTables()
	duenho := createTestUser("vendedor-vis5@test.com")
	cliente := createTestUser("cliente-vis4@test.com")
	prop := createTestPropertyDirect(duenho.ID, "Casa cancelable", "venta", "USD", 99000, models.PropertyTypeCasa, "Zona G", "3", models.PropertyStatusActivo)

	visita := &models.Visit{
		PropertyID: prop.ID,
		InquilinoID: cliente.ID,
		VendedorID:  duenho.ID,
		FechaVisita: time.Now(),
		Estado:      models.VisitStatusPendiente,
	}
	if err := testDB.DB.Create(visita).Error; err != nil {
		panic(err)
	}

	tokenVend := generateTestToken(duenho.ID, "vendedor-vis5@test.com", "usuario")
	body, _ := json.Marshal(map[string]string{"estado": "cancelada"})
	resp, result := doJSONRequest("PATCH", fmt.Sprintf("/api/panel/visits/%d", visita.ID), tokenVend, body)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "Estado de visita actualizado", result["message"])

	var visitaDB models.Visit
	testDB.DB.First(&visitaDB, visita.ID)
	assert.Equal(t, models.VisitStatusCancelada, visitaDB.Estado)

	var notif models.Notification
	err := testDB.DB.Where("user_id = ? AND tipo = ?", cliente.ID, "visita_cancelada").First(&notif).Error
	assert.NoError(t, err)
}
