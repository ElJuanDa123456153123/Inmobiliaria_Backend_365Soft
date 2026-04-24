package jobs

import (
	"fmt"
	"os"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
)

// CleanupJob - Job para limpiar imágenes de propiedades eliminadas
type CleanupJob struct {
	DB *config.Database
}

// NewCleanupJob - Crea un nuevo job de limpieza
func NewCleanupJob() (*CleanupJob, error) {
	db, err := config.NewDatabase()
	if err != nil {
		return nil, err
	}

	return &CleanupJob{
		DB: db,
	}, nil
}

// CleanupOldPropertyImages - Elimina imágenes físicas de propiedades eliminadas hace más de X días
func (j *CleanupJob) CleanupOldPropertyImages(retentionDays int) error {
	fmt.Printf("🧹 Iniciando limpieza de imágenes de propiedades eliminadas (retención: %d días)\n", retentionDays)

	// Calcular la fecha límite
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	fmt.Printf("📅 Fecha límite: %s\n", cutoffDate.Format("2006-01-02"))

	// Buscar propiedades eliminadas antes de la fecha límite
	var deletedProperties []models.Property
	err := j.DB.DB.Unscoped().Where("deleted_at < ?", cutoffDate).Find(&deletedProperties).Error
	if err != nil {
		fmt.Printf("❌ Error buscando propiedades eliminadas: %v\n", err)
		return err
	}

	if len(deletedProperties) == 0 {
		fmt.Println("✅ No hay propiedades para limpiar")
		return nil
	}

	fmt.Printf("📦 Encontradas %d propiedades eliminadas para limpiar\n", len(deletedProperties))

	// Para cada propiedad, eliminar sus imágenes físicas
	totalImagesDeleted := 0
	for _, property := range deletedProperties {
		// Buscar imágenes de la propiedad
		var images []models.PropertyImage
		err := j.DB.DB.Unscoped().Where("property_id = ?", property.ID).Find(&images).Error
		if err != nil {
			fmt.Printf("❌ Error buscando imágenes de propiedad %d: %v\n", property.ID, err)
			continue
		}

		if len(images) == 0 {
			continue
		}

		fmt.Printf("🗑️  Limpiando %d imágenes de propiedad ID %d (%s)\n", len(images), property.ID, property.Titulo)

		// Eliminar archivos físicos
		for _, image := range images {
			// Extraer nombre de archivo de la URL
			// URL formato: /api/imagenes/propiedad-123-456.png
			imagePath := fmt.Sprintf("uploads/imagenes/%s", extractFilename(image.ImageURL))

			// Eliminar archivo principal
			if err := os.Remove(imagePath); err != nil {
				if !os.IsNotExist(err) {
					fmt.Printf("⚠️  Error eliminando %s: %v\n", imagePath, err)
				}
			} else {
				fmt.Printf("✅ Eliminado: %s\n", imagePath)
				totalImagesDeleted++
			}

			// Eliminar thumbnail
			thumbnailPath := fmt.Sprintf("uploads/imagenes/%s", extractThumbnailFilename(image.ImageURL))
			if err := os.Remove(thumbnailPath); err != nil {
				if !os.IsNotExist(err) {
					fmt.Printf("⚠️  Error eliminando thumbnail %s: %v\n", thumbnailPath, err)
				}
			} else {
				fmt.Printf("✅ Eliminado thumbnail: %s\n", thumbnailPath)
				totalImagesDeleted++
			}
		}
	}

	fmt.Printf("🎉 Limpieza completada. %d imágenes eliminadas\n", totalImagesDeleted)
	return nil
}

// extractFilename - Extrae el nombre de archivo de la URL
// /api/imagenes/propiedad-123-456.png -> propiedad-123-456.png
func extractFilename(imageURL string) string {
	// La URL viene en formato /api/imagenes/filename.png
	// Necesitamos extraer solo el filename
	runes := []rune(imageURL)
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == '/' {
			return string(runes[i+1:])
		}
	}
	return imageURL
}

// extractThumbnailFilename - Genera el nombre del thumbnail
// propiedad-123-456.png -> propiedad-123-456-thumb.png
func extractThumbnailFilename(imageURL string) string {
	filename := extractFilename(imageURL)
	// Agregar -thumb antes de la extensión
	runes := []rune(filename)
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == '.' {
			return string(runes[:i]) + "-thumb" + string(runes[i:])
		}
	}
	return filename + "-thumb"
}

// StartCleanupScheduler - Inicia el scheduler para ejecutar la limpieza periódicamente
func (j *CleanupJob) StartCleanupScheduler(retentionDays int, intervalHours int) {
	fmt.Printf("⏰ Scheduler iniciado: Ejecutando limpieza cada %d horas\n", intervalHours)
	fmt.Printf("📅 Período de retención: %d días\n", retentionDays)

	ticker := time.NewTicker(time.Duration(intervalHours) * time.Hour)
	defer ticker.Stop()

	// Ejecutar inmediatamente al inicio
	j.CleanupOldPropertyImages(retentionDays)

	for range ticker.C {
		j.CleanupOldPropertyImages(retentionDays)
	}
}
