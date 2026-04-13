package services

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/dto"
	apperrors "github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/errors"
	"github.com/disintegration/imaging"
)

// StorageService - Interface para servicio de almacenamiento
type StorageService interface {
	// Upload de fotos
	UploadPropertyPhoto(ctx context.Context, propertyID uint, file *dto.PhotoFile, orden int) (*PhotoUploadResult, error)
	UploadPropertyPhotos(ctx context.Context, propertyID uint, files []dto.PhotoFile) (*PhotoUploadResult, error)

	// Procesamiento de imágenes
	ProcessImage(src io.Reader) (*ProcessedImages, error)
	GenerateThumbnail(src io.Reader, width, height int) ([]byte, string, error)

	// Gestión de archivos
	DeletePhoto(ctx context.Context, url string) error
	PhotoExists(ctx context.Context, url string) (bool, error)

	// URLs
	GetPublicURL(propertyID uint, filename string) string
	GetThumbnailURL(propertyID uint, filename string) string
}

// PhotoUploadResult - Resultado de upload de foto
type PhotoUploadResult struct {
	URL           string `json:"url"`
	URLThumbnail  string `json:"url_thumbnail"`
	FileName      string `json:"file_name"`
	FileSize      int64  `json:"file_size"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	ThumbnailSize int    `json:"thumbnail_size"`
}

// ProcessedImages - Imágenes procesadas
type ProcessedImages struct {
	Original      []byte
	OriginalSize  int64
	Processed     []byte
	ProcessedSize int64
	Thumbnail     []byte
	ThumbnailSize int64
	Width         int
	Height        int
	Format        string
}

// storageService - Implementación del servicio (local storage)
type storageService struct {
	uploadDir    string // Directorio base de upload
	publicURL    string // URL base pública
	maxFileSize  int64  // Tamaño máximo en bytes
	allowedExts  []string
}

// NewStorageService - Crea una nueva instancia del servicio
func NewStorageService(uploadDir, publicURL string) StorageService {
	return &storageService{
		uploadDir:   uploadDir,
		publicURL:   publicURL,
		maxFileSize: 5 * 1024 * 1024, // 5MB
		allowedExts: []string{".jpg", ".jpeg", ".png", ".webp"},
	}
}

// ========== UPLOAD DE FOTOS ==========

// UploadPropertyPhoto - Sube una foto de propiedad
func (s *storageService) UploadPropertyPhoto(ctx context.Context, propertyID uint, file *dto.PhotoFile, orden int) (*PhotoUploadResult, error) {
	// 1. Validaciones
	if err := s.validateFile(file); err != nil {
		return nil, err
	}

	// 2. Crear directorio de propiedad
	propertyDir := filepath.Join(s.uploadDir, "propiedades", fmt.Sprintf("%d", propertyID))
	if err := os.MkdirAll(propertyDir, 0755); err != nil {
		return nil, apperrors.InternalError(err)
	}

	// 3. Generar nombre único
	timestamp := time.Now().Unix()
	ext := filepath.Ext(file.Filename)
	baseName := fmt.Sprintf("%d_%d", propertyID, timestamp)
	originalName := fmt.Sprintf("%s%s", baseName, ext)
	thumbnailName := fmt.Sprintf("%s_thumb%s", baseName, ext)

	// 4. Procesar imagen
	processed, err := s.ProcessImage(bytes.NewReader(file.Content))
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeStorageUpload, "procesando imagen", http.StatusInternalServerError)
	}

	// 5. Guardar imagen procesada
	originalPath := filepath.Join(propertyDir, originalName)
	if err := os.WriteFile(originalPath, processed.Processed, 0644); err != nil {
		return nil, apperrors.InternalError(err)
	}

	// 6. Guardar thumbnail
	thumbnailPath := filepath.Join(propertyDir, thumbnailName)
	if err := os.WriteFile(thumbnailPath, processed.Thumbnail, 0644); err != nil {
		return nil, apperrors.InternalError(err)
	}

	// 7. Generar URLs
	url := s.GetPublicURL(propertyID, originalName)
	urlThumbnail := s.GetThumbnailURL(propertyID, thumbnailName)

	// 8. Retornar resultado
	return &PhotoUploadResult{
		URL:           url,
		URLThumbnail:  urlThumbnail,
		FileName:      originalName,
		FileSize:      int64(len(processed.Processed)),
		Width:         processed.Width,
		Height:        processed.Height,
		ThumbnailSize: len(processed.Thumbnail),
	}, nil
}

// UploadPropertyPhotos - Sube múltiples fotos
func (s *storageService) UploadPropertyPhotos(ctx context.Context, propertyID uint, files []dto.PhotoFile) (*PhotoUploadResult, error) {
	// Validar cantidad
	if len(files) == 0 {
		return nil, apperrors.ValidationError("files", "No hay archivos para subir")
	}

	if len(files) > 7 {
		return nil, apperrors.LimitExceeded("fotos", len(files))
	}

	// Subir primera foto como referencia
	result, err := s.UploadPropertyPhoto(ctx, propertyID, &files[0], 0)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ========== PROCESAMIENTO DE IMÁGENES ==========

// ProcessImage - Procesa una imagen (resize + thumbnail)
func (s *storageService) ProcessImage(src io.Reader) (*ProcessedImages, error) {
	// 1. Decodificar imagen
	img, format, err := image.Decode(src)
	if err != nil {
		return nil, fmt.Errorf("error decodificando imagen: %w", err)
	}

	// 2. Obtener dimensiones originales
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 3. Redimensionar si es necesario (max 1920x1080)
	maxWidth, maxHeight := 1920, 1080
	processed := img

	if width > maxWidth || height > maxHeight {
		processed = imaging.Fit(img, maxWidth, maxHeight, imaging.Lanczos)
		bounds = processed.Bounds()
		width = bounds.Dx()
		height = bounds.Dy()
	}

	// 4. Codificar imagen procesada
	var processedBuf bytes.Buffer
	switch format {
	case "jpeg":
		err = jpeg.Encode(&processedBuf, processed, &jpeg.Options{Quality: 85})
	case "png":
		err = png.Encode(&processedBuf, processed)
	default:
		// Intentar JPEG por defecto
		err = jpeg.Encode(&processedBuf, processed, &jpeg.Options{Quality: 85})
	}

	if err != nil {
		return nil, fmt.Errorf("error codificando imagen: %w", err)
	}

	// 5. Generar thumbnail
	thumbnailBuf, _, err := s.GenerateThumbnail(bytes.NewReader(processedBuf.Bytes()), 300, 300)
	if err != nil {
		return nil, fmt.Errorf("error generando thumbnail: %w", err)
	}

	// 6. Retornar resultado
	return &ProcessedImages{
		Original:      nil, // No guardamos el original para ahorrar espacio
		OriginalSize:  0,
		Processed:     processedBuf.Bytes(),
		ProcessedSize: int64(processedBuf.Len()),
		Thumbnail:     thumbnailBuf,
		ThumbnailSize: int64(len(thumbnailBuf)),
		Width:         width,
		Height:        height,
		Format:        format,
	}, nil
}

// GenerateThumbnail - Genera thumbnail de una imagen
func (s *storageService) GenerateThumbnail(src io.Reader, width, height int) ([]byte, string, error) {
	// 1. Decodificar imagen
	img, format, err := image.Decode(src)
	if err != nil {
		return nil, "", fmt.Errorf("error decodificando imagen: %w", err)
	}

	// 2. Redimensionar a thumbnail (cover fit)
	thumbnail := imaging.Fill(img, width, height, imaging.Center, imaging.Lanczos)

	// 3. Codificar thumbnail
	var buf bytes.Buffer
	var ext string

	switch format {
	case "jpeg":
		ext = ".jpg"
		err = jpeg.Encode(&buf, thumbnail, &jpeg.Options{Quality: 80})
	case "png":
		ext = ".png"
		err = png.Encode(&buf, thumbnail)
	default:
		ext = ".jpg"
		err = jpeg.Encode(&buf, thumbnail, &jpeg.Options{Quality: 80})
	}

	if err != nil {
		return nil, "", fmt.Errorf("error codificando thumbnail: %w", err)
	}

	return buf.Bytes(), ext, nil
}

// ========== GESTIÓN DE ARCHIVOS ==========

// DeletePhoto - Elimina una foto del storage
func (s *storageService) DeletePhoto(ctx context.Context, url string) error {
	// 1. Extraer path de la URL
	// URL formato: /uploads/propiedades/123/123_1234567890.jpg
	path := filepath.Join(s.uploadDir, url)

	// 2. Verificar que existe
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return apperrors.NotFound("archivo")
	}

	// 3. Eliminar archivo
	if err := os.Remove(path); err != nil {
		return apperrors.InternalError(err)
	}

	// 4. Intentar eliminar thumbnail también
	thumbnailPath := strings.Replace(path, ".", "_thumb.", 1)
	_ = os.Remove(thumbnailPath) // No error si no existe

	return nil
}

// PhotoExists - Verifica si una foto existe
func (s *storageService) PhotoExists(ctx context.Context, url string) (bool, error) {
	path := filepath.Join(s.uploadDir, url)

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// ========== URLS ==========

// GetPublicURL - Genera URL pública
func (s *storageService) GetPublicURL(propertyID uint, filename string) string {
	return fmt.Sprintf("%s/propiedades/%d/%s", s.publicURL, propertyID, filename)
}

// GetThumbnailURL - Genera URL de thumbnail
func (s *storageService) GetThumbnailURL(propertyID uint, filename string) string {
	// Reemplazar extensión con _thumb.ext
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	return fmt.Sprintf("%s/propiedades/%d/%s_thumb%s", s.publicURL, propertyID, base, ext)
}

// ========== VALIDACIONES ==========

// validateFile - Valida un archivo
func (s *storageService) validateFile(file *dto.PhotoFile) error {
	// 1. Validar tamaño
	if file.Size > s.maxFileSize {
		return apperrors.FileTooLarge(int(s.maxFileSize / (1024 * 1024)))
	}

	if file.Size == 0 {
		return apperrors.ValidationError("file", "El archivo está vacío")
	}

	// 2. Validar extensión
	ext := strings.ToLower(filepath.Ext(file.Filename))
	validExt := false
	for _, allowedExt := range s.allowedExts {
		if ext == allowedExt {
			validExt = true
			break
		}
	}

	if !validExt {
		return apperrors.InvalidFileFormat(s.allowedExts)
	}

	// 3. Validar tipo MIME
	validMime := false
	allowedMimes := []string{"image/jpeg", "image/png", "image/webp"}
	for _, allowed := range allowedMimes {
		if file.MimeType == allowed {
			validMime = true
			break
		}
	}

	if !validMime {
		return apperrors.InvalidFileFormat(allowedMimes)
	}

	// 4. Validar que el contenido coincida con el tipo MIME
	if err := s.validateImageContent(file.Content, file.MimeType); err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInvalidFile, "Contenido inválido para el tipo MIME", http.StatusBadRequest)
	}

	return nil
}

// validateImageContent - Valida que el contenido coincida con el tipo MIME
func (s *storageService) validateImageContent(content []byte, mimeType string) error {
	_, format, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("no se pudo decodificar la imagen: %w", err)
	}

	// Verificar que el formato coincida con el MIME type
	switch mimeType {
	case "image/jpeg":
		if format != "jpeg" {
			return fmt.Errorf("el archivo no es un JPEG válido")
		}
	case "image/png":
		if format != "png" {
			return fmt.Errorf("el archivo no es un PNG válido")
		}
	case "image/webp":
		if format != "webp" {
			return fmt.Errorf("el archivo no es un WEBP válido")
		}
	}

	return nil
}

// UploadFromMultipart - Sube archivo desde multipart form
func (s *storageService) UploadFromMultipart(ctx context.Context, propertyID uint, fileHeader *multipart.FileHeader, orden int) (*PhotoUploadResult, error) {
	// 1. Abrir archivo
	file, err := fileHeader.Open()
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	defer file.Close()

	// 2. Leer contenido
	content := make([]byte, fileHeader.Size)
	if _, err := file.Read(content); err != nil {
		return nil, apperrors.InternalError(err)
	}

	// 3. Detectar tipo MIME
	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		// Detectar desde extensión
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		switch ext {
		case ".jpg", ".jpeg":
			mimeType = "image/jpeg"
		case ".png":
			mimeType = "image/png"
		case ".webp":
			mimeType = "image/webp"
		default:
			return nil, apperrors.InvalidFileFormat([]string{".jpg", ".png", ".webp"})
		}
	}

	// 4. Crear PhotoFile
	photoFile := &dto.PhotoFile{
		Filename: fileHeader.Filename,
		Size:     fileHeader.Size,
		Content:  content,
		MimeType: mimeType,
	}

	// 5. Subir usando método principal
	return s.UploadPropertyPhoto(ctx, propertyID, photoFile, orden)
}

// CreateUploadDirectory - Crea directorio de upload si no existe
func CreateUploadDirectory(uploadDir string) error {
	dirs := []string{
		uploadDir,
		filepath.Join(uploadDir, "propiedades"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("error creando directorio %s: %w", dir, err)
		}
	}

	return nil
}

// CleanupOldFiles - Limpia archivos antiguos (para usar como cron job)
func CleanupOldFiles(uploadDir string, days int) error {
	cutoff := time.Now().AddDate(0, 0, -days)

	propertyDir := filepath.Join(uploadDir, "propiedades")

	return filepath.Walk(propertyDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Saltar directorios
		if info.IsDir() {
			return nil
		}

		// Eliminar archivos más antiguos que cutoff
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(path); err != nil {
				return err
			}
		}

		return nil
	})
}
