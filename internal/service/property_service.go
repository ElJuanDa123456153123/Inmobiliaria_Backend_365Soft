package service

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/repository"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/errors"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// PropertyService - Servicio de lógica de negocio para propiedades
type PropertyService struct {
	PropertyRepo     repository.PropertyRepositoryInterface
	ImageRepo        repository.PropertyImageRepositoryInterface
	UserRepo         repository.UserRepositoryInterface
	AuditRepo        repository.AuditRepositoryInterface
	AssignmentService *AssignmentService
	db               *gorm.DB
}

// NewPropertyService - Crea un nuevo servicio de propiedades
func NewPropertyService(
	propertyRepo repository.PropertyRepositoryInterface,
	imageRepo repository.PropertyImageRepositoryInterface,
	userRepo repository.UserRepositoryInterface,
	auditRepo repository.AuditRepositoryInterface,
	db *gorm.DB,
	assignmentService *AssignmentService,
) *PropertyService {
	return &PropertyService{
		PropertyRepo:     propertyRepo,
		ImageRepo:        imageRepo,
		UserRepo:         userRepo,
		AuditRepo:        auditRepo,
		db:               db,
		AssignmentService: assignmentService,
	}
}

// CreatePropertyRequest - Estructura para crear propiedad
type CreatePropertyRequest struct {
	UserID         uint
	Titulo         string
	TipoOperacion  string
	Precio         float64
	Moneda         string
	AceptaPermuta  bool
	Financia       bool
	// Campos específicos para ALQUILER
	Deposito           string
	PlazoMinimo        string
	ServiciosIncluidos []string
	// Campos específicos para ANTICRÉTICO
	DuracionContrato string
	OpcionCompra     bool
	TipoPropiedad  string
	Direccion      string
	Zona           string
	TipoMarcador   string
	Latitud        *float64
	Longitud       *float64
	MostrarDireccion bool
	Dormitorios     string
	Banos           string
	Garaje          string
	MetrosEdificados float64
	MetrosTerreno   float64
	Estado          string
	AnoConstruccion int
	AptoOficina     bool
	ViviendaSocial  bool
	Condominio      bool
	CantidadPisos   int
	Confort        []string
	Seguridad      []string
	NoAplicaConfort bool
	Imagenes       []CreatePropertyImageRequest
	VideoURL        string
	Descripcion     string
	TerminosCondiciones string
	AdvisorID       *uint // Campo para admin
}

// CreatePropertyImageRequest - Estructura para crear imagen de propiedad
type CreatePropertyImageRequest struct {
	URL   string
	Orden int
}

// CreateProperty - Crea una nueva propiedad con toda la lógica de negocio
func (s *PropertyService) CreateProperty(req *CreatePropertyRequest) (*models.Property, error) {
	// 🔒 Validaciones de negocio
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Obtener usuario para guardar datos del vendedor
	// Si el userID es 0, es el admin harcodeado (no está en BD)
	var user *models.User
	if req.UserID == 0 {
		// Admin harcodeado
		user = &models.User{
			ID:         0,
			Nombre:     "Administrador",
			Apellido:   "Sistema",
			Email:      "admin@365soft.com",
			Rol:        models.RoleAdmin,
			Activo:     true,
			Verificado:  true,
		}
	} else {
		// Buscar usuario en la base de datos
		var err error
		user, err = s.UserRepo.GetByID(req.UserID)
		if err != nil {
			return nil, errors.UserNotFound(req.UserID)
		}
	}

	// 🔒 Sanitización HTML (se hace en el handler, pero podría hacerse aquí también)
	// Crear propiedad
	now := time.Now()

	// Calcular fecha de expiración basada en la duración del plan
	var fechaExpiracion *time.Time
	if req.UserID != 0 {
		// Buscar suscripción activa del usuario
		var subscription models.Subscription
		if err := s.db.Where("user_id = ? AND estado = ? AND seleccionada = ?", req.UserID, "ACTIVA", true).
			Preload("Plan").
			First(&subscription).Error; err == nil {
			// Usar la duración del plan para calcular la expiración de la propiedad
			if subscription.Plan != nil && subscription.Plan.DuracionPropiedadMinutos > 0 {
				fechaExpiracionCalc := now.Add(time.Duration(subscription.Plan.DuracionPropiedadMinutos) * time.Minute)
				fechaExpiracion = &fechaExpiracionCalc
				fmt.Printf("⏰ Propiedad expirará en %d minutos (plan: %s): %s\n",
					subscription.Plan.DuracionPropiedadMinutos,
					subscription.Plan.Nombre,
					fechaExpiracionCalc.Format("2006-01-02 15:04:05"))
			}
		}
	}

	// Estado inicial: SIEMPRE ACTIVO (aprobación automática)
	// Los usuarios pueden activar/desactivar sus propiedades manualmente
	estadoInicial := models.PropertyStatusActivo
	fechaPublicacion := &now
	fmt.Printf("✅ Propiedad creada con estado ACTIVO (aprobación automática)\n")

	property := &models.Property{
		UserID:           req.UserID,
		Titulo:           req.Titulo,
		TipoOperacion:    models.OperationType(req.TipoOperacion),
		Precio:           req.Precio,
		Moneda:           req.Moneda,
		AceptaPermuta:    req.AceptaPermuta,
		Financia:         req.Financia,
		// Campos específicos para alquiler/anticrético
		Deposito:           req.Deposito,
		PlazoMinimo:        req.PlazoMinimo,
		DuracionContrato:   req.DuracionContrato,
		OpcionCompra:       req.OpcionCompra,
		TipoPropiedad:    models.PropertyType(req.TipoPropiedad),
		Direccion:        req.Direccion,
		Zona:             req.Zona,
		TipoMarcador:     req.TipoMarcador,
		Latitud:          req.Latitud,
		Longitud:         req.Longitud,
		MostrarDireccion: req.MostrarDireccion,
		Dormitorios:      req.Dormitorios,
		Banos:            req.Banos,
		Garaje:           req.Garaje,
		MetrosEdificados: req.MetrosEdificados,
		MetrosTerreno:    req.MetrosTerreno,
		Estado:           req.Estado,
		AnoConstruccion:  req.AnoConstruccion,
		AptoOficina:      req.AptoOficina,
		ViviendaSocial:   req.ViviendaSocial,
		Condominio:       req.Condominio,
		CantidadPisos:    req.CantidadPisos,
		NoAplicaConfort:  req.NoAplicaConfort,
		VideoURL:         req.VideoURL,
		Descripcion:      req.Descripcion,
		TerminosCondiciones: req.TerminosCondiciones,
		EstadoPropiedad:     estadoInicial,
		Destacado:           false,
		VistasContador:      0,
		FechaPublicacion:    fechaPublicacion,
		FechaExpiracion:     fechaExpiracion, // Fecha en que se eliminará automáticamente
		AdvisorID:           req.AdvisorID,
	}

	// Guardar snapshot de datos del vendedor
	property.VendedorNombre = fmt.Sprintf("%s %s", user.Nombre, user.Apellido)
	property.VendedorEmail = user.Email
	property.VendedorTelefono = user.Telefono
	property.VendedorContacto = user.NombreContacto
	property.VendedorHorario = user.HorarioContacto

	// Serializar arrays JSON
	if req.Confort != nil {
		confortBytes, _ := json.Marshal(req.Confort)
		property.Confort = datatypes.JSON(confortBytes)
	}
	if req.Seguridad != nil {
		seguridadBytes, _ := json.Marshal(req.Seguridad)
		property.Seguridad = datatypes.JSON(seguridadBytes)
	}
	if req.ServiciosIncluidos != nil {
		serviciosBytes, _ := json.Marshal(req.ServiciosIncluidos)
		property.ServiciosIncluidos = datatypes.JSON(serviciosBytes)
	}

	// Guardar propiedad
	if err := s.PropertyRepo.Create(property); err != nil {
		return nil, errors.DatabaseError(err)
	}

	// 🔄 ASIGNACIÓN DE ASESOR
	// Las propiedades se publican ACTIVAS automáticamente
	// El admin/asesor puede asignar un asesor después (similar a comprobantes de pago)
	// if s.AssignmentService != nil && user.Rol == models.RoleUsuario && req.UserID != 0 {
	//	if err := s.AssignmentService.AssignToLeastLoadedAdvisor(property, user.Rol); err != nil {
	//		fmt.Printf("⚠️  Error en asignación automática: %v\n", err)
	//		// No fallar la creación de propiedad si falla la asignación
	//	}
	// }
	fmt.Printf("✅ Propiedad ACTIVA publicada (sin asignación automática de asesor)\n")

	// Incrementar contador de propiedades usadas en la suscripción (si no es admin)
	if req.UserID != 0 {
		if err := s.db.Model(&models.Subscription{}).
			Where("user_id = ? AND estado = ?", req.UserID, "ACTIVA").
			UpdateColumn("propiedades_usadas", gorm.Expr("propiedades_usadas + ?", 1)).
			Error; err != nil {
			fmt.Printf("⚠️  Error incrementando contador de propiedades: %v\n", err)
			// No fallar la creación de la propiedad si falla el contador
		}
	}

	// Guardar imágenes
	if len(req.Imagenes) > 0 {
		images := make([]models.PropertyImage, len(req.Imagenes))
		for i, imgReq := range req.Imagenes {
			images[i] = models.PropertyImage{
				PropertyID: property.ID,
				ImageURL:   imgReq.URL,
				Orden:      imgReq.Orden,
				Principal:  imgReq.Orden == 0,
			}
		}

		if err := s.ImageRepo.CreateBatch(images); err != nil {
			// No fallar si falla la creación de imágenes
			fmt.Printf("Error creando imágenes: %v\n", err)
		}
	}

	return property, nil
}

// GetPropertyByID - Obtiene una propiedad por ID
func (s *PropertyService) GetPropertyByID(id uint) (*models.Property, error) {
	property, err := s.PropertyRepo.GetByID(id)
	if err != nil {
		return nil, errors.PropertyNotFound(id)
	}

	return property, nil
}

// ListProperties - Lista propiedades con filtros y paginación
func (s *PropertyService) ListProperties(params repository.ListPropertyParams) ([]models.Property, int64, error) {
	properties, total, err := s.PropertyRepo.List(params)
	if err != nil {
		return nil, 0, errors.DatabaseError(err)
	}

	return properties, total, nil
}

// UpdatePropertyStatusRequest - Request para actualizar estado
type UpdatePropertyStatusRequest struct {
	PropertyID uint
	UserID     uint
	NewStatus  models.PropertyStatus
}

// UpdatePropertyStatus - Actualiza el estado de una propiedad
func (s *PropertyService) UpdatePropertyStatus(req *UpdatePropertyStatusRequest) (*models.Property, error) {
	// Obtener propiedad
	property, err := s.PropertyRepo.GetByID(req.PropertyID)
	if err != nil {
		return nil, errors.PropertyNotFound(req.PropertyID)
	}

	// Verificar propiedad pertenezca al usuario
	if property.UserID != req.UserID {
		return nil, errors.Forbidden("No tienes permiso para modificar esta propiedad")
	}

	// ✅ El dueño SIEMPRE puede editar sus propiedades (sin restricción de suscripción)
	// Esto es un derecho de propiedad sobre sus propias publicaciones

	// Validar estado
	validStatuses := map[models.PropertyStatus]bool{
		models.PropertyStatusActivo:    true,
		models.PropertyStatusInactivo:  true,
		models.PropertyStatusVendido:   true,
	}
	if !validStatuses[req.NewStatus] {
		return nil, errors.ValidationError("estado", "Estado inválido. Debe ser: activo, inactivo o vendido")
	}

	// Actualizar estado
	property.EstadoPropiedad = req.NewStatus
	if err := s.PropertyRepo.Update(property); err != nil {
		return nil, errors.DatabaseError(err)
	}

	return property, nil
}

// DeletePropertyRequest - Request para eliminar propiedad
type DeletePropertyRequest struct {
	PropertyID uint
	UserID     uint
}

// DeleteProperty - Elimina (soft delete) una propiedad
func (s *PropertyService) DeleteProperty(req *DeletePropertyRequest) error {
	// Obtener propiedad
	property, err := s.PropertyRepo.GetByID(req.PropertyID)
	if err != nil {
		return errors.PropertyNotFound(req.PropertyID)
	}

	// Verificar propiedad pertenezca al usuario
	if property.UserID != req.UserID {
		return errors.Forbidden("No tienes permiso para eliminar esta propiedad")
	}

	// ✅ El dueño SIEMPRE puede eliminar sus propiedades (sin restricción de suscripción)

	// Buscar la suscripción del usuario para decrementar el contador si el plan lo permite
	if req.UserID != 0 {
		var subscription models.Subscription
		err := s.db.Where("user_id = ?", req.UserID).First(&subscription).Error
		if err == nil {
			// Obtener el plan para verificar si permite republicación
			var plan models.Plan
			if err := s.db.First(&plan, subscription.PlanID).Error; err == nil {
				if plan.PermitirRepublicacion && subscription.PropiedadesUsadas > 0 {
					// Decrementar el contador
					s.db.Model(&subscription).
						Where("propiedades_usadas > 0").
						UpdateColumn("propiedades_usadas", gorm.Expr("propiedades_usadas - 1"))

					fmt.Printf("✅ [ELIMINAR] Contador decrementado: Plan '%s' permite republicación\n", plan.Nombre)
				} else {
					fmt.Printf("⚠️  [ELIMINAR] Contador NO decrementado: Plan '%s' NO permite republicación\n", plan.Nombre)
				}
			}
		}
	}

	// Soft delete
	if err := s.PropertyRepo.Delete(property); err != nil {
		return errors.DatabaseError(err)
	}

	return nil
}

// DeletePropertyAdmin - Elimina una propiedad (admin) - sin verificar dueño
func (s *PropertyService) DeletePropertyAdmin(propertyID uint) error {
	// Obtener propiedad con imágenes
	property, err := s.PropertyRepo.GetByIDWithRelations(propertyID)
	if err != nil {
		return errors.PropertyNotFound(propertyID)
	}

	// 🔒 Eliminar imágenes físicas antes de soft delete
	if len(property.PropertyImages) > 0 {
		for _, img := range property.PropertyImages {
			imagePath := fmt.Sprintf("uploads/imagenes/%s", extractFilenameFromURL(img.ImageURL))
			thumbnailPath := fmt.Sprintf("uploads/imagenes/%s", extractThumbnailFilenameFromURL(img.ImageURL))

			// Eliminar imagen principal
			if err := os.Remove(imagePath); err != nil && !os.IsNotExist(err) {
				fmt.Printf("⚠️  Error eliminando imagen %s: %v\n", imagePath, err)
			} else {
				fmt.Printf("✅ Imagen eliminada: %s\n", imagePath)
			}

			// Eliminar thumbnail
			if err := os.Remove(thumbnailPath); err != nil && !os.IsNotExist(err) {
				fmt.Printf("⚠️  Error eliminando thumbnail %s: %v\n", thumbnailPath, err)
			} else {
				fmt.Printf("✅ Thumbnail eliminado: %s\n", thumbnailPath)
			}
		}
	}

	// Soft delete sin verificar dueño (admin puede eliminar cualquier propiedad)
	if err := s.PropertyRepo.Delete(property); err != nil {
		return errors.DatabaseError(err)
	}

	return nil
}

// GetUserProperties - Obtiene todas las propiedades de un usuario
func (s *PropertyService) GetUserProperties(userID uint) ([]models.Property, error) {
	properties, err := s.PropertyRepo.GetByUserID(userID)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	return properties, nil
}

// FullTextSearchRequest - Request para búsqueda full-text
type FullTextSearchRequest struct {
	Query          string
	TipoOperacion  string
	TipoPropiedad  string
	Moneda         string
	PrecioMin      float64
	PrecioMax      float64
	MetrosMin      float64
	MetrosMax      float64
	Page           int
	Limit          int
}

// FullTextSearch - Búsqueda full-text en propiedades
func (s *PropertyService) FullTextSearch(req *FullTextSearchRequest) ([]models.Property, int64, error) {
	// Validar query
	if len(req.Query) < 2 {
		return []models.Property{}, 0, nil // Query muy corto, no hay resultados
	}

	// Validar página y límite
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20
	}

	offset := (req.Page - 1) * req.Limit

	// Si hay filtros adicionales, usar búsqueda con filtros
	if req.TipoOperacion != "" || req.TipoPropiedad != "" || req.Moneda != "" ||
		req.PrecioMin > 0 || req.PrecioMax > 0 || req.MetrosMin > 0 || req.MetrosMax > 0 {

		filters := repository.ListSearchParams{
			TipoOperacion: req.TipoOperacion,
			TipoPropiedad: req.TipoPropiedad,
			Moneda:       req.Moneda,
			PrecioMin:    req.PrecioMin,
			PrecioMax:    req.PrecioMax,
			MetrosMin:    req.MetrosMin,
			MetrosMax:    req.MetrosMax,
		}

		return s.PropertyRepo.FullTextSearchWithFilters(req.Query, filters, req.Limit, offset)
	}

	// Búsqueda simple sin filtros adicionales
	return s.PropertyRepo.FullTextSearch(req.Query, req.Limit, offset)
}

// IncrementViewCount - Incrementa el contador de vistas
func (s *PropertyService) IncrementViewCount(propertyID uint) error {
	return s.PropertyRepo.IncrementViewCount(propertyID)
}

// validateCreateRequest - Valida el request de creación
func (s *PropertyService) validateCreateRequest(req *CreatePropertyRequest) error {
	// Validar título
	if len(req.Titulo) < 10 {
		return errors.ValidationError("titulo", "El título debe tener al menos 10 caracteres")
	}

	// Validar tipo de operación
	validOps := map[string]bool{
		"venta":       true,
		"alquiler":    true,
		"anticretico": true,
	}
	if !validOps[req.TipoOperacion] {
		return errors.ValidationError("tipo_operacion", "Tipo de operación inválido. Debe ser: venta, alquiler o anticretico")
	}

	// Validar moneda
	if req.Moneda != "USD" && req.Moneda != "BOB" {
		return errors.ValidationError("moneda", "Moneda inválida. Debe ser: USD o BOB")
	}

	// Validar precio según moneda Y tipo de operación
	if err := ValidarRangoPrecio(req.Precio, req.Moneda, req.TipoOperacion); err != nil {
		return err
	}

	// Validar coordenadas (Bolivia)
	if err := ValidarCoordenadasBolivia(req.Latitud, req.Longitud); err != nil {
		return err
	}

	// Validar metros edificados
	if req.MetrosEdificados <= 0 {
		return errors.ValidationError("metros_edificados", "Los metros edificados deben ser mayores a 0")
	}

	// Validar estado
	if req.Estado == "" {
		return errors.ValidationError("estado", "El estado es requerido")
	}

	// Validar descripción
	if len(req.Descripcion) < 20 {
		return errors.ValidationError("descripcion", "La descripción debe tener al menos 20 caracteres")
	}

	// Validar límite de imágenes
	const MAX_IMAGES = 30
	if len(req.Imagenes) > MAX_IMAGES {
		return errors.LimitExceeded(fmt.Sprintf("Máximo %d imágenes por propiedad", MAX_IMAGES))
	}

	return nil
}

// ==================== MÉTODOS PARA ADMIN ====================

// CreatePropertyAsAdmin - Crea una propiedad como administrador (directamente activa)
func (s *PropertyService) CreatePropertyAsAdmin(req *CreatePropertyRequest) (*models.Property, error) {
	// 🔒 Validaciones de negocio
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Obtener usuario para guardar datos del vendedor
	user, err := s.UserRepo.GetByID(req.UserID)
	if err != nil {
		return nil, errors.UserNotFound(req.UserID)
	}

	// Crear propiedad como admin (directamente activa)
	now := time.Now()
	property := &models.Property{
		UserID:           req.UserID,
		Titulo:           req.Titulo,
		TipoOperacion:    models.OperationType(req.TipoOperacion),
		Precio:           req.Precio,
		Moneda:           req.Moneda,
		AceptaPermuta:    req.AceptaPermuta,
		Financia:         req.Financia,
		TipoPropiedad:    models.PropertyType(req.TipoPropiedad),
		Direccion:        req.Direccion,
		Zona:             req.Zona,
		TipoMarcador:     req.TipoMarcador,
		Latitud:          req.Latitud,
		Longitud:         req.Longitud,
		MostrarDireccion: req.MostrarDireccion,
		Dormitorios:      req.Dormitorios,
		Banos:            req.Banos,
		Garaje:           req.Garaje,
		MetrosEdificados: req.MetrosEdificados,
		MetrosTerreno:    req.MetrosTerreno,
		Estado:           req.Estado,
		AnoConstruccion:  req.AnoConstruccion,
		AptoOficina:      req.AptoOficina,
		ViviendaSocial:   req.ViviendaSocial,
		Condominio:       req.Condominio,
		CantidadPisos:    req.CantidadPisos,
		NoAplicaConfort:  req.NoAplicaConfort,
		VideoURL:         req.VideoURL,
		Descripcion:      req.Descripcion,
		TerminosCondiciones: req.TerminosCondiciones,
		EstadoPropiedad:     models.PropertyStatusActivo, // Siempre activa para admin
		Destacado:           false,
		VistasContador:      0,
		FechaPublicacion:    &now,
		AdvisorID:           req.AdvisorID,
	}

	// Guardar snapshot de datos del vendedor
	property.VendedorNombre = fmt.Sprintf("%s %s", user.Nombre, user.Apellido)
	property.VendedorEmail = user.Email
	property.VendedorTelefono = user.Telefono
	property.VendedorContacto = user.NombreContacto
	property.VendedorHorario = user.HorarioContacto

	// Serializar arrays JSON
	if req.Confort != nil {
		confortBytes, _ := json.Marshal(req.Confort)
		property.Confort = datatypes.JSON(confortBytes)
	}
	if req.Seguridad != nil {
		seguridadBytes, _ := json.Marshal(req.Seguridad)
		property.Seguridad = datatypes.JSON(seguridadBytes)
	}
	if req.ServiciosIncluidos != nil {
		serviciosBytes, _ := json.Marshal(req.ServiciosIncluidos)
		property.ServiciosIncluidos = datatypes.JSON(serviciosBytes)
	}

	// Guardar propiedad
	if err := s.PropertyRepo.Create(property); err != nil {
		return nil, errors.DatabaseError(err)
	}

	// Guardar imágenes
	if len(req.Imagenes) > 0 {
		images := make([]models.PropertyImage, len(req.Imagenes))
		for i, imgReq := range req.Imagenes {
			images[i] = models.PropertyImage{
				PropertyID: property.ID,
				ImageURL:   imgReq.URL,
				Orden:      imgReq.Orden,
				Principal:  imgReq.Orden == 0,
			}
		}

		if err := s.ImageRepo.CreateBatch(images); err != nil {
			fmt.Printf("Error creando imágenes: %v\n", err)
		}
	}

	return property, nil
}

// GetPropertyByIDWithRelations - Obtiene una propiedad con todas sus relaciones precargadas
func (s *PropertyService) GetPropertyByIDWithRelations(id uint) (*models.Property, error) {
	property, err := s.PropertyRepo.GetByIDWithRelations(id)
	if err != nil {
		return nil, errors.PropertyNotFound(id)
	}

	return property, nil
}

// UpdateProperty - Actualiza campos específicos de una propiedad (para admin)
func (s *PropertyService) UpdateProperty(id uint, updates map[string]interface{}) (*models.Property, error) {
	// Obtener propiedad
	property, err := s.PropertyRepo.GetByID(id)
	if err != nil {
		return nil, errors.PropertyNotFound(id)
	}

	// Actualizar campos proporcionados
	if titulo, ok := updates["titulo"].(string); ok {
		property.Titulo = titulo
	}
	if tipoOperacion, ok := updates["tipo_operacion"].(string); ok {
		property.TipoOperacion = models.OperationType(tipoOperacion)
	}
	if precio, ok := updates["precio"].(float64); ok {
		property.Precio = precio
	}
	if moneda, ok := updates["moneda"].(string); ok {
		property.Moneda = moneda
	}
	if aceptaPermuta, ok := updates["acepta_permuta"].(bool); ok {
		property.AceptaPermuta = aceptaPermuta
	}
	if financia, ok := updates["financia"].(bool); ok {
		property.Financia = financia
	}
	if tipoPropiedad, ok := updates["tipo_propiedad"].(string); ok {
		property.TipoPropiedad = models.PropertyType(tipoPropiedad)
	}
	if direccion, ok := updates["direccion"].(string); ok {
		property.Direccion = direccion
	}
	if zona, ok := updates["zona"].(string); ok {
		property.Zona = zona
	}
	if tipoMarcador, ok := updates["tipo_marcador"].(string); ok {
		property.TipoMarcador = tipoMarcador
	}
	if latitud, ok := updates["latitud"].(*float64); ok {
		property.Latitud = latitud
	}
	if longitud, ok := updates["longitud"].(*float64); ok {
		property.Longitud = longitud
	}
	if mostrarDireccion, ok := updates["mostrar_direccion"].(bool); ok {
		property.MostrarDireccion = mostrarDireccion
	}
	if dormitorios, ok := updates["dormitorios"].(string); ok {
		property.Dormitorios = dormitorios
	}
	if banos, ok := updates["banos"].(string); ok {
		property.Banos = banos
	}
	if garaje, ok := updates["garaje"].(string); ok {
		property.Garaje = garaje
	}
	if metrosEdificados, ok := updates["metros_edificados"].(float64); ok {
		property.MetrosEdificados = metrosEdificados
	}
	if metrosTerreno, ok := updates["metros_terreno"].(float64); ok {
		property.MetrosTerreno = metrosTerreno
	}
	if estado, ok := updates["estado"].(string); ok {
		property.Estado = estado
	}
	if anoConstruccion, ok := updates["ano_construccion"].(int); ok {
		property.AnoConstruccion = anoConstruccion
	}
	if aptoOficina, ok := updates["apto_oficina"].(bool); ok {
		property.AptoOficina = aptoOficina
	}
	if viviendaSocial, ok := updates["vivienda_social"].(bool); ok {
		property.ViviendaSocial = viviendaSocial
	}
	if condominio, ok := updates["condominio"].(bool); ok {
		property.Condominio = condominio
	}
	if cantidadPisos, ok := updates["cantidad_pisos"].(int); ok {
		property.CantidadPisos = cantidadPisos
	}
	if noAplicaConfort, ok := updates["no_aplica_confort"].(bool); ok {
		property.NoAplicaConfort = noAplicaConfort
	}
	if videoURL, ok := updates["video_url"].(string); ok {
		property.VideoURL = videoURL
	}
	if descripcion, ok := updates["descripcion"].(string); ok {
		property.Descripcion = descripcion
	}
	if terminosCondiciones, ok := updates["terminos_condiciones"].(string); ok {
		property.TerminosCondiciones = terminosCondiciones
	}
	if advisorID, ok := updates["advisor_id"].(*uint); ok {
		property.AdvisorID = advisorID
	}
	if mostrarVendedor, ok := updates["mostrar_vendedor"].(bool); ok {
		property.MostrarVendedor = mostrarVendedor
	}
	if mostrarAgencia, ok := updates["mostrar_agencia"].(bool); ok {
		property.MostrarAgencia = mostrarAgencia
	}
	if agenciaNombrePersonalizado, ok := updates["agencia_nombre_personalizado"].(string); ok {
		property.AgenciaNombrePersonalizado = agenciaNombrePersonalizado
	}

	// Guardar cambios
	if err := s.PropertyRepo.Update(property); err != nil {
		return nil, errors.DatabaseError(err)
	}

	return property, nil
}

// UpdatePropertyStatusAdmin - Actualiza el estado de una propiedad (para admin, sin verificar usuario)
func (s *PropertyService) UpdatePropertyStatusAdmin(id uint, newStatus string) (*models.Property, error) {
	// Obtener propiedad
	property, err := s.PropertyRepo.GetByID(id)
	if err != nil {
		return nil, errors.PropertyNotFound(id)
	}

	// Validar estado
	validStatuses := map[string]bool{
		"activo":    true,
		"inactivo":  true,
		"pendiente": true,
		"vendido":   true,
		"rechazado": true,
	}

	if !validStatuses[newStatus] {
		return nil, errors.ValidationError("estado", "Estado inválido. Debe ser: activo, inactivo, pendiente, vendido o rechazado")
	}

	// Actualizar estado
	property.EstadoPropiedad = models.PropertyStatus(newStatus)
	if err := s.PropertyRepo.Update(property); err != nil {
		return nil, errors.DatabaseError(err)
	}

	return property, nil
}

// ToggleFeatured - Marca o quita destacado de una propiedad
func (s *PropertyService) ToggleFeatured(id uint, destacado bool) (*models.Property, error) {
	// Obtener propiedad
	property, err := s.PropertyRepo.GetByID(id)
	if err != nil {
		return nil, errors.PropertyNotFound(id)
	}

	// Actualizar destacado
	property.Destacado = destacado
	if err := s.PropertyRepo.Update(property); err != nil {
		return nil, errors.DatabaseError(err)
	}

	return property, nil
}

// AssignAdvisor - Asigna o remueve un asesor de una propiedad
func (s *PropertyService) AssignAdvisor(id uint, advisorID *uint) (*models.Property, error) {
	// Obtener propiedad
	property, err := s.PropertyRepo.GetByID(id)
	if err != nil {
		return nil, errors.PropertyNotFound(id)
	}

	// Actualizar asesor
	property.AdvisorID = advisorID
	if err := s.PropertyRepo.Update(property); err != nil {
		return nil, errors.DatabaseError(err)
	}

	return property, nil
}

// Helper functions para extraer nombre de archivo de URL
func extractFilenameFromURL(imageURL string) string {
	// La URL viene en formato /api/imagenes/propiedad-123-456.png
	// Necesitamos extraer solo el filename
	runes := []rune(imageURL)
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == '/' {
			return string(runes[i+1:])
		}
	}
	return imageURL
}

// extractThumbnailFilenameFromURL - Genera el nombre del thumbnail
// propiedad-123-456.png -> propiedad-123-456-thumb.png
func extractThumbnailFilenameFromURL(imageURL string) string {
	filename := extractFilenameFromURL(imageURL)
	// Agregar -thumb antes de la extensión
	runes := []rune(filename)
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == '.' {
			return string(runes[:i]) + "-thumb" + string(runes[i:])
		}
	}
	return filename + "-thumb"
}
