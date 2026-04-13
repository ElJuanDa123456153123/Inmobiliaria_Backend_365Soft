package database

import (
	"fmt"
	"log"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"gorm.io/gorm"
)

// AutoMigrate - Ejecuta migraciones automáticas de las tablas
func AutoMigrate() error {
	// Conectar a la base de datos
	db, err := config.NewDatabase()
	if err != nil {
		return fmt.Errorf("error conectando a la base de datos: %w", err)
	}

	log.Println("🔄 Iniciando migraciones...")

	// Migrar modelos en orden (respetando dependencias)
	models := []interface{}{
		&models.User{},           // 1. Usuarios (ya existe)
		&models.Category{},       // 2. Categorías
		&models.Property{},       // 3. Propiedades
		&models.PropertyPhoto{},  // 4. Fotos de propiedad
		&models.PropertyCategory{}, // 5. Tabla pivote propiedades-categorias
		&models.AuditLog{},       // 6. Logs de auditoría
	}

	for i, model := range models {
		log.Printf("   [%d/%d] Migrando: %T", i+1, len(models), model)
		err = db.DB.AutoMigrate(model)
		if err != nil {
			return fmt.Errorf("error migrando %T: %w", model, err)
		}
	}

	// Crear índices adicionales para optimización
	if err := createAdditionalIndexes(db.DB); err != nil {
		log.Printf("⚠️  Warning creando índices adicionales: %v", err)
	}

	// Insertar datos semilla (categorías iniciales)
	if err := seedInitialData(db.DB); err != nil {
		log.Printf("⚠️  Warning insertando datos semilla: %v", err)
	}

	log.Println("✅ Migraciones ejecutadas exitosamente")
	return nil
}

// createAdditionalIndexes - Crea índices compuestos para optimización
func createAdditionalIndexes(db *gorm.DB) error {
	log.Println("📊 Creando índices de optimización...")

	// Índice compuesto para búsqueda de propiedades
	// Este es el índice más usado para el listado principal
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_propiedades_busqueda
		ON propiedades(estado, tipo_propiedad, operacion, ciudad, destacado, created_at DESC)
		WHERE estado = 'activa' AND deleted_at IS NULL
	`).Error; err != nil {
		log.Printf("   ⚠️  Error creando idx_propiedades_busqueda: %v", err)
	}

	// Índice para filtros de precios
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_propiedades_precio
		ON propiedades(precio, estado)
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		log.Printf("   ⚠️  Error creando idx_propiedades_precio: %v", err)
	}

	// Índice para búsqueda por ciudad y estado
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_propiedades_ciudad_estado
		ON propiedades(ciudad, estado)
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		log.Printf("   ⚠️  Error creando idx_propiedades_ciudad_estado: %v", err)
	}

	// Índice para propiedades destacadas
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_propiedades_destacadas
		ON propiedades(destacado, estado, numero_visitas DESC)
		WHERE destacado = true AND estado = 'activa' AND deleted_at IS NULL
	`).Error; err != nil {
		log.Printf("   ⚠️  Error creando idx_propiedades_destacadas: %v", err)
	}

	// Índices para fotos
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_fotos_propiedad_orden
		ON fotos_propiedad(property_id, orden)
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		log.Printf("   ⚠️  Error creando idx_fotos_propiedad_orden: %v", err)
	}

	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_fotos_portada
		ON fotos_propiedad(property_id, es_portada)
		WHERE es_portada = true AND deleted_at IS NULL
	`).Error; err != nil {
		log.Printf("   ⚠️  Error creando idx_fotos_portada: %v", err)
	}

	// Índices para auditoría (partitioning simulado con índices)
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_audit_logs_user_created
		ON audit_logs(user_id, accion, created_at DESC)
	`).Error; err != nil {
		log.Printf("   ⚠️  Error creando idx_audit_logs_user_created: %v", err)
	}

	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_audit_logs_entity_created
		ON audit_logs(entidad, entity_id, created_at DESC)
	`).Error; err != nil {
		log.Printf("   ⚠️  Error creando idx_audit_logs_entity_created: %v", err)
	}

	log.Println("✅ Índices creados exitosamente")
	return nil
}

// seedInitialData - Inserta datos iniciales (categorías)
func seedInitialData(db *gorm.DB) error {
	log.Println("🌱 Insertando datos semilla...")

	// Verificar si ya existen categorías
	var count int64
	if err := db.Model(&models.Category{}).Count(&count).Error; err != nil {
		return err
	}

	// Si ya hay categorías, no insertar
	if count > 0 {
		log.Println("   ℹ️  Las categorías ya existen, omitiendo seed")
		return nil
	}

	// Insertar categorías iniciales
	categorias := []models.Category{
		{
			Nombre:     "Residencial",
			Slug:       "residencial",
			Icono:      "home",
			Activo:     true,
			Orden:      1,
			MetaDescripcion: "Propiedades residenciales: casas, departamentos, etc.",
		},
		{
			Nombre:     "Comercial",
			Slug:       "comercial",
			Icono:      "building",
			Activo:     true,
			Orden:      2,
			MetaDescripcion: "Propiedades comerciales: locales, oficinas, galpones.",
		},
		{
			Nombre:     "Industrial",
			Slug:       "industrial",
			Icono:      "factory",
			Activo:     true,
			Orden:      3,
			MetaDescripcion: "Propiedades industriales: fábricas, nave industriales.",
		},
		{
			Nombre:     "Terrenos",
			Slug:       "terrenos",
			Icono:      "land",
			Activo:     true,
			Orden:      4,
			MetaDescripcion: "Terrenos y lotes para venta o alquiler.",
		},
		{
			Nombre:     "Lujo",
			Slug:       "lujo",
			Icono:      "star",
			Activo:     true,
			Orden:      5,
			MetaDescripcion: "Propiedades de lujo y exclusivas.",
		},
	}

	if err := db.Create(&categorias).Error; err != nil {
		return fmt.Errorf("error insertando categorías: %w", err)
	}

	log.Printf("✅ %d categorías insertadas exitosamente", len(categorias))
	return nil
}
