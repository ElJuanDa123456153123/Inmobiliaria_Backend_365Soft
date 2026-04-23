package handlers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	image "image"
	_ "image/jpeg"
	_ "image/png"
	"image/png"
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

// UploadImage - Sube una imagen y la guarda como .webp
func (h *ImageHandler) UploadImage(c *fiber.Ctx) error {
	// Obtener archivo del formulario
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No se proporcionó ninguna imagen",
		})
	}

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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al crear directorio de uploads",
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

	// Guardar imagen optimizada
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

	// 🔒 Generar thumbnail (300x300)
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

	return c.JSON(fiber.Map{
		"message":    "Imagen subida exitosamente",
		"url":        imageURL,
		"filename":   filename,
		"thumbnail":  fmt.Sprintf("/api/imagenes/%s", thumbnailName),
		"width":      optimizedImg.Bounds().Dx(),
		"height":     optimizedImg.Bounds().Dy(),
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al crear directorio de uploads",
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
		filename := fmt.Sprintf("propiedad-%d-%d.webp", timestamp, i)
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

		// Codificar como PNG (cambiar a WebP real si tienes la librería)
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
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Imagen no encontrada",
		})
	}

	// Servir el archivo
	return c.SendFile(filepath)
}
