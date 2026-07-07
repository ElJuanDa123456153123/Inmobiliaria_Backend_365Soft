package services

import (
	"context"
	"fmt"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"gorm.io/gorm"
)

// QrPagoService - Servicio de pagos QR
type QrPagoService struct {
	db         *gorm.DB
	bancoQr    *BancoQrService
	cfg        *config.Config
}

// NewQrPagoService crea una nueva instancia del servicio
func NewQrPagoService(db *gorm.DB, cfg *config.Config) *QrPagoService {
	return &QrPagoService{
		db:      db,
		bancoQr: NewBancoQrService(cfg),
		cfg:     cfg,
	}
}

// GenerarQrParams - Parámetros para generar un QR
type GenerarQrParams struct {
	PropertyID *uint // Puede ser nil si se compra el plan antes de tener la propiedad
	UserID     uint
	PlanID     uint
}

// GenerarQr genera un QR dinámico para publicar una propiedad
func (s *QrPagoService) GenerarQr(ctx context.Context, params GenerarQrParams) (*models.QrPago, error) {
	// 1. Obtener el plan para conocer el monto
	var plan models.Plan
	if err := s.db.First(&plan, params.PlanID).Error; err != nil {
		return nil, fmt.Errorf("error obteniendo plan: %w", err)
	}

	// 2. Generar alias único (usar user_id si property_id es 0)
	alias := s.bancoQr.GenerarAliasUnico(params.PropertyID)

	// 3. Calcular fecha de vencimiento (15 minutos)
	fechaVencimiento := s.bancoQr.CalcularFechaVencimiento(15)

	// 4. Generar detalle de la glosa
	detalleGlosa := fmt.Sprintf("Publicación de propiedad - Plan %s", plan.Nombre)

	// 5. Llamar al banco MC4
	respuestaBanco, err := s.bancoQr.GenerarQrDinamico(ctx, struct {
		Alias            string
		Monto            float64
		DetalleGlosa     string
		FechaVencimiento time.Time
	}{
		Alias:            alias,
		Monto:            plan.Precio,
		DetalleGlosa:     detalleGlosa,
		FechaVencimiento: fechaVencimiento,
	})

	if err != nil {
		return nil, fmt.Errorf("error generando QR en banco: %w", err)
	}

	if !s.bancoQr.ValidarRespuestaBanco(respuestaBanco.Codigo) {
		return nil, fmt.Errorf("error generando QR: %s", respuestaBanco.Mensaje)
	}

	// 6. Guardar QR en BD (property_id puede ser 0)
	// Usar el alias que generamos (no el del banco, por si viene vacío)
	aliasBanco := respuestaBanco.Objeto.Alias
	if aliasBanco == "" {
		aliasBanco = alias // Fallback al alias generado
	}

	qrPago := &models.QrPago{
		Alias:            aliasBanco,
		Estado:           "PENDIENTE",
		Monto:            plan.Precio,
		Moneda:           "BOB",
		PropertyID:       params.PropertyID, // Puede ser 0
		UserID:           params.UserID,
		PlanID:           params.PlanID,
		FechaVencimiento: fechaVencimiento,
		ImagenQr:         respuestaBanco.Objeto.ImagenQr,
		DetalleGlosa:     detalleGlosa,
	}

	// Validar que el alias no esté vacío antes de guardar
	if qrPago.Alias == "" {
		return nil, fmt.Errorf("el alias del banco está vacío, no se puede crear el QR")
	}

	if err := s.db.Create(qrPago).Error; err != nil {
		return nil, fmt.Errorf("error guardando QR en BD: %w", err)
	}

	return qrPago, nil
}

// VerificarEstadoResponse - Respuesta de verificación de estado
type VerificarEstadoResponse struct {
	Estado        string `json:"estado"`
	PagoProcesado bool   `json:"pago_procesado"`
}

// VerificarEstado consulta el estado del QR al banco y actualiza la BD
func (s *QrPagoService) VerificarEstado(ctx context.Context, qrPagoID uint) (*VerificarEstadoResponse, error) {
	// 1. Obtener QR de BD
	var qrPago models.QrPago
	if err := s.db.First(&qrPago, qrPagoID).Error; err != nil {
		return nil, fmt.Errorf("QR no encontrado: %w", err)
	}

	// 2. Consultar al banco
	respuestaBanco, err := s.bancoQr.VerificarEstadoQr(ctx, qrPago.Alias)
	if err != nil {
		// Fallback: retornar estado de BD si falla el banco
		return &VerificarEstadoResponse{
			Estado:        qrPago.Estado,
			PagoProcesado: false,
		}, nil
	}

	if !s.bancoQr.ValidarRespuestaBanco(respuestaBanco.Codigo) {
		return nil, fmt.Errorf("error verificando estado: %s", respuestaBanco.Mensaje)
	}

	estadoActual := respuestaBanco.Objeto.EstadoActual
	estadoAnterior := qrPago.Estado

	// 3. Actualizar estado en BD
	if err := s.db.Model(&qrPago).Update("estado", estadoActual).Error; err != nil {
		return nil, fmt.Errorf("error actualizando estado: %w", err)
	}

	// 4. Si recién se pagó, procesar
	pagoProcesado := false
	if estadoActual == "PAGADO" && estadoAnterior != "PAGADO" {
		if err := s.procesarPago(qrPagoID); err != nil {
			return nil, fmt.Errorf("error procesando pago: %w", err)
		}
		pagoProcesado = true
	}

	// 5. Crear notificación si el QR expiró o fue rechazado
	if (estadoActual == "EXPIRADO" || estadoActual == "VENCIDO") && estadoAnterior != "EXPIRADO" && estadoAnterior != "VENCIDO" {
		// Obtener nombre del plan
		var plan models.Plan
		planNombre := "tu plan"
		if err := s.db.Select("nombre").First(&plan, qrPago.PlanID).Error; err == nil {
			planNombre = plan.Nombre
		}

		notification := models.Notification{
			UserID:      qrPago.UserID,
			Tipo:        "pago_error",
			Titulo:      "Pago expirado",
			Mensaje:     fmt.Sprintf("Tu pago de Bs. %.2f por el plan '%s' ha expirado. Por favor, genera un nuevo QR para completar tu pago.", qrPago.Monto, planNombre),
			IsRead:      false,
		}
		if qrPago.PropertyID != nil && *qrPago.PropertyID > 0 {
			notification.PropertyID = qrPago.PropertyID
		}
		s.db.Create(&notification)
		fmt.Printf("⏰ [VERIFICAR_ESTADO] Notificación de expiración creada para usuario %d\n", qrPago.UserID)
	}

	if estadoActual == "RECHAZADO" && estadoAnterior != "RECHAZADO" {
		notification := models.Notification{
			UserID:      qrPago.UserID,
			Tipo:        "pago_error",
			Titulo:      "Pago rechazado",
			Mensaje:     fmt.Sprintf("Tu pago de Bs. %.2f ha sido rechazado. Por favor, verifica los datos e intenta nuevamente.", qrPago.Monto),
			IsRead:      false,
		}
		if qrPago.PropertyID != nil && *qrPago.PropertyID > 0 {
			notification.PropertyID = qrPago.PropertyID
		}
		s.db.Create(&notification)
		fmt.Printf("❌ [VERIFICAR_ESTADO] Notificación de rechazo creada para usuario %d\n", qrPago.UserID)
	}

	return &VerificarEstadoResponse{
		Estado:        estadoActual,
		PagoProcesado: pagoProcesado,
	}, nil
}

// procesarPago procesa un pago exitoso: activa la propiedad y crea suscripción
func (s *QrPagoService) procesarPago(qrPagoID uint) error {
	fmt.Printf("🔧 [PROCESAR_PAGO] Iniciando procesamiento del pago ID: %d\n", qrPagoID)

	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Obtener QR con relaciones
		var qrPago models.QrPago
		if err := tx.Preload("Plan").First(&qrPago, qrPagoID).Error; err != nil {
			fmt.Printf("❌ [PROCESAR_PAGO] Error obteniendo QR: %v\n", err)
			return err
		}

		fmt.Printf("📋 [PROCESAR_PAGO] QR encontrado - UserID: %d, PlanID: %d, Plan: %v\n", qrPago.UserID, qrPago.PlanID, qrPago.Plan)

		// 2. Marcar QR como PAGADO con fecha de proceso
		now := time.Now()
		if err := tx.Model(&qrPago).Updates(map[string]interface{}{
			"estado":        "PAGADO",
			"fecha_proceso": &now,
		}).Error; err != nil {
			fmt.Printf("❌ [PROCESAR_PAGO] Error actualizando QR: %v\n", err)
			return err
		}

		// 3. Buscar suscripción seleccionada del MISMO plan
		var existingSubscription models.Subscription
		result := tx.Where("user_id = ? AND plan_id = ? AND seleccionada = ?", qrPago.UserID, qrPago.PlanID, true).First(&existingSubscription)

		fmt.Printf("🔍 [PROCESAR_PAGO] Buscando suscripción seleccionada del mismo plan - result.Error: %v\n", result.Error)

		// Desmarcar suscripciones de DIFERENTE plan (si paga el mismo plan 2 veces, se acumulan)
		if err := tx.Model(&models.Subscription{}).
			Where("user_id = ? AND plan_id != ? AND seleccionada = ?", qrPago.UserID, qrPago.PlanID, true).
			Update("seleccionada", false).Error; err != nil {
			fmt.Printf("⚠️  [PROCESAR_PAGO] Error desmarcando suscripciones de otros planes: %v\n", err)
		}

		if result.Error == gorm.ErrRecordNotFound {
			// No hay suscripción seleccionada de este plan
			// Crear nueva suscripción
			fmt.Printf("✅ [PROCESAR_PAGO] Creando nueva suscripción (no hay selección de este plan)\n")

			fechaFin := time.Now().AddDate(0, qrPago.Plan.DuracionMeses, 0)

			subscription := models.Subscription{
				UserID:                qrPago.UserID,
				PlanID:                qrPago.PlanID,
				FechaInicio:           time.Now(),
				FechaFin:              &fechaFin,
				PropiedadesUsadas:     0, // Iniciar en 0 (todavía no publicó nada)
				Estado:                "ACTIVA",
				RenovacionAutomatica:  true,
				Seleccionada:          true, // Marcar como seleccionada
			}

			fmt.Printf("💾 [PROCESAR_PAGO] Guardando suscripción: UserID=%d, PlanID=%d, FechaFin=%v\n", subscription.UserID, subscription.PlanID, subscription.FechaFin)

			if err := tx.Create(&subscription).Error; err != nil {
				fmt.Printf("❌ [PROCESAR_PAGO] Error creando suscripción: %v\n", err)
				return fmt.Errorf("error creando suscripción: %w", err)
			}

			fmt.Printf("✅ [PROCESAR_PAGO] Suscripción creada exitosamente, ID: %d\n", subscription.ID)
		} else if result.Error != nil {
			fmt.Printf("❌ [PROCESAR_PAGO] Error buscando suscripción: %v\n", result.Error)
			return result.Error
		} else {
			// Ya existe una suscripción seleccionada de este plan
			// CREAR OTRA NUEVA suscripción (acumular planes del mismo tipo)
			fmt.Printf("➕ [PROCESAR_PAGO] Ya existe suscripción seleccionada de este plan, creando otra para acumular\n")

			fechaFin := time.Now().AddDate(0, qrPago.Plan.DuracionMeses, 0)

			// Desmarcar la suscripción anterior del mismo plan
			if err := tx.Model(&existingSubscription).
				Update("seleccionada", false).Error; err != nil {
				fmt.Printf("⚠️  [PROCESAR_PAGO] Error desmarcando suscripción anterior: %v\n", err)
			}

			subscription := models.Subscription{
				UserID:                qrPago.UserID,
				PlanID:                qrPago.PlanID,
				FechaInicio:           time.Now(),
				FechaFin:              &fechaFin,
				PropiedadesUsadas:     0, // Iniciar en 0 (todavía no publicó nada)
				Estado:                "ACTIVA",
				RenovacionAutomatica:  true,
				Seleccionada:          true, // Marcar como seleccionada
			}

			fmt.Printf("💾 [PROCESAR_PAGO] Guardando nueva suscripción: UserID=%d, PlanID=%d, FechaFin=%v\n", subscription.UserID, subscription.PlanID, subscription.FechaFin)

			if err := tx.Create(&subscription).Error; err != nil {
				fmt.Printf("❌ [PROCESAR_PAGO] Error creando nueva suscripción: %v\n", err)
				return fmt.Errorf("error creando nueva suscripción: %w", err)
			}

			fmt.Printf("✅ [PROCESAR_PAGO] Nueva suscripción creada para acumular, ID: %d\n", subscription.ID)
		}

		// 4. Activar la propiedad (solo si existe property_id)
		var propertyTitulo string
		if qrPago.PropertyID != nil && *qrPago.PropertyID > 0 {
			fmt.Printf("🏠 [PROCESAR_PAGO] Activando propiedad ID: %d\n", *qrPago.PropertyID)
			if err := tx.Model(&models.Property{}).Where("id = ?", *qrPago.PropertyID).Update("estado_propiedad", "activo").Error; err != nil {
				fmt.Printf("❌ [PROCESAR_PAGO] Error activando propiedad: %v\n", err)
				return fmt.Errorf("error activando propiedad: %w", err)
			}
			fmt.Printf("✅ [PROCESAR_PAGO] Propiedad activada\n")

			// Obtener título de la propiedad para la notificación
			var property models.Property
			if err := tx.Select("titulo").First(&property, *qrPago.PropertyID).Error; err == nil {
				propertyTitulo = property.Titulo
			}
		} else {
			fmt.Printf("⏭️  [PROCESAR_PAGO] No hay propiedad para activar (property_id es nil o 0)\n")
		}

		// 5. Crear notificación de pago exitoso para el usuario
		mensaje := fmt.Sprintf("Tu pago de Bs. %.2f por el plan '%s' ha sido confirmado exitosamente.",
			qrPago.Monto, qrPago.Plan.Nombre)
		titulo := "¡Pago confirmado!"

		if propertyTitulo != "" {
			mensaje += fmt.Sprintf(" Tu propiedad '%s' ya está activa.", propertyTitulo)
		} else {
			mensaje += " Ya puedes publicar tus propiedades."
		}

		notification := models.Notification{
			UserID:      qrPago.UserID,
			Tipo:        "pago_confirmado",
			Titulo:      titulo,
			Mensaje:     mensaje,
			IsRead:      false,
		}

		// Asignar PropertyID si existe
		if qrPago.PropertyID != nil && *qrPago.PropertyID > 0 {
			notification.PropertyID = qrPago.PropertyID
		}

		if err := tx.Create(&notification).Error; err != nil {
			fmt.Printf("⚠️  [PROCESAR_PAGO] Error creando notificación: %v\n", err)
			// No fallar el proceso si error la notificación
		} else {
			fmt.Printf("🔔 [PROCESAR_PAGO] Notificación creada para usuario %d\n", qrPago.UserID)
		}

		fmt.Printf("✅ [PROCESAR_PAGO] Procesamiento completado exitosamente\n")
		return nil
	})
}

// ProcesarWebhook procesa el webhook del banco (notificación de pago)
func (s *QrPagoService) ProcesarWebhook(ctx context.Context, payload BancoWebhookPayload) (map[string]string, error) {
	// 1. Buscar QR por alias
	var qrPago models.QrPago
	if err := s.db.Where("alias = ?", payload.Alias).First(&qrPago).Error; err != nil {
		return map[string]string{
			"codigo": "1212",
			"mensaje": "Alias no encontrado",
		}, nil
	}

	// GUARDAR estado anterior ANTES de actualizar
	estadoAnterior := qrPago.Estado

	// 2. Actualizar datos del pago
	now := time.Now()
	fechaProceso := now
	if payload.Fechaproceso != "" {
		// Intentar parsear la fecha del banco
		if parsed, err := time.Parse("2006-01-02T15:04:05", payload.Fechaproceso); err == nil {
			fechaProceso = parsed
		}
	}

	updates := map[string]interface{}{
		"estado":            "PAGADO",
		"numero_orden":      payload.NumeroOrdenOriginante,
		"nombre_cliente":    payload.NombreCliente,
		"documento_cliente": payload.DocumentoCliente,
		"cuenta_cliente":    payload.CuentaCliente,
		"fecha_proceso":     &fechaProceso,
	}

	if err := s.db.Model(&qrPago).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 3. Procesar el pago si aún no se ha procesado (usar estadoAnterior guardado)
	if estadoAnterior != "PAGADO" {
		fmt.Printf("✅ [WEBHOOK] Procesando pago, estado anterior: %s\n", estadoAnterior)
		if err := s.procesarPago(qrPago.ID); err != nil {
			fmt.Printf("❌ [WEBHOOK] Error procesando pago: %v\n", err)
			return nil, err
		}
		fmt.Printf("✅ [WEBHOOK] Pago procesado exitosamente\n")
	}

	// 4. Retornar respuesta esperada por el banco
	return map[string]string{
		"codigo": "0000",
		"mensaje": "Registro Exitoso",
	}, nil
}

// CancelarQr cancela un QR que no ha sido pagado
func (s *QrPagoService) CancelarQr(qrPagoID uint) error {
	var qrPago models.QrPago
	if err := s.db.First(&qrPago, qrPagoID).Error; err != nil {
		return fmt.Errorf("QR no encontrado: %w", err)
	}

	if qrPago.Estado == "PAGADO" {
		return fmt.Errorf("no se puede cancelar un QR ya pagado")
	}

	// Actualizar estado en BD
	if err := s.db.Model(&qrPago).Update("estado", "CANCELADO").Error; err != nil {
		return err
	}

	// Opcional: Inhabilitar QR en el banco
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = s.bancoQr.InhabilitarQr(ctx, qrPago.Alias)

	return nil
}
