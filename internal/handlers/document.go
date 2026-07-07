package handlers

import (
	"fmt"
	"strconv"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/repository"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// DocumentHandler - Handler para documentos
type DocumentHandler struct {
	DB                *gorm.DB
	DocumentRepo      *repository.DocumentRepository
	PropertyRepo      *repository.PropertyRepository
	Storage           *services.StorageService
}

// NewDocumentHandler - Crea un nuevo handler de documentos
func NewDocumentHandler(cfg *config.Config) (*DocumentHandler, error) {
	dbConfig, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	// Inicializar servicio de almacenamiento
	storage, err := services.NewStorageService(cfg.R2)
	if err != nil {
		return nil, fmt.Errorf("error inicializando storage: %w", err)
	}

	return &DocumentHandler{
		DB:           dbConfig.DB,
		DocumentRepo: repository.NewDocumentRepository(dbConfig.DB),
		PropertyRepo: repository.NewPropertyRepository(dbConfig),
		Storage:      storage,
	}, nil
}

// UploadDocument - Sube un nuevo documento
func (h *DocumentHandler) UploadDocument(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	// Obtener archivo del formulario
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No se proporcionó archivo"})
	}

	// Validar tamaño del archivo (max 10MB)
	if file.Size > 10*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "El archivo excede el tamaño máximo de 10MB"})
	}

	// Obtener metadatos del formulario
	documentType := c.FormValue("type")
	title := c.FormValue("title")
	propertyIDStr := c.FormValue("property_id")
	isPublicStr := c.FormValue("is_public")
	description := c.FormValue("description")

	// Log para debug
	fmt.Printf("DEBUG: Recibiendo documento - type=%s, title=%s\n", documentType, title)

	// Validar tipo de documento
	if documentType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tipo de documento es requerido"})
	}

	// Validar tipos permitidos
	validTypes := map[string]bool{
		"PROPERTY_PHOTO": true,
		"RECEIPT":        true,
		"CONTRACT":       true,
		"PUBLIC":         true,
	}
	if !validTypes[documentType] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tipo de documento no válido"})
	}

	// Validar título
	if title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Título es requerido"})
	}

	// Parsear property_id si se proporciona
	var propertyID *uint
	if propertyIDStr != "" {
		propID, err := strconv.ParseUint(propertyIDStr, 10, 32)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID de propiedad inválido"})
		}
		pID := uint(propID)

		// Verificar que la propiedad existe
		_, err = h.PropertyRepo.GetByID(pID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Propiedad no encontrada"})
		}
		propertyID = &pID
	}

	// Parsear is_public
	isPublic := false
	if isPublicStr == "true" {
		isPublic = true
	}

	// Subir archivo usando el servicio de almacenamiento
	fileURL, fileSize, err := h.Storage.UploadFile(file, userID, documentType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Error subiendo archivo: %v", err),
		})
	}

	// Crear documento
	document := models.Document{
		UserID:      userID,
		PropertyID:  propertyID,
		Type:        documentType,
		Title:       title,
		FileURL:     fileURL,
		FileName:    file.Filename,
		FileSize:    fileSize,
		MimeType:    file.Header.Get("Content-Type"),
		IsPublic:    isPublic,
		Description: description,
	}

	// Guardar en base de datos
	if err := h.DocumentRepo.Create(&document); err != nil {
		// Si falla, eliminar el archivo subido
		_ = h.Storage.DeleteFile(fileURL)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error creando documento"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":   "Documento subido exitosamente",
		"document":  document,
	})
}

// GetDocuments - Obtiene documentos del usuario autenticado
func (h *DocumentHandler) GetDocuments(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	// Obtener parámetros de paginación
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	// Validar límite
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}
	if page < 1 {
		page = 1
	}

	// Obtener filtro por tipo si existe
	docType := c.Query("type")

	var documents []models.Document
	var total int64
	var totalPages int

	if docType != "" {
		// Filtrar por tipo
		documents, _ := h.DocumentRepo.GetByType(userID, docType)
		total = int64(len(documents))
		totalPages = 1
		return c.JSON(fiber.Map{
			"documents":   documents,
			"total":       total,
			"total_pages": totalPages,
			"page":        page,
			"limit":       limit,
		})
	} else {
		// Obtener todos los documentos del usuario
		var err error
		documents, total, totalPages, err = h.DocumentRepo.GetByUserID(userID, page, limit)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error obteniendo documentos"})
		}
	}

	return c.JSON(fiber.Map{
		"documents":   documents,
		"total":       total,
		"total_pages": totalPages,
		"page":        page,
		"limit":       limit,
	})
}

// GetPublicDocuments - Obtiene documentos públicos
func (h *DocumentHandler) GetPublicDocuments(c *fiber.Ctx) error {
	// Obtener parámetros de paginación
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	// Validar límite
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}
	if page < 1 {
		page = 1
	}

	documents, total, totalPages, err := h.DocumentRepo.GetPublicDocuments(page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error obteniendo documentos públicos"})
	}

	return c.JSON(fiber.Map{
		"documents":   documents,
		"total":       total,
		"total_pages": totalPages,
		"page":        page,
		"limit":       limit,
	})
}

// GetPropertyDocuments - Obtiene documentos de una propiedad (solo públicos)
func (h *DocumentHandler) GetPropertyDocuments(c *fiber.Ctx) error {
	propertyIDStr := c.Params("id")
	propertyID, err := strconv.ParseUint(propertyIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID de propiedad inválido"})
	}

	documents, err := h.DocumentRepo.GetByPropertyID(uint(propertyID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error obteniendo documentos"})
	}

	return c.JSON(fiber.Map{
		"documents": documents,
		"total":     int64(len(documents)),
	})
}

// DeleteDocument - Elimina un documento
func (h *DocumentHandler) DeleteDocument(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	documentIDStr := c.Params("id")
	documentID, err := strconv.ParseUint(documentIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID de documento inválido"})
	}

	// Verificar si es admin
	isAdmin := c.Locals("is_admin") == true

	// Verificar que el documento existe y pertenece al usuario (o es admin)
	document, err := h.DocumentRepo.GetByID(uint(documentID))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Documento no encontrado"})
	}

	if !isAdmin && document.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "No tienes permiso para eliminar este documento"})
	}

	// Eliminar documento
	if err := h.DocumentRepo.Delete(uint(documentID)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error eliminando documento"})
	}

	// Eliminar archivo del storage
	if err := h.Storage.DeleteFile(document.FileURL); err != nil {
		// Loguear error pero no fallar la operación
		fmt.Printf("⚠️  Error eliminando archivo del storage: %v\n", err)
	}

	return c.JSON(fiber.Map{"message": "Documento eliminado exitosamente"})
}

// GetAllDocuments - Obtiene todos los documentos (admin only)
func (h *DocumentHandler) GetAllDocuments(c *fiber.Ctx) error {
	// Verificar si es admin
	isAdmin := c.Locals("is_admin") == true
	if !isAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Solo administradores pueden ver todos los documentos"})
	}

	// Obtener parámetros de paginación
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	// Validar límite
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 50
	}
	if page < 1 {
		page = 1
	}

	// Obtener filtro por tipo si existe
	docType := c.Query("type")

	// Obtener todos los documentos
	documents, total, totalPages, err := h.DocumentRepo.GetAll(page, limit, docType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error obteniendo documentos"})
	}

	return c.JSON(fiber.Map{
		"documents":   documents,
		"total":       total,
		"total_pages": totalPages,
		"page":        page,
		"limit":       limit,
	})
}
