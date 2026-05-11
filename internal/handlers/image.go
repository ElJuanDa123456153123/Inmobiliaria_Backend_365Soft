package handlers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	image "image"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"io"
	"os"
	stdpath "path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gofiber/fiber/v2"
)

// ImageHandler - Handler para manejo de imágenes
type ImageHandler struct{}

// NewImageHandler - Crea un nuevo handler de imágenes
func NewImageHandler() (*ImageHandler, error) {
	return &ImageHandler{}, nil
}

// UploadImage - Sube una imagen y la guarda como PNG
func (h *ImageHandler) UploadImage(c *fiber.Ctx) error {
	fmt.Println("📤 UploadImage llamado")

	// Obtener archivo del formulario
	file, err := c.FormFile("image")
	if err != nil {
		fmt.Printf("❌ Error obteniendo archivo del formulario: %v\n", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No se proporcionó ninguna imagen",
		})
	}

	fmt.Printf("✅ Archivo recibido: %s (tamaño: %d bytes)\n", file.Filename, file.Size)

	// Validar tamaño máximo (10MB)
	const maxFileSize = 10 * 1024 * 1024
	if file.Size > maxFileSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "La imagen es demasiado grande. Máximo 10MB",
		})
	}

	// Abrir el archivo
	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al abrir el archivo",
		})
	}
	defer src.Close()

	// Decodificar la imagen para validar que sea una imagen real
	img, format, err := image.Decode(src)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "El archivo no es una imagen válida",
		})
	}

	// Validar formato
	if format != "jpeg" && format != "png" && format != "webp" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato no soportado. Solo JPEG, PNG y WebP",
		})
	}

	// Crear directorio de uploads si no existe
	uploadDir := "uploads/imagenes"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		// Log del error para debugging
		fmt.Printf("❌ Error creando directorio '%s': %v\n", uploadDir, err)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":          fmt.Sprintf("Error al crear directorio de uploads: %s. Asegúrate de que la aplicación tiene permisos de escritura.", uploadDir),
			"detailed_error": err.Error(),
			"upload_dir":     uploadDir,
		})
	}

	// Generar nombre único para el archivo
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("propiedad-%d-%d.png", timestamp, file.Size)
	thumbnailName := fmt.Sprintf("propiedad-%d-%d-thumb.png", timestamp, file.Size)
	filepath := stdpath.Join(uploadDir, filename)
	thumbnailPath := stdpath.Join(uploadDir, thumbnailName)

	// 🔒 Procesar imagen optimizada
	// Redimensionar si es muy grande (máximo 1920px de ancho o alto)
	var optimizedImg image.Image
	if img.Bounds().Dx() > 1920 || img.Bounds().Dy() > 1920 {
		optimizedImg = imaging.Fit(img, 1920, 1920, imaging.Lanczos)
	} else {
		optimizedImg = img
	}

	// Guardar imagen optimizada como PNG
	out, err := os.Create(filepath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al crear el archivo",
		})
	}
	defer out.Close()

	if err := png.Encode(out, optimizedImg); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al codificar la imagen",
		})
	}

	// 🔒 Generar thumbnail (300x300) también como PNG
	thumbnail := imaging.Thumbnail(optimizedImg, 300, 300, imaging.Lanczos)
	thumbOut, err := os.Create(thumbnailPath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al crear el thumbnail",
		})
	}
	defer thumbOut.Close()

	if err := png.Encode(thumbOut, thumbnail); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al codificar el thumbnail",
		})
	}

	// Retornar la URL de la imagen
	imageURL := fmt.Sprintf("/api/imagenes/%s", filename)

	fmt.Printf("✅ Imagen guardada exitosamente: %s\n", filepath)
	fmt.Printf("✅ Thumbnail guardado: %s\n", thumbnailPath)

	return c.JSON(fiber.Map{
		"message":   "Imagen subida exitosamente",
		"url":       imageURL,
		"filename":  filename,
		"thumbnail": fmt.Sprintf("/api/imagenes/%s", thumbnailName),
		"width":     optimizedImg.Bounds().Dx(),
		"height":    optimizedImg.Bounds().Dy(),
	})
}

// UploadImagesFromBase64 - Sube múltiples imágenes desde base64
// Útil para migración desde el sistema anterior
func (h *ImageHandler) UploadImagesFromBase64(c *fiber.Ctx) error {
	// Parsear request body
	var req struct {
		Images []string `json:"images"` // Array de strings base64
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	if len(req.Images) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No se proporcionaron imágenes",
		})
	}

	if len(req.Images) > 30 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Máximo 30 imágenes permitidas",
		})
	}

	// Crear directorio de uploads si no existe
	uploadDir := "uploads/imagenes"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		// Log del error para debugging
		fmt.Printf("❌ Error creando directorio '%s': %v\n", uploadDir, err)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":          fmt.Sprintf("Error al crear directorio de uploads: %s. Asegúrate de que la aplicación tiene permisos de escritura.", uploadDir),
			"detailed_error": err.Error(),
			"upload_dir":     uploadDir,
		})
	}

	// Procesar cada imagen
	var urls []string
	timestamp := time.Now().Unix()

	for i, base64Data := range req.Images {
		// Limpiar el base64 si tiene prefijo data:image
		base64Data = strings.TrimPrefix(base64Data, "data:image/jpeg;base64,")
		base64Data = strings.TrimPrefix(base64Data, "data:image/png;base64,")
		base64Data = strings.TrimPrefix(base64Data, "data:image/webp;base64,")

		// Decodificar base64
		imgData, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, strings.NewReader(base64Data)))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Error decodificando imagen %d", i+1),
			})
		}

		// Decodificar la imagen
		img, _, err := image.Decode(bytes.NewReader(imgData))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Error decodificando imagen %d: formato inválido", i+1),
			})
		}

		// Generar nombre único
		filename := fmt.Sprintf("propiedad-%d-%d.png", timestamp, i)
		filepath := stdpath.Join(uploadDir, filename)

		// Redimensionar si es muy grande
		var resizedImg image.Image
		if img.Bounds().Dx() > 1920 {
			resizedImg = imaging.Fit(img, 1920, 0, imaging.Lanczos)
		} else {
			resizedImg = img
		}

		// Crear archivo
		out, err := os.Create(filepath)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Error guardando imagen %d", i+1),
			})
		}

		// Codificar como PNG
		if err := png.Encode(out, resizedImg); err != nil {
			out.Close()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Error codificando imagen %d", i+1),
			})
		}
		out.Close()

		// Agregar URL a la lista
		url := fmt.Sprintf("/api/imagenes/%s", filename)
		urls = append(urls, url)
	}

	return c.JSON(fiber.Map{
		"message": "Imágenes subidas exitosamente",
		"urls":    urls,
	})
}

// ServeImage - Sirve una imagen desde el sistema de archivos
func (h *ImageHandler) ServeImage(c *fiber.Ctx) error {
	filename := c.Params("filename")

	// Validar que no contenga path traversal
	if strings.Contains(filename, "..") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nombre de archivo inválido",
		})
	}

	// Construir ruta completa
	filepath := stdpath.Join("uploads/imagenes", filename)

	// Verificar que el archivo existe
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		// Log del error para debugging
		fmt.Printf("⚠️  Imagen no encontrada: %s\n", filepath)

		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":    "Imagen no encontrada",
			"filename": filename,
			"message":  fmt.Sprintf("La imagen '%s' no existe en el servidor. Asegúrate de que la propiedad tenga imágenes cargadas.", filename),
		})
	}

	// Servir el archivo
	return c.SendFile(filepath)
}

// UploadAvatar - Sube una imagen de avatar y la guarda como WEBP
func (h *ImageHandler) UploadAvatar(c *fiber.Ctx) error {
	fmt.Println("📤 UploadAvatar llamado")

	// Obtener archivo del formulario
	file, err := c.FormFile("image")
	if err != nil {
		fmt.Printf("❌ Error obteniendo archivo del formulario: %v\n", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No se proporcionó ninguna imagen",
		})
	}

	fmt.Printf("✅ Archivo recibido: %s (tamaño: %d bytes)\n", file.Filename, file.Size)

	// Validar tamaño máximo (5MB para avatares)
	const maxFileSize = 5 * 1024 * 1024
	if file.Size > maxFileSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "La imagen es demasiado grande. Máximo 5MB",
		})
	}

	// Abrir el archivo
	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al abrir el archivo",
		})
	}
	defer src.Close()

	// Decodificar la imagen para validar que sea una imagen real
	img, format, err := image.Decode(src)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "El archivo no es una imagen válida",
		})
	}

	// Validar formato
	if format != "jpeg" && format != "png" && format != "webp" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato no soportado. Solo JPEG, PNG y WebP",
		})
	}

	// Crear directorio de uploads para avatares si no existe
	uploadDir := "uploads/avatars"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		fmt.Printf("❌ Error creando directorio '%s': %v\n", uploadDir, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":          fmt.Sprintf("Error al crear directorio de uploads: %s", uploadDir),
			"detailed_error": err.Error(),
			"upload_dir":     uploadDir,
		})
	}

	// Generar nombre único para el archivo
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("avatar-%d.png", timestamp)
	filepath := stdpath.Join(uploadDir, filename)

	// Redimensionar avatar a máximo 400x400 (suficiente para perfil)
	var avatarImg image.Image
	if img.Bounds().Dx() > 400 || img.Bounds().Dy() > 400 {
		avatarImg = imaging.Fit(img, 400, 400, imaging.Lanczos)
	} else {
		avatarImg = img
	}

	// Guardar como WEBP (formato moderno y compresión excelente)
	out, err := os.Create(filepath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al crear el archivo",
		})
	}
	defer out.Close()

	// Guardar como PNG (formato más compatible y sin pérdida de calidad)
	// NOTA: Si se desea WEBP en el futuro, se necesita una librería adicional
	if err := png.Encode(out, avatarImg); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al codificar la imagen",
		})
	}

	// Retornar la URL del avatar (URL completa)
	// En desarrollo, usar localhost:3001 para que el navegador pueda acceder
	// En producción, usar el host real de la request
	var avatarURL string

	// Verificar si estamos en desarrollo por variable de entorno
	if os.Getenv("APP_ENV") == "development" {
		// En desarrollo, siempre usar localhost:3001
		avatarURL = fmt.Sprintf("http://localhost:3001/api/avatars/%s", filename)
	} else {
		// En producción, usar el host de la request
		host := c.Get("Host")
		if host == "" {
			host = "localhost:3001"
		}

		// Determinar protocolo
		protocol := "http"
		if c.Protocol() == "https" || c.Get("X-Forwarded-Proto") == "https" {
			protocol = "https"
		}

		avatarURL = fmt.Sprintf("%s://%s/api/avatars/%s", protocol, host, filename)
	}

	fmt.Printf("✅ Avatar guardado exitosamente: %s\n", filepath)
	fmt.Printf("🔗 URL generada: %s\n", avatarURL)

	return c.JSON(fiber.Map{
		"message":  "Avatar subido exitosamente",
		"url":      avatarURL,
		"filename": filename,
		"width":    avatarImg.Bounds().Dx(),
		"height":   avatarImg.Bounds().Dy(),
	})
}

// ServeAvatar - Sirve una imagen de avatar desde el sistema de archivos
func (h *ImageHandler) ServeAvatar(c *fiber.Ctx) error {
	filename := c.Params("filename")

	// Validar que no contenga path traversal
	if strings.Contains(filename, "..") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nombre de archivo inválido",
		})
	}

	// Construir ruta completa
	filepath := stdpath.Join("uploads/avatars", filename)

	// Verificar que el archivo existe
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		fmt.Printf("⚠️  Avatar no encontrado: %s\n", filepath)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":    "Avatar no encontrado",
			"filename": filename,
		})
	}

	// Servir el archivo
	return c.SendFile(filepath)
}
