package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// RentalRequestHandler - Handler para solicitudes de alquiler
type RentalRequestHandler struct {
	DB *gorm.DB
}

// NewRentalRequestHandler - Crea un nuevo handler de solicitudes de alquiler
func NewRentalRequestHandler() (*RentalRequestHandler, error) {
	dbConfig, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	return &RentalRequestHandler{
		DB: dbConfig.DB,
	}, nil
}

// CreateRentalRequestRequest - Body para crear solicitud de alquiler
type CreateRentalRequestRequest struct {
	Mensaje          string `json:"mensaje"`
	TelefonoContacto string `json:"telefono_contacto"`
	HabitacionID     *uint  `json:"habitacion_id,omitempty"` // requerida si la propiedad se alquila por habitaciones
}

// CreateRentalRequest - El interesado solicita alquilar una propiedad
// POST /api/properties/:id/rental-request
func (h *RentalRequestHandler) CreateRentalRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	propertyID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de propiedad inválido",
		})
	}

	var req CreateRentalRequestRequest
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Error parseando request",
			})
		}
	}

	// Validar que la propiedad existe, es de alquiler y está activa
	var prop models.Property
	if err := h.DB.Where("id = ? AND deleted_at IS NULL", propertyID).First(&prop).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Propiedad no encontrada",
		})
	}

	if prop.TipoOperacion != models.OperationTypeAlquiler && prop.TipoOperacion != models.OperationTypeAnticretico {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Solo se puede solicitar alquiler o anticrético en este tipo de propiedad",
		})
	}

	if prop.EstadoPropiedad != models.PropertyStatusActivo {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "La propiedad no está disponible",
		})
	}

	// Habitación: obligatoria si la propiedad se alquila por habitaciones
	var habitacionID *uint
	if prop.AlquilaPorHabitaciones {
		if req.HabitacionID == nil || *req.HabitacionID == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Debes elegir una habitación disponible",
			})
		}
		var hab models.Habitacion
		if err := h.DB.Where("id = ? AND property_id = ? AND deleted_at IS NULL", *req.HabitacionID, propertyID).
			First(&hab).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "La habitación no existe en esta propiedad",
			})
		}
		if !hab.Activa {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Esa habitación no está disponible (en mantenimiento)",
			})
		}
		// Ocupada = tiene contrato activo
		var activos int64
		h.DB.Model(&models.RentalContract{}).
			Where("habitacion_id = ? AND estado = ? AND deleted_at IS NULL",
				hab.ID, models.ContractStatusActivo).
			Count(&activos)
		if activos > 0 {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Esa habitación ya está alquilada. Elige otra disponible",
			})
		}
		habitacionID = req.HabitacionID
	} else if prop.Alquilada {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Esta propiedad ya está alquilada",
		})
	}

	// No puede solicitar su propia propiedad
	if prop.UserID == userID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No puedes solicitar alquilar tu propia propiedad",
		})
	}

	// Evitar solicitudes duplicadas pendientes del mismo usuario para la misma habitación
	dupQuery := h.DB.Model(&models.RentalRequest{}).
		Where("property_id = ? AND inquilino_id = ? AND estado = ? AND deleted_at IS NULL",
			propertyID, userID, models.RentalRequestStatusPendiente)
	if habitacionID != nil {
		dupQuery = dupQuery.Where("habitacion_id = ?", *habitacionID)
	} else {
		dupQuery = dupQuery.Where("habitacion_id IS NULL")
	}
	var dupCount int64
	dupQuery.Count(&dupCount)
	if dupCount > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Ya tienes una solicitud pendiente para esta propiedad",
		})
	}

	solicitud := models.RentalRequest{
		PropertyID:       uint(propertyID),
		InquilinoID:      userID,
		VendedorID:       prop.UserID,
		HabitacionID:     habitacionID,
		Mensaje:          strings.TrimSpace(req.Mensaje),
		TelefonoContacto: strings.TrimSpace(req.TelefonoContacto),
		Estado:           models.RentalRequestStatusPendiente,
	}

	if err := h.DB.Create(&solicitud).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creando solicitud",
		})
	}

	// Cargar relaciones
	h.DB.Preload("Property").Preload("Inquilino").First(&solicitud, solicitud.ID)

	// 🔔 Notificar al propietario
	notificacion := models.Notification{
		UserID:     prop.UserID,
		PropertyID: &prop.ID,
		Tipo:       "solicitud_alquiler",
		Titulo:     "Nueva solicitud de alquiler",
		Mensaje:    solicitud.Inquilino.Nombre + " quiere alquilar \"" + prop.Titulo + "\"",
		IsRead:     false,
	}
	h.DB.Create(&notificacion)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":   "Solicitud enviada. El propietario será notificado",
		"solicitud": solicitud,
	})
}

// GetRentalRequests - Lista solicitudes recibidas por el propietario
// GET /api/panel/rental-requests?estado=pendiente
func (h *RentalRequestHandler) GetRentalRequests(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	estado := c.Query("estado", "")

	query := h.DB.Where("vendedor_id = ?", userID)
	if estado != "" {
		query = query.Where("estado = ?", estado)
	}

	var solicitudes []models.RentalRequest
	if err := query.
		Preload("Property").
		Preload("Property.PropertyImages", func(db *gorm.DB) *gorm.DB {
			return db.Order("principal DESC, orden ASC")
		}).
		Preload("Inquilino").
		Preload("Habitacion").
		Order("created_at DESC").
		Find(&solicitudes).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo solicitudes",
		})
	}

	return c.JSON(fiber.Map{
		"solicitudes": solicitudes,
		"total":       len(solicitudes),
	})
}

// GetMyRentalRequests - Lista solicitudes enviadas por el interesado
// GET /api/rental-requests/my
func (h *RentalRequestHandler) GetMyRentalRequests(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	estado := c.Query("estado", "")

	query := h.DB.Where("inquilino_id = ?", userID)
	if estado != "" {
		query = query.Where("estado = ?", estado)
	}

	var solicitudes []models.RentalRequest
	if err := query.
		Preload("Property").
		Preload("Property.PropertyImages", func(db *gorm.DB) *gorm.DB {
			return db.Order("principal DESC, orden ASC")
		}).
		Preload("Habitacion").
		Order("created_at DESC").
		Find(&solicitudes).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo solicitudes",
		})
	}

	return c.JSON(fiber.Map{
		"solicitudes": solicitudes,
		"total":       len(solicitudes),
	})
}

// AcceptRentalRequestRequest - Body para aceptar solicitud (datos del contrato)
type AcceptRentalRequestRequest struct {
	FechaInicio   string  `json:"fecha_inicio"`   // RFC3339 o YYYY-MM-DD
	FechaFin      string  `json:"fecha_fin"`      // RFC3339 o YYYY-MM-DD
	MontoMensual  float64 `json:"monto_mensual"`  // Renta mensual acordada (alquiler)
	MontoAnticres float64 `json:"monto_anticres"` // Anticrés único (anticrético)
}

// AcceptRentalRequest - El propietario acepta la solicitud: crea el contrato,
// genera las cuotas automáticas, registra el depósito y marca la propiedad como alquilada
// PATCH /api/rental-requests/:id/accept
func (h *RentalRequestHandler) AcceptRentalRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	requestID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de solicitud inválido",
		})
	}

	var req AcceptRentalRequestRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	// Parsear fechas (acepta RFC3339 o YYYY-MM-DD)
	fechaInicio, err := parseFecha(req.FechaInicio)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de fecha_inicio inválido (use YYYY-MM-DD)",
		})
	}
	fechaFin, err := parseFecha(req.FechaFin)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de fecha_fin inválido (use YYYY-MM-DD)",
		})
	}
	if !fechaFin.After(fechaInicio) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "fecha_fin debe ser posterior a fecha_inicio",
		})
	}

	// Cargar solicitud
	var solicitud models.RentalRequest
	if err := h.DB.Where("id = ? AND vendedor_id = ? AND deleted_at IS NULL", requestID, userID).
		Preload("Property").
		Preload("Inquilino").
		Preload("Habitacion").
		First(&solicitud).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Solicitud no encontrada",
		})
	}

	if solicitud.Estado != models.RentalRequestStatusPendiente {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "La solicitud ya fue procesada (estado: " + string(solicitud.Estado) + ")",
		})
	}

	// Validar propiedad
	prop := solicitud.Property
	if prop.EstadoPropiedad != models.PropertyStatusActivo {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "La propiedad ya no está disponible",
		})
	}

	// Revalidar disponibilidad de la habitación elegida
	if solicitud.HabitacionID != nil {
		var activos int64
		h.DB.Model(&models.RentalContract{}).
			Where("habitacion_id = ? AND estado = ? AND deleted_at IS NULL",
				*solicitud.HabitacionID, models.ContractStatusActivo).
			Count(&activos)
		if activos > 0 {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "La habitación elegida ya fue alquilada. Rechaza esta solicitud y pide al interesado elegir otra",
			})
		}
	} else if prop.Alquilada {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "La propiedad ya está alquilada",
		})
	}

	esAnticretico := prop.TipoOperacion == models.OperationTypeAnticretico

	// Validar monto según el tipo de operación
	if esAnticretico {
		if req.MontoAnticres <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "monto_anticres debe ser mayor a 0",
			})
		}
	} else {
		if req.MontoMensual <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "monto_mensual debe ser mayor a 0",
			})
		}
	}

	// Preparar datos según el tipo de contrato
	var montoDeposito float64
	var cuotas []models.RentPayment

	if esAnticretico {
		// Anticrético: sin cuotas mensuales ni depósito; un solo pago de anticrés
		cuotas = nil
		montoDeposito = 0
	} else {
		// Calcular depósito a partir de la publicación (ej: "1 mes", "2 meses")
		montoDeposito = calcularDeposito(prop.Deposito, req.MontoMensual)

		// Generar cuotas: una por cada mes entre fecha_inicio y fecha_fin
		cuotas, err = generarCuotas(fechaInicio, fechaFin, req.MontoMensual)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
	}

	// ===== TRANSACCIÓN: contrato + cuotas/anticrés + propiedad + solicitudes =====
	var contrato models.RentalContract
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Crear contrato
		tipoContrato := models.ContractTypeAlquiler
		montoMensual := req.MontoMensual
		if esAnticretico {
			tipoContrato = models.ContractTypeAnticretico
			montoMensual = 0 // El anticrético no tiene renta mensual
		}

		contrato = models.RentalContract{
			PropertyID:    prop.ID,
			InquilinoID:   solicitud.InquilinoID,
			VendedorID:    userID,
			HabitacionID:  solicitud.HabitacionID,
			FechaInicio:   fechaInicio,
			FechaFin:      fechaFin,
			MontoMensual:  montoMensual,
			TipoContrato:  tipoContrato,
			MontoDeposito: montoDeposito,
			Estado:        models.ContractStatusActivo,
		}
		if err := tx.Create(&contrato).Error; err != nil {
			return err
		}

		// 2. Generar cuotas mensuales automáticas (solo alquiler)
		if len(cuotas) > 0 {
			for i := range cuotas {
				cuotas[i].ContractID = contrato.ID
				cuotas[i].InquilinoID = solicitud.InquilinoID
				cuotas[i].VendedorID = userID
			}
			if err := tx.Create(&cuotas).Error; err != nil {
				return err
			}
		}

		// 3. Registrar el depósito como pago especial (si aplica)
		if montoDeposito > 0 {
			deposito := models.RentPayment{
				ContractID:         contrato.ID,
				InquilinoID:        solicitud.InquilinoID,
				VendedorID:         userID,
				MesCorrespondiente: int(fechaInicio.Month()),
				Anio:               fechaInicio.Year(),
				Tipo:               models.PaymentTypeDeposito,
				Monto:              montoDeposito,
				FechaPago:          fechaInicio,
				Estado:             models.PaymentStatusPendiente,
			}
			if err := tx.Create(&deposito).Error; err != nil {
				return err
			}
		}

		// 4. Registrar el anticrés como pago único (anticrético)
		if esAnticretico {
			anticres := models.RentPayment{
				ContractID:         contrato.ID,
				InquilinoID:        solicitud.InquilinoID,
				VendedorID:         userID,
				MesCorrespondiente: int(fechaInicio.Month()),
				Anio:               fechaInicio.Year(),
				Tipo:               models.PaymentTypeAnticres,
				Monto:              req.MontoAnticres,
				FechaPago:          fechaInicio,
				Estado:             models.PaymentStatusPendiente,
			}
			if err := tx.Create(&anticres).Error; err != nil {
				return err
			}
		}

		// 5. Marcar propiedad como alquilada:
		//    - Propiedad completa → siempre.
		//    - Por habitaciones → solo cuando ya NO quedan habitaciones activas libres.
		if prop.AlquilaPorHabitaciones {
			var habitacionesActivas, ocupadas int64
			tx.Model(&models.Habitacion{}).
				Where("property_id = ? AND activa = ? AND deleted_at IS NULL", prop.ID, true).
				Count(&habitacionesActivas)
			if habitacionesActivas == 0 {
				// Sin habitaciones configuradas: comportamiento de propiedad completa
				tx.Model(&models.Property{}).Where("id = ?", prop.ID).Update("alquilada", true)
			} else {
				// Ocupadas = habitaciones con contrato activo (incluida la recién creada)
				tx.Model(&models.RentalContract{}).
					Where("property_id = ? AND estado = ? AND habitacion_id IS NOT NULL AND deleted_at IS NULL",
						prop.ID, models.ContractStatusActivo).
					Count(&ocupadas)
				// +1 por el contrato recién creado dentro de la transacción
				if ocupadas+1 >= habitacionesActivas {
					tx.Model(&models.Property{}).Where("id = ?", prop.ID).Update("alquilada", true)
				} else {
					tx.Model(&models.Property{}).Where("id = ?", prop.ID).Update("alquilada", false)
				}
			}
		} else {
			if err := tx.Model(&models.Property{}).Where("id = ?", prop.ID).
				Update("alquilada", true).Error; err != nil {
				return err
			}
		}

		// 6. Cancelar las demás solicitudes pendientes SOLO de esta misma habitación
		//    (las de otras habitaciones siguen vigentes)
		cancelQuery := tx.Model(&models.RentalRequest{}).
			Where("property_id = ? AND id != ? AND estado = ? AND deleted_at IS NULL",
				prop.ID, solicitud.ID, models.RentalRequestStatusPendiente)
		if solicitud.HabitacionID != nil {
			cancelQuery = cancelQuery.Where("habitacion_id = ?", *solicitud.HabitacionID)
		} else {
			cancelQuery = cancelQuery.Where("habitacion_id IS NULL")
		}
		if err := cancelQuery.Update("estado", models.RentalRequestStatusCancelada).Error; err != nil {
			return err
		}

		// 7. Vincular solicitud con contrato
		if err := tx.Model(&models.RentalRequest{}).Where("id = ?", solicitud.ID).
			Updates(map[string]interface{}{
				"estado":      models.RentalRequestStatusAceptada,
				"contract_id": contrato.ID,
			}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creando el contrato: " + err.Error(),
		})
	}

	// Cargar relaciones del contrato
	h.DB.Preload("Property").Preload("Inquilino").Preload("Vendedor").First(&contrato, contrato.ID)

	// 🔔 Notificar al inquilino
	mensajeNotif := "El propietario aceptó tu solicitud para \"" + prop.Titulo + "\". El contrato ya está activo"
	if esAnticretico {
		mensajeNotif = "El propietario aceptó tu solicitud anticrética para \"" + prop.Titulo + "\". Registra el pago del anticrés"
	}
	notificacion := models.Notification{
		UserID:     solicitud.InquilinoID,
		PropertyID: &prop.ID,
		Tipo:       "solicitud_aceptada",
		Titulo:     "Tu solicitud fue aceptada",
		Mensaje:    mensajeNotif,
		IsRead:     false,
	}
	h.DB.Create(&notificacion)

	respuesta := fiber.Map{
		"message":          "Contrato creado exitosamente",
		"contrato":         contrato,
		"cuotas_generadas": len(cuotas),
		"monto_deposito":   montoDeposito,
	}
	if esAnticretico {
		respuesta["monto_anticres"] = req.MontoAnticres
	}

	return c.JSON(respuesta)
}

// RejectRentalRequest - El propietario rechaza la solicitud
// PATCH /api/rental-requests/:id/reject
func (h *RentalRequestHandler) RejectRentalRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	requestID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de solicitud inválido",
		})
	}

	var solicitud models.RentalRequest
	if err := h.DB.Where("id = ? AND vendedor_id = ? AND deleted_at IS NULL", requestID, userID).
		Preload("Property").
		First(&solicitud).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Solicitud no encontrada",
		})
	}

	if solicitud.Estado != models.RentalRequestStatusPendiente {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "La solicitud ya fue procesada (estado: " + string(solicitud.Estado) + ")",
		})
	}

	if err := h.DB.Model(&solicitud).Update("estado", models.RentalRequestStatusRechazada).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error rechazando solicitud",
		})
	}

	// 🔔 Notificar al interesado
	notificacion := models.Notification{
		UserID:     solicitud.InquilinoID,
		PropertyID: &solicitud.PropertyID,
		Tipo:       "solicitud_rechazada",
		Titulo:     "Solicitud de alquiler rechazada",
		Mensaje:    "El propietario de \"" + solicitud.Property.Titulo + "\" rechazó tu solicitud",
		IsRead:     false,
	}
	h.DB.Create(&notificacion)

	return c.JSON(fiber.Map{
		"message": "Solicitud rechazada",
	})
}

// ===== HELPERS =====

// parseFecha - Acepta RFC3339 o YYYY-MM-DD
func parseFecha(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}

// calcularDeposito - Convierte el texto de depósito de la publicación ("1 mes", "2 meses")
// en un monto: meses × renta mensual. Si no se puede parsear, devuelve 0.
func calcularDeposito(deposito string, montoMensual float64) float64 {
	deposito = strings.ToLower(strings.TrimSpace(deposito))
	if deposito == "" || deposito == "no" || deposito == "sin deposito" || deposito == "sin depósito" {
		return 0
	}

	// Extraer el primer número del texto
	num := ""
	for _, r := range deposito {
		if r >= '0' && r <= '9' {
			num += string(r)
		} else if num != "" {
			break
		}
	}
	meses, err := strconv.Atoi(num)
	if err != nil || meses <= 0 {
		return 0
	}
	if meses > 12 {
		meses = 12
	}
	return montoMensual * float64(meses)
}

// generarCuotas - Genera una cuota pendiente por cada mes entre fecha_inicio y fecha_fin.
// La fecha de vencimiento de cada cuota es el mismo día del mes de inicio.
func generarCuotas(fechaInicio, fechaFin time.Time, monto float64) ([]models.RentPayment, error) {
	// Normalizar a medianoche UTC para comparaciones estables
	inicio := time.Date(fechaInicio.Year(), fechaInicio.Month(), fechaInicio.Day(), 0, 0, 0, 0, time.UTC)
	fin := time.Date(fechaFin.Year(), fechaFin.Month(), fechaFin.Day(), 0, 0, 0, 0, time.UTC)

	// Máximo 10 años de contrato (seguridad)
	maxCuotas := 120

	var cuotas []models.RentPayment
	cursor := inicio
	for cursor.Before(fin) && len(cuotas) < maxCuotas {
		cuotas = append(cuotas, models.RentPayment{
			MesCorrespondiente: int(cursor.Month()),
			Anio:               cursor.Year(),
			Tipo:               models.PaymentTypeCuota,
			Monto:              monto,
			FechaPago:          cursor, // fecha de vencimiento de la cuota
			Estado:             models.PaymentStatusPendiente,
		})
		cursor = cursor.AddDate(0, 1, 0)
	}

	if len(cuotas) == 0 {
		return nil, fmt.Errorf("el rango de fechas no genera cuotas")
	}

	return cuotas, nil
}
