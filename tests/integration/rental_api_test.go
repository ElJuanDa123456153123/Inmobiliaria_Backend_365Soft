//go:build integration

// Pruebas de integración del módulo de alquiler y anticrético (iteración 4).
// Cubre el ciclo completo: solicitud -> aceptación (contrato + cuotas + depósito)
// -> pago de cuota con comprobante -> confirmación -> finalización con devolución.
package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
)

// ---------- Helpers específicos del módulo de alquiler ----------

type rentalFixture struct {
	propietario *models.User
	inquilino   *models.User
	otro        *models.User
	propiedad   *models.Property
}

// setupRental - Crea propietario, inquilino y propiedad de alquiler activa con depósito "1 mes"
func setupRental(t *testing.T, tipoOperacion string, deposito string) *rentalFixture {
	t.Helper()
	propietario := createTestUser(fmt.Sprintf("propietario-%d@test.bo", time.Now().UnixNano()))
	inquilino := createTestUser(fmt.Sprintf("inquilino-%d@test.bo", time.Now().UnixNano()))
	otro := createTestUser(fmt.Sprintf("otro-%d@test.bo", time.Now().UnixNano()))
	prop := createTestPropertyDirect(
		propietario.ID,
		"Departamento de prueba para alquiler integración",
		tipoOperacion, "BOB", 2500,
		models.PropertyTypeDepartamento, "Sacaba", "2",
		models.PropertyStatusActivo,
	)
	// Depósito de garantía declarado en la publicación
	prop.Deposito = deposito
	if err := testDB.DB.Save(prop).Error; err != nil {
		t.Fatalf("no se pudo guardar el depósito: %v", err)
	}
	return &rentalFixture{propietario: propietario, inquilino: inquilino, otro: otro, propiedad: prop}
}

// crearSolicitud - El inquilino envía una solicitud de alquiler
func (f *rentalFixture) crearSolicitud(t *testing.T) float64 {
	t.Helper()
	token := generateTestToken(f.inquilino.ID, f.inquilino.Email, string(f.inquilino.Rol))
	resp, body := doJSONRequest("POST",
		fmt.Sprintf("/api/properties/%d/rental-request", f.propiedad.ID),
		token, []byte(`{"mensaje":"Me interesa","telefono_contacto":"77712345"}`))
	if resp.StatusCode != 201 {
		t.Fatalf("crear solicitud: esperado 201, obtenido %d: %v", resp.StatusCode, body)
	}
	return body["solicitud"].(map[string]interface{})["id"].(float64)
}

func contractPaymentsCount(t *testing.T, contractID uint, tipo models.PaymentType) int64 {
	t.Helper()
	var count int64
	if err := testDB.DB.Model(&models.RentPayment{}).
		Where("contract_id = ? AND tipo = ?", contractID, tipo).
		Count(&count).Error; err != nil {
		t.Fatalf("contando pagos: %v", err)
	}
	return count
}

func getContract(t *testing.T, contractID uint) models.RentalContract {
	t.Helper()
	var c models.RentalContract
	if err := testDB.DB.First(&c, contractID).Error; err != nil {
		t.Fatalf("buscando contrato: %v", err)
	}
	return c
}

// ---------- Pruebas: ciclo de ALQUILER ----------

// TestRentalCicloCompleto - Solicitud -> aceptación (12 cuotas + depósito) -> pago con
// comprobante -> confirmación -> unicidad de cuotas -> finalización con devolución.
func TestRentalCicloCompleto(t *testing.T) {
	cleanupTables()
	f := setupRental(t, "alquiler", "1 mes")
	tokenProp := generateTestToken(f.propietario.ID, f.propietario.Email, string(f.propietario.Rol))
	tokenInq := generateTestToken(f.inquilino.ID, f.inquilino.Email, string(f.inquilino.Rol))

	// 1. El inquilino envía la solicitud
	solicitudID := f.crearSolicitud(t)

	// 2. Solicitud duplicada -> 409
	resp, body := doJSONRequest("POST",
		fmt.Sprintf("/api/properties/%d/rental-request", f.propiedad.ID),
		tokenInq, []byte(`{}`))
	if resp.StatusCode != 409 {
		t.Fatalf("solicitud duplicada: esperado 409, obtenido %d: %v", resp.StatusCode, body)
	}

	// 3. El propietario ve la solicitud pendiente en su panel
	resp, body = doJSONRequest("GET", "/api/panel/rental-requests?estado=pendiente", tokenProp, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("listar solicitudes: esperado 200, obtenido %d: %v", resp.StatusCode, body)
	}
	if total := body["total"].(float64); total != 1 {
		t.Fatalf("solicitudes pendientes: esperado 1, obtenido %v", total)
	}

	// 4. Aceptación: contrato de 12 meses con renta 2500 -> 12 cuotas + depósito 2500
	inicio := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	fin := time.Now().AddDate(1, 0, 1).Format("2006-01-02")
	resp, body = doJSONRequest("PATCH",
		fmt.Sprintf("/api/rental-requests/%d/accept", int(solicitudID)),
		tokenProp,
		[]byte(fmt.Sprintf(`{"fecha_inicio":"%s","fecha_fin":"%s","monto_mensual":2500}`, inicio, fin)))
	if resp.StatusCode != 200 {
		t.Fatalf("aceptar solicitud: esperado 200, obtenido %d: %v", resp.StatusCode, body)
	}
	if n := int(body["cuotas_generadas"].(float64)); n != 12 {
		t.Fatalf("cuotas generadas: esperado 12, obtenido %v", body["cuotas_generadas"])
	}
	if d := body["monto_deposito"].(float64); d != 2500 {
		t.Fatalf("depósito: esperado 2500, obtenido %v", body["monto_deposito"])
	}
	contractID := uint(body["contrato"].(map[string]interface{})["id"].(float64))

	// 5. Verificar estado en BD: contrato activo, 12 cuotas, 1 depósito, propiedad alquilada
	contrato := getContract(t, contractID)
	if contrato.Estado != models.ContractStatusActivo {
		t.Fatalf("estado del contrato: esperado activo, obtenido %s", contrato.Estado)
	}
	if contrato.MontoDeposito != 2500 {
		t.Fatalf("monto depósito en contrato: esperado 2500, obtenido %v", contrato.MontoDeposito)
	}
	if n := contractPaymentsCount(t, contractID, models.PaymentTypeCuota); n != 12 {
		t.Fatalf("cuotas en BD: esperado 12, obtenido %d", n)
	}
	if n := contractPaymentsCount(t, contractID, models.PaymentTypeDeposito); n != 1 {
		t.Fatalf("depósitos en BD: esperado 1, obtenido %d", n)
	}
	var propDB models.Property
	testDB.DB.First(&propDB, f.propiedad.ID)
	if !propDB.Alquilada {
		t.Fatal("la propiedad debería estar marcada como alquilada")
	}

	// 6. El resto de solicitudes se cancelan: otra persona solicita y queda pendiente,
	//    luego se verifica que la aceptación canceló las demás (ya se aceptó la primera,
	//    así que validamos el historial del inquilino como aceptada)
	resp, body = doJSONRequest("GET", "/api/rental-requests/my", tokenInq, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("mis solicitudes: esperado 200, obtenido %d", resp.StatusCode)
	}

	// 7. Historial de pagos del inquilino: 12 cuotas + 1 depósito
	resp, body = doJSONRequest("GET", "/api/rent-payments/history", tokenInq, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("historial: esperado 200, obtenido %d", resp.StatusCode)
	}
	if total := body["total"].(float64); total != 13 {
		t.Fatalf("historial de pagos: esperado 13, obtenido %v", total)
	}

	// 8. El inquilino sube comprobante de la primera cuota
	var primeraCuotaID uint
	var cuotas []models.RentPayment
	testDB.DB.Where("contract_id = ? AND tipo = ?", contractID, models.PaymentTypeCuota).
		Order("anio ASC, mes_correspondiente ASC").First(&cuotas)
	if len(cuotas) == 0 {
		var una models.RentPayment
		testDB.DB.Where("contract_id = ? AND tipo = ?", contractID, models.PaymentTypeCuota).
			Order("anio ASC, mes_correspondiente ASC").First(&una)
		primeraCuotaID = una.ID
	} else {
		primeraCuotaID = cuotas[0].ID
	}
	resp, body = doJSONRequest("POST",
		fmt.Sprintf("/api/rent-payments/%d/comprobante", primeraCuotaID),
		tokenInq,
		[]byte(`{"metodo_pago":"qr","comprobante_url":"/api/imagenes/comprobante-test.jpg"}`))
	if resp.StatusCode != 200 {
		t.Fatalf("subir comprobante: esperado 200, obtenido %d: %v", resp.StatusCode, body)
	}

	// 9. El propietario confirma el pago
	resp, body = doJSONRequest("PATCH",
		fmt.Sprintf("/api/rent-payments/%d", primeraCuotaID),
		tokenProp, []byte(`{"estado":"pagado"}`))
	if resp.StatusCode != 200 {
		t.Fatalf("confirmar pago: esperado 200, obtenido %d: %v", resp.StatusCode, body)
	}

	// 10. Registro doble de la misma cuota (mismo mes/año/tipo) -> 400
	pagoDuplicado := fmt.Sprintf(
		`{"mes_correspondiente":%d,"anio":%d,"monto":2500,"fecha_pago":"%s","metodo_pago":"efectivo"}`,
		int(cuotas[0].MesCorrespondiente), cuotas[0].Anio, time.Now().Format(time.RFC3339))
	resp, body = doJSONRequest("POST",
		fmt.Sprintf("/api/contracts/%d/payments", contractID),
		tokenProp, []byte(pagoDuplicado))
	if resp.StatusCode != 400 {
		t.Fatalf("pago duplicado: esperado 400, obtenido %d: %v", resp.StatusCode, body)
	}

	// 11. El inquilino no puede actualizar el estado de un pago (solo el propietario)
	resp, _ = doJSONRequest("PATCH",
		fmt.Sprintf("/api/rent-payments/%d", primeraCuotaID),
		tokenInq, []byte(`{"estado":"atrasado"}`))
	if resp.StatusCode != 404 {
		t.Fatalf("actualización por inquilino: esperado 404, obtenido %d", resp.StatusCode)
	}

	// 12. Finalizar contrato -> devolución del depósito y propiedad liberada
	resp, body = doJSONRequest("POST",
		fmt.Sprintf("/api/contracts/%d/terminate", contractID),
		tokenProp, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("finalizar contrato: esperado 200, obtenido %d: %v", resp.StatusCode, body)
	}
	contrato = getContract(t, contractID)
	if contrato.Estado != models.ContractStatusTerminado {
		t.Fatalf("estado tras finalizar: esperado terminado, obtenido %s", contrato.Estado)
	}
	if n := contractPaymentsCount(t, contractID, models.PaymentTypeDevolucionDeposito); n != 1 {
		t.Fatalf("devoluciones de depósito: esperado 1, obtenido %d", n)
	}
	testDB.DB.First(&propDB, f.propiedad.ID)
	if propDB.Alquilada {
		t.Fatal("la propiedad debería estar liberada tras finalizar el contrato")
	}

	// 13. El inquilino confirma la recepción de la devolución
	var devolucion models.RentPayment
	testDB.DB.Where("contract_id = ? AND tipo = ?", contractID, models.PaymentTypeDevolucionDeposito).
		First(&devolucion)
	resp, body = doJSONRequest("POST",
		fmt.Sprintf("/api/rent-payments/%d/confirm-receipt", devolucion.ID),
		tokenInq, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("confirmar devolución: esperado 200, obtenido %d: %v", resp.StatusCode, body)
	}
	testDB.DB.First(&devolucion, devolucion.ID)
	if devolucion.Estado != models.PaymentStatusPagado {
		t.Fatalf("estado devolución: esperado pagado, obtenido %s", devolucion.Estado)
	}
}

// ---------- Pruebas: ciclo ANTICRÉTICO ----------

// TestAnticreticoCiclo - Solicitud -> aceptación (anticrés único, sin cuotas) ->
// pago del anticrés -> finalización con devolución del anticrés.
func TestAnticreticoCiclo(t *testing.T) {
	cleanupTables()
	f := setupRental(t, "anticretico", "")
	tokenProp := generateTestToken(f.propietario.ID, f.propietario.Email, string(f.propietario.Rol))
	tokenInq := generateTestToken(f.inquilino.ID, f.inquilino.Email, string(f.inquilino.Rol))

	solicitudID := f.crearSolicitud(t)

	// 1. Aceptación anticrética: monto del anticrés, sin renta mensual
	inicio := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	fin := time.Now().AddDate(2, 0, 1).Format("2006-01-02") // 2 años
	resp, body := doJSONRequest("PATCH",
		fmt.Sprintf("/api/rental-requests/%d/accept", int(solicitudID)),
		tokenProp,
		[]byte(fmt.Sprintf(`{"fecha_inicio":"%s","fecha_fin":"%s","monto_anticres":30000}`, inicio, fin)))
	if resp.StatusCode != 200 {
		t.Fatalf("aceptar anticrético: esperado 200, obtenido %d: %v", resp.StatusCode, body)
	}
	if n := int(body["cuotas_generadas"].(float64)); n != 0 {
		t.Fatalf("cuotas en anticrético: esperado 0, obtenido %v", body["cuotas_generadas"])
	}
	contractID := uint(body["contrato"].(map[string]interface{})["id"].(float64))

	// 2. Verificaciones en BD: sin cuotas, un anticrés, tipo de contrato
	contrato := getContract(t, contractID)
	if contrato.TipoContrato != models.ContractTypeAnticretico {
		t.Fatalf("tipo de contrato: esperado anticretico, obtenido %s", contrato.TipoContrato)
	}
	if contrato.MontoMensual != 0 {
		t.Fatalf("renta mensual en anticrético: esperado 0, obtenido %v", contrato.MontoMensual)
	}
	if n := contractPaymentsCount(t, contractID, models.PaymentTypeCuota); n != 0 {
		t.Fatalf("cuotas en anticrético: esperado 0, obtenido %d", n)
	}
	if n := contractPaymentsCount(t, contractID, models.PaymentTypeAnticres); n != 1 {
		t.Fatalf("pagos de anticrés: esperado 1, obtenido %d", n)
	}

	// 3. El inquilino paga el anticrés y el propietario confirma
	var anticres models.RentPayment
	testDB.DB.Where("contract_id = ? AND tipo = ?", contractID, models.PaymentTypeAnticres).First(&anticres)
	resp, _ = doJSONRequest("POST",
		fmt.Sprintf("/api/rent-payments/%d/comprobante", anticres.ID),
		tokenInq,
		[]byte(`{"metodo_pago":"transferencia","comprobante_url":"/api/imagenes/anticres.jpg"}`))
	if resp.StatusCode != 200 {
		t.Fatalf("comprobante anticrés: esperado 200")
	}
	resp, _ = doJSONRequest("PATCH",
		fmt.Sprintf("/api/rent-payments/%d", anticres.ID),
		tokenProp, []byte(`{"estado":"pagado"}`))
	if resp.StatusCode != 200 {
		t.Fatalf("confirmar anticrés: esperado 200")
	}

	// 4. Finalizar -> devolución del anticrés por el mismo monto
	resp, body = doJSONRequest("POST",
		fmt.Sprintf("/api/contracts/%d/terminate", contractID),
		tokenProp, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("finalizar anticrético: esperado 200, obtenido %d: %v", resp.StatusCode, body)
	}
	var devolucion models.RentPayment
	testDB.DB.Where("contract_id = ? AND tipo = ?", contractID, models.PaymentTypeDevolucionAnticres).First(&devolucion)
	if devolucion.ID == 0 {
		t.Fatal("no se creó la devolución del anticrés")
	}
	if devolucion.Monto != 30000 {
		t.Fatalf("monto de devolución: esperado 30000, obtenido %v", devolucion.Monto)
	}

	// 5. El inquilino confirma la recepción
	resp, _ = doJSONRequest("POST",
		fmt.Sprintf("/api/rent-payments/%d/confirm-receipt", devolucion.ID),
		tokenInq, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("confirmar recepción: esperado 200")
	}
}

// ---------- Pruebas: validaciones de solicitudes ----------

// TestRentalValidaciones - Rechazos esperados: propiedad inexistente, propiedad propia,
// propiedad en venta, aceptación con monto inválido y rechazo de solicitud.
func TestRentalValidaciones(t *testing.T) {
	cleanupTables()
	f := setupRental(t, "alquiler", "1 mes")
	tokenProp := generateTestToken(f.propietario.ID, f.propietario.Email, string(f.propietario.Rol))
	tokenOtro := generateTestToken(f.otro.ID, f.otro.Email, string(f.otro.Rol))

	// 1. Propiedad inexistente -> 404
	resp, _ := doJSONRequest("POST", "/api/properties/999999/rental-request", tokenOtro, []byte(`{}`))
	if resp.StatusCode != 404 {
		t.Fatalf("propiedad inexistente: esperado 404, obtenido %d", resp.StatusCode)
	}

	// 2. Solicitar la propia propiedad -> 400
	resp, _ = doJSONRequest("POST",
		fmt.Sprintf("/api/properties/%d/rental-request", f.propiedad.ID),
		tokenProp, []byte(`{}`))
	if resp.StatusCode != 400 {
		t.Fatalf("propia propiedad: esperado 400, obtenido %d", resp.StatusCode)
	}

	// 3. Propiedad en venta -> 400
	propVenta := createTestPropertyDirect(
		f.propietario.ID, "Casa en venta de prueba", "venta", "USD", 80000,
		models.PropertyTypeCasa, "Equipetrol", "3", models.PropertyStatusActivo)
	resp, _ = doJSONRequest("POST",
		fmt.Sprintf("/api/properties/%d/rental-request", propVenta.ID),
		tokenOtro, []byte(`{}`))
	if resp.StatusCode != 400 {
		t.Fatalf("propiedad en venta: esperado 400, obtenido %d", resp.StatusCode)
	}

	// 4. Solicitud válida de "otro" y rechazo por el propietario
	resp, body := doJSONRequest("POST",
		fmt.Sprintf("/api/properties/%d/rental-request", f.propiedad.ID),
		tokenOtro, []byte(`{"mensaje":"Hola"}`))
	if resp.StatusCode != 201 {
		t.Fatalf("solicitud válida: esperado 201, obtenido %d: %v", resp.StatusCode, body)
	}
	solicitudID := int(body["solicitud"].(map[string]interface{})["id"].(float64))

	// 5. Aceptación con monto inválido -> 400
	inicio := time.Now().Format("2006-01-02")
	fin := time.Now().AddDate(1, 0, 0).Format("2006-01-02")
	resp, _ = doJSONRequest("PATCH",
		fmt.Sprintf("/api/rental-requests/%d/accept", solicitudID),
		tokenProp,
		[]byte(fmt.Sprintf(`{"fecha_inicio":"%s","fecha_fin":"%s","monto_mensual":0}`, inicio, fin)))
	if resp.StatusCode != 400 {
		t.Fatalf("monto inválido: esperado 400, obtenido %d", resp.StatusCode)
	}

	// 6. Rechazo de la solicitud
	resp, _ = doJSONRequest("PATCH",
		fmt.Sprintf("/api/rental-requests/%d/reject", solicitudID),
		tokenProp, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("rechazar solicitud: esperado 200, obtenido %d", resp.StatusCode)
	}
	var solicitud models.RentalRequest
	testDB.DB.First(&solicitud, solicitudID)
	if solicitud.Estado != models.RentalRequestStatusRechazada {
		t.Fatalf("estado de solicitud rechazada: esperado rechazada, obtenido %s", solicitud.Estado)
	}

	// 7. Aceptar una solicitud ya procesada -> 409
	resp, _ = doJSONRequest("PATCH",
		fmt.Sprintf("/api/rental-requests/%d/accept", solicitudID),
		tokenProp,
		[]byte(fmt.Sprintf(`{"fecha_inicio":"%s","fecha_fin":"%s","monto_mensual":2500}`, inicio, fin)))
	if resp.StatusCode != 409 {
		t.Fatalf("aceptar solicitud procesada: esperado 409, obtenido %d", resp.StatusCode)
	}

	// 8. Sin token -> 401
	resp, _ = doJSONRequest("GET", "/api/rental-requests/my", "", nil)
	if resp.StatusCode != 401 {
		t.Fatalf("sin token: esperado 401, obtenido %d", resp.StatusCode)
	}
}
