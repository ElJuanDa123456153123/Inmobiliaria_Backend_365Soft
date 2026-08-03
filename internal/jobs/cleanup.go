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

// DeleteExpiredProperties - Elimina propiedades que han expirado
func (j *CleanupJob) DeleteExpiredProperties() error {
	fmt.Printf("🔍 Buscando propiedades expiradas...\n")

	// Buscar propiedades donde la fecha de expiración es menor a ahora
	var expiredProperties []models.Property
	now := time.Now()
	err := j.DB.DB.Where("fecha_expiracion IS NOT NULL AND fecha_expiracion < ?", now).
		Find(&expiredProperties).Error
	if err != nil {
		fmt.Printf("❌ Error buscando propiedades expiradas: %v\n", err)
		return err
	}

	if len(expiredProperties) == 0 {
		fmt.Println("✅ No hay propiedades expiradas para eliminar")
		return nil
	}

	fmt.Printf("📦 Encontradas %d propiedades expiradas para eliminar\n", len(expiredProperties))

	// Para cada propiedad, eliminarla junto con sus imágenes
	totalDeleted := 0
	for _, property := range expiredProperties {
		fmt.Printf("🗑️  Eliminando propiedad ID %d (%s) - Expiró: %s\n",
			property.ID, property.Titulo, property.FechaExpiracion.Format("2006-01-02 15:04"))

		// Buscar imágenes de la propiedad
		var images []models.PropertyImage
		err := j.DB.DB.Where("property_id = ?", property.ID).Find(&images).Error
		if err != nil {
			fmt.Printf("⚠️  Error buscando imágenes de propiedad %d: %v\n", property.ID, err)
		}

		// Eliminar archivos físicos de imágenes
		for _, image := range images {
			imagePath := fmt.Sprintf("uploads/imagenes/%s", extractFilename(image.ImageURL))
			thumbnailPath := fmt.Sprintf("uploads/imagenes/%s", extractThumbnailFilename(image.ImageURL))

			// Eliminar archivo principal
			if err := os.Remove(imagePath); err != nil {
				if !os.IsNotExist(err) {
					fmt.Printf("⚠️  Error eliminando %s: %v\n", imagePath, err)
				}
			}

			// Eliminar thumbnail
			if err := os.Remove(thumbnailPath); err != nil {
				if !os.IsNotExist(err) {
					fmt.Printf("⚠️  Error eliminando thumbnail %s: %v\n", thumbnailPath, err)
				}
			}
		}

		// Soft delete de la propiedad (GORM lo marca como deleted)
		if err := j.DB.DB.Delete(&property).Error; err != nil {
			fmt.Printf("❌ Error eliminando propiedad %d: %v\n", property.ID, err)
		} else {
			fmt.Printf("✅ Propiedad ID %d eliminada correctamente\n", property.ID)
			totalDeleted++
		}
	}

	fmt.Printf("🎉 Eliminación completada. %d propiedades eliminadas\n", totalDeleted)
	return nil
}

// StartExpirationScheduler - Inicia el scheduler para verificar y eliminar propiedades expiradas
func (j *CleanupJob) StartExpirationScheduler(intervalMinutes int) {
	fmt.Printf("⏰ Scheduler de expiración iniciado: Verificando cada %d minutos\n", intervalMinutes)

	ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
	defer ticker.Stop()

	// Ejecutar inmediatamente al inicio
	j.DeleteExpiredProperties()

	for range ticker.C {
		j.DeleteExpiredProperties()
	}
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

// CleanupOldSoftDeletedRecords - Elimina permanentemente registros soft-deleted antiguos
func (j *CleanupJob) CleanupOldSoftDeletedRecords(retentionMonths int) error {
	fmt.Printf("🧹 Iniciando limpieza de registros soft-deleted (retención: %d meses)\n", retentionMonths)

	// Calcular la fecha límite
	cutoffDate := time.Now().AddDate(0, -retentionMonths, 0)
	fmt.Printf("📅 Fecha límite: %s\n", cutoffDate.Format("2006-01-02"))

	// Contar cuántos registros serán eliminados
	var count int64
	err := j.DB.DB.Unscoped().Model(&models.Property{}).
		Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoffDate).
		Count(&count).Error
	if err != nil {
		fmt.Printf("❌ Error contando registros a eliminar: %v\n", err)
		return err
	}

	if count == 0 {
		fmt.Println("✅ No hay registros soft-deleted antiguos para limpiar")
		return nil
	}

	fmt.Printf("📦 Encontrados %d registros para eliminar permanentemente\n", count)

	// Eliminar permanentemente los registros soft-deleted antiguos
	// Unscoped() permite eliminar registros soft-deleted permanentemente
	result := j.DB.DB.Unscoped().
		Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoffDate).
		Delete(&models.Property{})

	if result.Error != nil {
		fmt.Printf("❌ Error eliminando registros: %v\n", result.Error)
		return result.Error
	}

	fmt.Printf("✅ Limpieza completada. %d registros eliminados permanentemente\n", result.RowsAffected)
	return nil
}

// StartSoftDeleteCleanupScheduler - Inicia el scheduler para limpiar soft-deletes periódicamente
func (j *CleanupJob) StartSoftDeleteCleanupScheduler(retentionMonths int, intervalDays int) {
	fmt.Printf("⏰ Scheduler de limpieza soft-delete iniciado: Ejecutando cada %d días\n", intervalDays)
	fmt.Printf("📅 Período de retención: %d meses\n", retentionMonths)

	ticker := time.NewTicker(time.Duration(intervalDays) * 24 * time.Hour)
	defer ticker.Stop()

	// Ejecutar inmediatamente al inicio
	j.CleanupOldSoftDeletedRecords(retentionMonths)

	for range ticker.C {
		j.CleanupOldSoftDeletedRecords(retentionMonths)
	}
}
