package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database - Estructura para la conexión de base de datos
type Database struct {
	DB *gorm.DB
}

// NewDatabase - Crea una nueva conexión a PostgreSQL con reintentos
func NewDatabase() (*Database, error) {
	// Obtener URL de la base de datos desde variables de entorno
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		// Valor por defecto para desarrollo
		databaseURL = "postgres://inmobiliaria_user:password@localhost:5432/inmobiliaria?sslmode=disable"
	}

	// Configurar GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	// Configurar reintentos
	maxRetries := 10
	retryDelay := 3 * time.Second

	var db *gorm.DB
	var err error

	// Intentar conectar con reintentos
	for i := 0; i < maxRetries; i++ {
		// Abrir conexión
		db, err = gorm.Open(postgres.Open(databaseURL), gormConfig)
		if err == nil {
			// Conexión exitosa
			break
		}

		// Log del intento fallido
		if i < maxRetries-1 {
			log.Printf("⏳ Intento %d/%d: No se pudo conectar a PostgreSQL. Reintentando en %v...",
				i+1, maxRetries, retryDelay)
			log.Printf("   Error: %v", err)
			time.Sleep(retryDelay)
		}
	}

	// Verificar si finalmente se pudo conectar
	if err != nil {
		return nil, fmt.Errorf("error conectando a la base de datos después de %d intentos: %w", maxRetries, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("error obteniendo conexión SQL: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	log.Println("✅ Conectado a PostgreSQL exitosamente")
	return &Database{DB: db}, nil
}

// Close - Cierra la conexión a la base de datos
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// NewDatabaseFromDSN - Crea una nueva conexión a PostgreSQL desde un DSN
func NewDatabaseFromDSN(dsn string) (*Database, error) {
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Silencioso en tests
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("error conectando a la base de datos: %w", err)
	}

	// Configurar pool de conexiones
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("error obteniendo conexión SQL: %w", err)
	}

	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(20)

	return &Database{DB: db}, nil
}
