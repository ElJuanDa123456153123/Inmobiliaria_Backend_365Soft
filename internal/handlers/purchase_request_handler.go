package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// PurchaseRequestHandler - Handler para solicitudes de compra de propiedades en venta
type PurchaseRequestHandler struct {
	DB *gorm.DB
}

// NewPurchaseRequestHandler - Crea un nuevo handler de solicitudes de compra
func NewPurchaseRequestHandler() (*PurchaseRequestHandler, error) {
	dbConfig, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	return &PurchaseRequestHandler{
		DB: dbConfig.DB,
	}, nil
}

// CreatePurchaseRequestRequest - Body para crear solicitud de compra
type CreatePurchaseRequestRequest struct {
	Mensaje          string  `json:"mensaje"`
	TelefonoContacto string  `json:"telefono_contacto"`
	MontoOfertado    float64 `json:"monto_ofertado"` // opcional: oferta concreta; 0 = al precio publicado
}

// CreatePurchaseRequest - Un comprador interesado solicita comprar una propiedad en venta
// POST /api/properties/:id/purchase-request
func (h *PurchaseRequestHandler) CreatePurchaseRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	propertyID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de propiedad inválido",
		})
	}

	var req CreatePurchaseRequestRequest
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Error parseando request",
			})
		}
	}

	if req.MontoOfertado < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "El monto ofertado no puede ser negativo",
		})
	}

	// Validar que la propiedad existe, es de venta y está disponible
	var prop models.Property
	if err := h.DB.Where("id = ? AND deleted_at IS NULL", propertyID).First(&prop).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Propiedad no encontrada",
		})
	}

	if prop.TipoOperacion != models.OperationTypeVenta {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Solo se puede enviar una solicitud de compra en propiedades de tipo venta",
		})
	}

	if prop.EstadoPropiedad != models.PropertyStatusActivo {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Esta propiedad ya no está disponible (estado: " + string(prop.EstadoPropiedad) + ")",
		})
	}

	// No puede comprar su propia propiedad
	if prop.UserID == userID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No puedes enviar una solicitud de compra a tu propia propiedad",
		})
	}

	// Evitar solicitudes duplicadas pendientes del mismo comprador para la misma propiedad
	var dupCount int64
	h.DB.Model(&models.PurchaseRequest{}).
		Where("property_id = ? AND comprador_id = ? AND estado = ? AND deleted_at IS NULL",
			propertyID, userID, models.PurchaseRequestStatusPendiente).
		Count(&dupCount)
	if dupCount > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Ya tienes una solicitud de compra pendiente para esta propiedad",
		})
	}

	solicitud := models.PurchaseRequest{
		PropertyID:       uint(propertyID),
		CompradorID:      userID,
		VendedorID:       prop.UserID,
		Mensaje:          strings.TrimSpace(req.Mensaje),
		TelefonoContacto: strings.TrimSpace(req.TelefonoContacto),
		MontoOfertado:    req.MontoOfertado,
		Estado:           models.PurchaseRequestStatusPendiente,
	}

	if err := h.DB.Create(&solicitud).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creando la solicitud de compra",
		})
	}

	// Cargar relaciones
	h.DB.Preload("Property").Preload("Comprador").First(&solicitud, solicitud.ID)

	// 🔔 Notificar al propietario
	notificacion := models.Notification{
		UserID:     prop.UserID,
		PropertyID: &prop.ID,
		Tipo:       "solicitud_compra",
		Titulo:     "Nueva solicitud de compra",
		Mensaje:    solicitud.Comprador.Nombre + " quiere comprar \"" + prop.Titulo + "\"",
		IsRead:     false,
	}
	h.DB.Create(&notificacion)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":   "Solicitud de compra enviada. El propietario será notificado",
		"solicitud": solicitud,
	})
}

// GetMyPurchaseRequests - Lista las solicitudes de compra enviadas por el comprador
// GET /api/purchase-requests/my
func (h *PurchaseRequestHandler) GetMyPurchaseRequests(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	estado := c.Query("estado", "")

	query := h.DB.Where("comprador_id = ?", userID)
	if estado != "" {
		query = query.Where("estado = ?", estado)
	}

	var solicitudes []models.PurchaseRequest
	if err := query.
		Preload("Property").
		Preload("Property.PropertyImages", func(db *gorm.DB) *gorm.DB {
			return db.Order("principal DESC, orden ASC")
		}).
		Preload("Vendedor").
		Preload("Pagos", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
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

// GetReceivedPurchaseRequests - Lista las solicitudes de compra recibidas por el vendedor
// GET /api/panel/purchase-requests?estado=pendiente
func (h *PurchaseRequestHandler) GetReceivedPurchaseRequests(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	estado := c.Query("estado", "")

	query := h.DB.Where("vendedor_id = ?", userID)
	if estado != "" {
		query = query.Where("estado = ?", estado)
	}

	var solicitudes []models.PurchaseRequest
	if err := query.
		Preload("Property").
		Preload("Property.PropertyImages", func(db *gorm.DB) *gorm.DB {
			return db.Order("principal DESC, orden ASC")
		}).
		Preload("Comprador").
		Preload("Pagos", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
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

// AcceptPurchaseRequestRequest - Body para aceptar una compra: define el plan de pagos
type AcceptPurchaseRequestRequest struct {
	Modalidad     string  `json:"modalidad"`      // "unico" (default) | "arras_saldo"
	MontoAceptado float64 `json:"monto_aceptado"` // opcional: default = oferta si hay, si no precio publicado
	MontoArras    float64 `json:"monto_arras"`    // requerido si modalidad = arras_saldo
	FechaSaldo    string  `json:"fecha_saldo"`    // opcional YYYY-MM-DD (vencimiento del saldo/pago único)
}

// AcceptPurchaseRequest - El vendedor acepta la oferta: define el plan de pagos,
// marca la propiedad como vendida, rechaza las demás solicitudes y notifica al comprador
// PATCH /api/panel/purchase-requests/:id/accept
func (h *PurchaseRequestHandler) AcceptPurchaseRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	requestID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de solicitud inválido",
		})
	}

	var req AcceptPurchaseRequestRequest
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Error parseando request",
			})
		}
	}

	var solicitud models.PurchaseRequest
	if err := h.DB.Where("id = ? AND vendedor_id = ? AND deleted_at IS NULL", requestID, userID).
		Preload("Property").
		Preload("Comprador").
		First(&solicitud).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Solicitud no encontrada",
		})
	}

	if solicitud.Estado != models.PurchaseRequestStatusPendiente {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "La solicitud ya fue procesada (estado: " + string(solicitud.Estado) + ")",
		})
	}

	// Validar que la propiedad siga disponible
	prop := solicitud.Property
	if prop.EstadoPropiedad != models.PropertyStatusActivo {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "La propiedad ya no está disponible (estado: " + string(prop.EstadoPropiedad) + ")",
		})
	}

	// ===== PLAN DE PAGOS =====
	modalidad := strings.TrimSpace(req.Modalidad)
	if modalidad == "" {
		modalidad = "unico"
	}
	if modalidad != "unico" && modalidad != "arras_saldo" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "modalidad debe ser 'unico' o 'arras_saldo'",
		})
	}

	// Monto final de la venta: explícito > oferta del comprador > precio publicado
	montoAceptado := req.MontoAceptado
	if montoAceptado <= 0 {
		if solicitud.MontoOfertado > 0 {
			montoAceptado = solicitud.MontoOfertado
		} else {
			montoAceptado = prop.Precio
		}
	}

	// Fecha de vencimiento del saldo / pago único (opcional)
	var fechaSaldo *time.Time
	if strings.TrimSpace(req.FechaSaldo) != "" {
		f, err := parseFecha(req.FechaSaldo)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Formato de fecha_saldo inválido (use YYYY-MM-DD)",
			})
		}
		fechaSaldo = &f
	}

	hoy := time.Now()
	var pagos []models.SalePayment
	switch modalidad {
	case "arras_saldo":
		if req.MontoArras <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "monto_arras debe ser mayor a 0",
			})
		}
		if req.MontoArras >= montoAceptado {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Las arras deben ser menores al monto total de la venta",
			})
		}
		arrasVence := hoy.AddDate(0, 0, 7)
		pagos = append(pagos, models.SalePayment{
			Tipo:             "arras",
			Monto:            req.MontoArras,
			FechaVencimiento: &arrasVence,
		})
		pagos = append(pagos, models.SalePayment{
			Tipo:             "saldo",
			Monto:            montoAceptado - req.MontoArras,
			FechaVencimiento: fechaSaldo,
		})
	default: // unico
		pagos = append(pagos, models.SalePayment{
			Tipo:             "unico",
			Monto:            montoAceptado,
			FechaVencimiento: fechaSaldo,
		})
	}

	fechaAceptada := hoy

	// ===== TRANSACCIÓN: solicitud aceptada + propiedad vendida + otras rechazadas + pagos =====
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Aceptar la solicitud con el monto y fecha del cierre
		if err := tx.Model(&models.PurchaseRequest{}).Where("id = ?", solicitud.ID).
			Updates(map[string]interface{}{
				"estado":         models.PurchaseRequestStatusAceptada,
				"monto_aceptado": montoAceptado,
				"fecha_aceptada": fechaAceptada,
			}).Error; err != nil {
			return err
		}

		// 2. Marcar la propiedad como vendida
		if err := tx.Model(&models.Property{}).Where("id = ?", prop.ID).
			Update("estado_propiedad", models.PropertyStatusVendido).Error; err != nil {
			return err
		}

		// 3. Rechazar las demás solicitudes de compra pendientes de esta propiedad
		if err := tx.Model(&models.PurchaseRequest{}).
			Where("property_id = ? AND id != ? AND estado = ? AND deleted_at IS NULL",
				prop.ID, solicitud.ID, models.PurchaseRequestStatusPendiente).
			Update("estado", models.PurchaseRequestStatusRechazada).Error; err != nil {
			return err
		}

		// 4. Generar el plan de pagos
		for i := range pagos {
			pagos[i].PurchaseRequestID = solicitud.ID
			pagos[i].CompradorID = solicitud.CompradorID
			pagos[i].VendedorID = userID
		}
		if err := tx.Create(&pagos).Error; err != nil {
			return err
		}

		// 5. Generar el checklist documental de la venta (4 documentos requeridos al vendedor)
		if err := asegurarChecklist(tx, solicitud.ID); err != nil {
			return err
		}

		// 6. Crear automáticamente el hilo de mensajería vendedor ↔ comprador
		//    (así ambos se ven en Consultas aunque nunca hayan chateado)
		propiedadID := prop.ID
		mensajeBienvenida := "🎉 ¡Venta concretada para \"" + prop.Titulo +
			"\"! Por este chat puedes consultarme sobre el plan de pagos o cualquier duda."
		if err := tx.Create(&models.Message{
			DeUserID:    userID,
			ParaUserID:  solicitud.CompradorID,
			PropertyID:  &propiedadID,
			Mensaje:     mensajeBienvenida,
			TipoMensaje: "texto",
			Leido:       false,
		}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error aceptando la solicitud: " + err.Error(),
		})
	}

	// Recargar con relaciones
	h.DB.Preload("Property").Preload("Comprador").Preload("Pagos").First(&solicitud, solicitud.ID)

	// 🔔 Notificar al comprador
	notificacion := models.Notification{
		UserID:     solicitud.CompradorID,
		PropertyID: &prop.ID,
		Tipo:       "compra_aceptada",
		Titulo:     "¡Tu oferta de compra fue aceptada!",
		Mensaje:    "El propietario aceptó tu solicitud para \"" + prop.Titulo + "\" por " + fmt.Sprintf("%.2f", montoAceptado) + ". Revisa el plan de pagos en Mis Compras",
		IsRead:     false,
	}
	h.DB.Create(&notificacion)

	return c.JSON(fiber.Map{
		"message":        "¡Venta concretada! La propiedad está marcada como vendida",
		"solicitud":      solicitud,
		"monto_aceptado": montoAceptado,
		"pagos":          len(pagos),
	})
}

// RejectPurchaseRequest - El vendedor rechaza la solicitud de compra
// PATCH /api/panel/purchase-requests/:id/reject
func (h *PurchaseRequestHandler) RejectPurchaseRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	requestID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de solicitud inválido",
		})
	}

	var solicitud models.PurchaseRequest
	if err := h.DB.Where("id = ? AND vendedor_id = ? AND deleted_at IS NULL", requestID, userID).
		Preload("Property").
		First(&solicitud).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Solicitud no encontrada",
		})
	}

	if solicitud.Estado != models.PurchaseRequestStatusPendiente {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "La solicitud ya fue procesada (estado: " + string(solicitud.Estado) + ")",
		})
	}

	if err := h.DB.Model(&solicitud).Update("estado", models.PurchaseRequestStatusRechazada).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error rechazando la solicitud",
		})
	}

	// 🔔 Notificar al comprador
	notificacion := models.Notification{
		UserID:     solicitud.CompradorID,
		PropertyID: &solicitud.PropertyID,
		Tipo:       "compra_rechazada",
		Titulo:     "Solicitud de compra rechazada",
		Mensaje:    "El propietario de \"" + solicitud.Property.Titulo + "\" rechazó tu solicitud de compra",
		IsRead:     false,
	}
	h.DB.Create(&notificacion)

	return c.JSON(fiber.Map{
		"message": "Solicitud de compra rechazada",
	})
}

// AnularVentaRequest - Body para anular una venta
type AnularVentaRequest struct {
	Motivo string `json:"motivo"`
}

// AnularVenta - El vendedor anula una venta aceptada (p. ej. por fraude):
// la solicitud queda "anulada" con su motivo, la propiedad vuelve a estar
// disponible y los pagos pendientes (no pagados) se eliminan. Los pagos con
// comprobante o confirmados se conservan como historial/evidencia.
// PATCH /api/panel/purchase-requests/:id/anular
func (h *PurchaseRequestHandler) AnularVenta(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	requestID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de solicitud inválido",
		})
	}

	var req AnularVentaRequest
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Error parseando request",
			})
		}
	}
	motivo := strings.TrimSpace(req.Motivo)
	if motivo == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "El motivo de la anulación es requerido",
		})
	}
	if len(motivo) > 300 {
		motivo = motivo[:300]
	}

	var solicitud models.PurchaseRequest
	if err := h.DB.Where("id = ? AND vendedor_id = ? AND deleted_at IS NULL", requestID, userID).
		Preload("Property").
		Preload("Comprador").
		First(&solicitud).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Solicitud no encontrada",
		})
	}

	if solicitud.Estado != models.PurchaseRequestStatusAceptada {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Solo se puede anular una venta aceptada (estado: " + string(solicitud.Estado) + ")",
		})
	}

	prop := solicitud.Property
	fechaAnulada := time.Now()

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Marcar la venta como anulada con su motivo
		if err := tx.Model(&models.PurchaseRequest{}).Where("id = ?", solicitud.ID).
			Updates(map[string]interface{}{
				"estado":           models.PurchaseRequestStatusAnulada,
				"motivo_anulacion": motivo,
				"fecha_anulada":    fechaAnulada,
			}).Error; err != nil {
			return err
		}

		// 2. La propiedad vuelve a estar disponible (si sigue marcada vendida)
		if err := tx.Model(&models.Property{}).
			Where("id = ? AND estado_propiedad = ?", prop.ID, models.PropertyStatusVendido).
			Update("estado_propiedad", models.PropertyStatusActivo).Error; err != nil {
			return err
		}

		// 3. Los pagos aún no pagados se eliminan (nadie debe poder pagarlos);
		//    los que ya tienen comprobante o están confirmados se conservan como evidencia
		if err := tx.Where("purchase_request_id = ? AND estado = ? AND deleted_at IS NULL",
			solicitud.ID, models.SalePaymentStatusPendiente).
			Delete(&models.SalePayment{}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error anulando la venta: " + err.Error(),
		})
	}

	// 🔔 Notificar al comprador
	notificacion := models.Notification{
		UserID:     solicitud.CompradorID,
		PropertyID: &prop.ID,
		Tipo:       "venta_anulada",
		Titulo:     "Venta anulada",
		Mensaje:    "El propietario anuló la venta de \"" + prop.Titulo + "\". Motivo: " + motivo,
		IsRead:     false,
	}
	h.DB.Create(&notificacion)

	return c.JSON(fiber.Map{
		"message": "Venta anulada. La propiedad volvió a estar disponible",
	})
}

// CancelPurchaseRequest - El comprador cancela su propia solicitud pendiente
// PATCH /api/purchase-requests/:id/cancel
func (h *PurchaseRequestHandler) CancelPurchaseRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	requestID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de solicitud inválido",
		})
	}

	var solicitud models.PurchaseRequest
	if err := h.DB.Where("id = ? AND comprador_id = ? AND deleted_at IS NULL", requestID, userID).
		Preload("Property").
		First(&solicitud).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Solicitud no encontrada",
		})
	}

	if solicitud.Estado != models.PurchaseRequestStatusPendiente {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "La solicitud ya fue procesada (estado: " + string(solicitud.Estado) + ")",
		})
	}

	if err := h.DB.Model(&solicitud).Update("estado", "cancelada").Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error cancelando la solicitud",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Solicitud cancelada",
	})
}

// SubmitSalePaymentComprobante - El comprador registra su pago: método y foto del
// comprobante (el dinero circula directo hacia el vendedor; el sistema lo registra)
// PATCH /api/purchase-payments/:id/comprobante
func (h *PurchaseRequestHandler) SubmitSalePaymentComprobante(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	paymentID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de pago inválido",
		})
	}

	var req struct {
		MetodoPago     string `json:"metodo_pago"`
		ComprobanteURL string `json:"comprobante_url"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	if strings.TrimSpace(req.MetodoPago) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "metodo_pago es requerido (qr, transferencia o efectivo)",
		})
	}
	if strings.TrimSpace(req.ComprobanteURL) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "comprobante_url es requerido",
		})
	}

	// El pago debe pertenecer al comprador autenticado
	var pago models.SalePayment
	if err := h.DB.Where("id = ? AND comprador_id = ?", paymentID, userID).
		Preload("PurchaseRequest").
		First(&pago).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Pago no encontrado",
		})
	}

	if pago.Estado == models.SalePaymentStatusConfirmado {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Este pago ya fue confirmado por el vendedor",
		})
	}

	ahora := time.Now()
	pago.MetodoPago = strings.TrimSpace(req.MetodoPago)
	pago.ComprobanteURL = strings.TrimSpace(req.ComprobanteURL)
	pago.FechaPago = &ahora
	pago.FechaComprobante = &ahora
	pago.Estado = models.SalePaymentStatusComprobante

	if err := h.DB.Save(&pago).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error guardando el comprobante",
		})
	}

	// 🔔 Notificar al vendedor que debe confirmar
	tituloPago := "Pago de venta"
	if pago.Tipo != "" {
		tituloPago = "Pago de venta (" + pago.Tipo + ")"
	}
	notificacion := models.Notification{
		UserID:     pago.VendedorID,
		PropertyID: &pago.PurchaseRequest.PropertyID,
		Tipo:       "pago_venta",
		Titulo:     tituloPago + " registrado por el comprador",
		Mensaje:    "El comprador registró un pago de " + fmt.Sprintf("%.2f", pago.Monto) + " por \"" + pago.PurchaseRequest.Property.Titulo + "\". Revísaló y confírmalo",
		IsRead:     false,
	}
	h.DB.Create(&notificacion)

	return c.JSON(fiber.Map{
		"message": "Comprobante registrado. El vendedor lo confirmará",
		"pago":    pago,
	})
}

// ConfirmSalePayment - El vendedor confirma la recepción del pago
// POST /api/panel/purchase-payments/:id/confirm
func (h *PurchaseRequestHandler) ConfirmSalePayment(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	paymentID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de pago inválido",
		})
	}

	var pago models.SalePayment
	if err := h.DB.Where("id = ? AND vendedor_id = ?", paymentID, userID).
		Preload("PurchaseRequest").
		Preload("PurchaseRequest.Property").
		First(&pago).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Pago no encontrado",
		})
	}

	if pago.Estado == models.SalePaymentStatusConfirmado {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Este pago ya está confirmado",
		})
	}
	if pago.Estado != models.SalePaymentStatusComprobante {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "El comprador aún no registró el comprobante de este pago",
		})
	}

	pago.Estado = models.SalePaymentStatusConfirmado
	ahora := time.Now()
	pago.FechaPago = &ahora
	if err := h.DB.Save(&pago).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error confirmando el pago",
		})
	}

	// 🔔 Notificar al comprador
	notificacion := models.Notification{
		UserID:     pago.CompradorID,
		PropertyID: &pago.PurchaseRequest.PropertyID,
		Tipo:       "pago_confirmado",
		Titulo:     "Pago confirmado por el vendedor",
		Mensaje:    "El vendedor confirmó la recepción de " + fmt.Sprintf("%.2f", pago.Monto) + " por \"" + pago.PurchaseRequest.Property.Titulo + "\"",
		IsRead:     false,
	}
	h.DB.Create(&notificacion)

	return c.JSON(fiber.Map{
		"message": "Pago confirmado",
		"pago":    pago,
	})
}

// RejectSalePaymentComprobante - El vendedor rechaza el comprobante enviado por
// el comprador: el pago vuelve a estado pendiente (sin comprobante) y el
// comprador es notificado para que vuelva a subirlo.
// PATCH /api/panel/purchase-payments/:id/reject-comprobante
func (h *PurchaseRequestHandler) RejectSalePaymentComprobante(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	paymentID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de pago inválido",
		})
	}

	var pago models.SalePayment
	if err := h.DB.Where("id = ? AND vendedor_id = ?", paymentID, userID).
		Preload("PurchaseRequest").
		Preload("PurchaseRequest.Property").
		First(&pago).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Pago no encontrado",
		})
	}

	if pago.Estado != models.SalePaymentStatusComprobante {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Solo se puede rechazar un pago con comprobante pendiente de confirmación",
		})
	}

	// Volver a pendiente y limpiar el comprobante enviado
	pago.MetodoPago = ""
	pago.ComprobanteURL = ""
	pago.FechaPago = nil
	pago.FechaComprobante = nil
	pago.Estado = models.SalePaymentStatusPendiente

	if err := h.DB.Save(&pago).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error rechazando el comprobante",
		})
	}

	// 🔔 Notificar al comprador
	notificacion := models.Notification{
		UserID:     pago.CompradorID,
		PropertyID: &pago.PurchaseRequest.PropertyID,
		Tipo:       "comprobante_rechazado",
		Titulo:     "Comprobante rechazado",
		Mensaje:    "El vendedor rechazó tu comprobante de " + fmt.Sprintf("%.2f", pago.Monto) + " por \"" + pago.PurchaseRequest.Property.Titulo + "\". Por favor, vuelve a subirlo",
		IsRead:     false,
	}
	h.DB.Create(&notificacion)

	return c.JSON(fiber.Map{
		"message": "Comprobante rechazado. El pago volvió a estado pendiente",
		"pago":    pago,
	})
}

// GetMinutaPDF - Genera y descarga la Minuta/Promesa de Compraventa en PDF de
// una venta aceptada. Accesible por el vendedor y por el comprador.
// GET /api/purchase-requests/:id/minuta
func (h *PurchaseRequestHandler) GetMinutaPDF(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	requestID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de solicitud inválido",
		})
	}

	var solicitud models.PurchaseRequest
	if err := h.DB.Where("id = ? AND (comprador_id = ? OR vendedor_id = ?) AND deleted_at IS NULL",
		requestID, userID, userID).
		Preload("Property").
		Preload("Comprador").
		Preload("Vendedor").
		Preload("Pagos", func(db *gorm.DB) *gorm.DB {
			return db.Order("id ASC")
		}).
		First(&solicitud).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Solicitud no encontrada",
		})
	}

	if solicitud.Estado != models.PurchaseRequestStatusAceptada {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "La minuta se genera solo para ventas aceptadas",
		})
	}

	pdfBytes, err := generarPDFMinuta(&solicitud)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error generando la minuta: " + err.Error(),
		})
	}

	filename := fmt.Sprintf("minuta-venta-%d.pdf", solicitud.ID)
	c.Set("Content-Type", "application/pdf")
	// Por defecto se muestra inline (visor del navegador); con ?download=1 se descarga
	if c.Query("download") == "1" {
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	} else {
		c.Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filename))
	}
	return c.Send(pdfBytes)
}

// asegurarChecklist - Crea las filas del checklist documental (estado pendiente)
// para una venta, si aún no existen. Idempotente: se puede invocar en la
// transacción de aceptación y también perezosamente al consultar el checklist
// de ventas aceptadas con anterioridad a esta funcionalidad.
func asegurarChecklist(tx *gorm.DB, purchaseRequestID uint) error {
	for _, tipo := range models.OrdenTiposDocumentoVenta {
		var count int64
		if err := tx.Model(&models.SaleDocument{}).
			Where("purchase_request_id = ? AND tipo = ? AND deleted_at IS NULL",
				purchaseRequestID, tipo).
			Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.Create(&models.SaleDocument{
				PurchaseRequestID: purchaseRequestID,
				Tipo:              tipo,
				Estado:            models.SaleDocPendiente,
			}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// GetChecklistVenta - Checklist documental de la compraventa (4 documentos del
// vendedor). El comprador y el vendedor pueden consultarlo; al vendedor se le
// informa además cuántos documentos ya fueron verificados.
// GET /api/purchase-requests/:id/documentos  (comprador o vendedor)
// GET /api/panel/purchase-requests/:id/documentos  (vendedor)
func (h *PurchaseRequestHandler) GetChecklistVenta(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	requestID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de solicitud inválido",
		})
	}

	var solicitud models.PurchaseRequest
	if err := h.DB.Where("id = ? AND (comprador_id = ? OR vendedor_id = ?) AND deleted_at IS NULL",
		requestID, userID, userID).First(&solicitud).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Solicitud no encontrada",
		})
	}

	if solicitud.Estado != models.PurchaseRequestStatusAceptada {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "El checklist documental se genera al aceptarse la compra",
		})
	}

	// Completar el checklist si la venta es anterior a esta funcionalidad
	if err := asegurarChecklist(h.DB, solicitud.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error preparando el checklist documental",
		})
	}

	var documentos []models.SaleDocument
	if err := h.DB.Where("purchase_request_id = ? AND deleted_at IS NULL", solicitud.ID).
		Order("id ASC").
		Find(&documentos).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo el checklist documental",
		})
	}

	verificados := 0
	for _, doc := range documentos {
		if doc.Estado == models.SaleDocVerificado {
			verificados++
		}
	}

	return c.JSON(fiber.Map{
		"documentos":   documentos,
		"etiquetas":    models.TiposDocumentoVenta,
		"verificados":  verificados,
		"total":        len(documentos),
	})
}

// SubirDocumentoVenta - El vendedor adjunta el archivo de un documento del
// checklist (la imagen se sube antes vía /api/imagenes/upload).
// PATCH /api/panel/purchase-requests/:id/documentos/:docId
func (h *PurchaseRequestHandler) SubirDocumentoVenta(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	requestID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID de solicitud inválido"})
	}
	docID, err := c.ParamsInt("docId")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID de documento inválido"})
	}

	var req struct {
		ArchivoURL string `json:"archivo_url"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Error parseando request"})
	}
	if strings.TrimSpace(req.ArchivoURL) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "archivo_url es requerido"})
	}

	var solicitud models.PurchaseRequest
	if err := h.DB.Where("id = ? AND vendedor_id = ? AND deleted_at IS NULL", requestID, userID).
		First(&solicitud).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Solicitud no encontrada"})
	}
	if solicitud.Estado != models.PurchaseRequestStatusAceptada {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "El checklist documental corresponde a ventas aceptadas",
		})
	}

	var doc models.SaleDocument
	if err := h.DB.Where("id = ? AND purchase_request_id = ? AND deleted_at IS NULL",
		docID, solicitud.ID).First(&doc).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Documento no encontrado"})
	}

	doc.ArchivoURL = strings.TrimSpace(req.ArchivoURL)
	if doc.Estado != models.SaleDocVerificado {
		doc.Estado = models.SaleDocSubido
	}
	if err := h.DB.Save(&doc).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error guardando el documento",
		})
	}

	return c.JSON(fiber.Map{
		"message":   "Documento subido. Márcalo como verificado cuando valides su contenido",
		"documento": doc,
	})
}

// VerificarDocumentoVenta - El vendedor marca como verificado un documento que
// ya tiene archivo. Notifica al comprador del avance.
// POST /api/panel/purchase-requests/:id/documentos/:docId/verificar
func (h *PurchaseRequestHandler) VerificarDocumentoVenta(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	requestID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID de solicitud inválido"})
	}
	docID, err := c.ParamsInt("docId")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID de documento inválido"})
	}

	var solicitud models.PurchaseRequest
	if err := h.DB.Where("id = ? AND vendedor_id = ? AND deleted_at IS NULL", requestID, userID).
		Preload("Property").
		First(&solicitud).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Solicitud no encontrada"})
	}
	if solicitud.Estado != models.PurchaseRequestStatusAceptada {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "El checklist documental corresponde a ventas aceptadas",
		})
	}

	var doc models.SaleDocument
	if err := h.DB.Where("id = ? AND purchase_request_id = ? AND deleted_at IS NULL",
		docID, solicitud.ID).First(&doc).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Documento no encontrado"})
	}

	if doc.ArchivoURL == "" {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Primero sube el archivo de este documento",
		})
	}
	if doc.Estado == models.SaleDocVerificado {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Este documento ya está verificado",
		})
	}

	doc.Estado = models.SaleDocVerificado
	if err := h.DB.Save(&doc).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error verificando el documento",
		})
	}

	// Avance completo: notificar al comprador que la documentación está completa
	var pendientes int64
	h.DB.Model(&models.SaleDocument{}).
		Where("purchase_request_id = ? AND estado != ? AND deleted_at IS NULL",
			solicitud.ID, models.SaleDocVerificado).
		Count(&pendientes)
	if pendientes == 0 {
		notificacion := models.Notification{
			UserID:     solicitud.CompradorID,
			PropertyID: &solicitud.PropertyID,
			Tipo:       "documentacion_venta",
			Titulo:     "Documentación de la venta completa",
			Mensaje:    "El vendedor verificó todos los documentos de \"" + solicitud.Property.Titulo + "\". Ya puedes coordinar la escritura pública",
			IsRead:     false,
		}
		h.DB.Create(&notificacion)
	}

	return c.JSON(fiber.Map{
		"message":   "Documento verificado",
		"documento": doc,
	})
}
