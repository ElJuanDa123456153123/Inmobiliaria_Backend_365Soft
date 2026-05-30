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
		&models.Favorite{},         // ❤️ Favoritos
		&models.SavedSearch{},      // 🔍 Búsquedas guardadas
		&models.Notification{},     // 🔔 Notificaciones
		&models.ViewHistory{},      // 👁️ Historial de vistas
		&models.Payment{},          // 💳 Pagos y suscripciones
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

	// Migrar datos existentes (mostrar_vendedor y asesores)
	if err := MigrateExistingData(db); err != nil {
		log.Printf("⚠️  Advertencia migrando datos existentes: %v\n", err)
		// No fallar si la migración falla
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

// MigrateExistingData - Migra datos existentes para sincronizar usuarios y asesores,
// y actualizar propiedades para mostrar 365Soft por defecto (mostrar_vendedor=false, mostrar_agencia=false)
func MigrateExistingData(db *config.Database) error {
	log.Println("🔄 Verificando datos existentes para migrar...")

	// 1. Agregar columna mostrar_agencia si no existe (GORM AutoMigrate lo hace, pero aseguramos datos)
	// GORM AutoMigrate ya agregó la columna con default:false

	// 2. Actualizar propiedades para mostrar_vendedor = false (365Soft por defecto)
	var propsUpdated int64
	if err := db.DB.Model(&models.Property{}).
		Where("mostrar_vendedor IS NULL").
		Update("mostrar_vendedor", false).Error; err != nil {
		log.Printf("❌ Error actualizando mostrar_vendedor: %v\n", err)
	} else {
		propsUpdated = db.DB.RowsAffected
		if propsUpdated > 0 {
			log.Printf("✅ Propiedades actualizadas (mostrar_vendedor=false): %d\n", propsUpdated)
		}
	}

	// 3. Asegurar que mostrar_agencia tenga valor (false por defecto)
	var agenciaUpdated int64
	if err := db.DB.Model(&models.Property{}).
		Where("mostrar_agencia IS NULL").
		Update("mostrar_agencia", false).Error; err != nil {
		log.Printf("❌ Error actualizando mostrar_agencia: %v\n", err)
	} else {
		agenciaUpdated = db.DB.RowsAffected
		if agenciaUpdated > 0 {
			log.Printf("✅ Propiedades actualizadas (mostrar_agencia=false): %d\n", agenciaUpdated)
		}
	}

	// 4. Crear registros en asesores para usuarios con rol "agente" que no tengan uno
	var users []models.User
	if err := db.DB.Where("rol = ? AND deleted_at IS NULL", models.RoleAgente).Find(&users).Error; err != nil {
		log.Printf("❌ Error buscando usuarios con rol agente: %v\n", err)
		return err
	}

	advisorsCreated := 0
	for _, user := range users {
		// Verificar si ya existe un asesor para este usuario
		var existingAdvisor models.Advisor
		if err := db.DB.Where("user_id = ?", user.ID).First(&existingAdvisor).Error; err == nil {
			// Ya existe, continuar
			continue
		}

		// Crear nuevo asesor
		advisor := models.Advisor{
			Nombre:    user.Nombre,
			Email:     user.Email,
			Telefono:  user.Telefono,
			AvatarURL: user.AvatarURL,
			Activo:    user.Activo,
			UserID:    &user.ID,
		}

		if err := db.DB.Create(&advisor).Error; err != nil {
			log.Printf("❌ Error creando asesor para usuario %d: %v\n", user.ID, err)
		} else {
			advisorsCreated++
			log.Printf("✅ Asesor creado para usuario %d (%s)\n", user.ID, user.Nombre)
		}
	}

	if advisorsCreated > 0 {
		log.Printf("✅ Total de asesores creados: %d\n", advisorsCreated)
	}

	if propsUpdated > 0 || agenciaUpdated > 0 || advisorsCreated > 0 {
		log.Printf("🎉 Migración de datos completada: %d propiedades actualizadas (vendedor), %d propiedades actualizadas (agencia), %d asesores creados\n", propsUpdated, agenciaUpdated, advisorsCreated)
	}

	return nil
}
