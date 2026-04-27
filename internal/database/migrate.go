package database

import (
	"fmt"
	"log"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
)

// AutoMigrate - Ejecuta migraciones automáticas de las tablas
func AutoMigrate() error {
	// Conectar a la base de datos
	db, err := config.NewDatabase()
	if err != nil {
		return fmt.Errorf("error conectando a la base de datos: %w", err)
	}

	// Migrar todos los modelos
	log.Println("🔄 Ejecutando migraciones automáticas...")
	err = db.DB.AutoMigrate(
		&models.User{},
		&models.Property{},
		&models.PropertyImage{},
		&models.PropertyAdvisor{}, // Múltiples asesores por propiedad
		&models.Advisor{},         // Asesores inmobiliarios
		&models.AuditLog{},         // 🔒 Auditoría
		&models.Plan{},             // 📋 Planes de suscripción
	)
	if err != nil {
		return fmt.Errorf("error ejecutando migración: %w", err)
	}

	// Migraciones manuales para cambios que AutoMigrate no detecta
	// Cambiar image_url de varchar(1000) a TEXT para soportar base64
	if err := migratePropertyImageURL(db); err != nil {
		log.Printf("⚠️  Advertencia: %v\n", err)
		// No fallar si la migración manual falla (puede que ya se haya ejecutado)
	}

	// Configurar búsqueda full-text en PostgreSQL
	if err := SetupFullTextSearch(); err != nil {
		log.Printf("⚠️  Advertencia configurando búsqueda full-text: %v\n", err)
		// No fallar si la configuración falla (puede que ya se haya ejecutado)
	}

	log.Println("✅ Migraciones ejecutadas exitosamente")
	return nil
}

// migratePropertyImageURL - Migra la columna image_url a TEXT
func migratePropertyImageURL(db *config.Database) error {
	// Verificar si la columna ya es TEXT
	var columnType string
	err := db.DB.Raw(`
		SELECT data_type
		FROM information_schema.columns
		WHERE table_name = 'propiedades_imagenes'
		AND column_name = 'image_url'
	`).Scan(&columnType).Error

	if err != nil {
		return fmt.Errorf("error verificando tipo de columna: %w", err)
	}

	// Si ya es TEXT, no hacer nada
	if columnType == "text" {
		log.Println("✓ La columna image_url ya es TEXT")
		return nil
	}

	// Alterar la columna a TEXT
	log.Println("Migrando columna image_url a TEXT...")
	err = db.DB.Exec(`
		ALTER TABLE propiedades_imagenes
		ALTER COLUMN image_url TYPE TEXT
	`).Error

	if err != nil {
		return fmt.Errorf("error alterando columna image_url: %w", err)
	}

	log.Println("✓ Columna image_url migrada a TEXT exitosamente")
	return nil
}
