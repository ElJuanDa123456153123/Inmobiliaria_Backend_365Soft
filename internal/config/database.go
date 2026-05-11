package config

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database - Estructura para la conexión de base de datos
type Database struct {
	DB *gorm.DB
}

// Singleton para la conexión de base de datos
var (
	dbInstance *Database
	dbOnce     sync.Once
	dbMutex    sync.RWMutex
)

// GetDatabase - Obtiene la instancia singleton de la base de datos
func GetDatabase() (*Database, error) {
	var err error
	dbOnce.Do(func() {
		dbInstance, err = NewDatabase()
	})
	return dbInstance, err
}

// NewDatabase - Crea una nueva conexión a PostgreSQL con reintentos
func NewDatabase() (*Database, error) {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	// Obtener URL de la base de datos desde variables de entorno
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		// Valor por defecto para desarrollo
		databaseURL = "postgres://inmobiliaria_user:password@localhost:5432/inmobiliaria?sslmode=disable"
	}

	// Configurar GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	var db *gorm.DB
	var err error

	// Abrir conexión
	db, err = gorm.Open(postgres.Open(databaseURL), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("error conectando a la base de datos: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("error obteniendo conexión SQL: %w", err)
	}

	// Configurar pool de conexiones con valores más conservadores
	sqlDB.SetMaxIdleConns(5)           // Conexiones inactivas máximas
	sqlDB.SetMaxOpenConns(25)          // Conexiones abiertas máximas (reducido de 100)
	sqlDB.SetConnMaxLifetime(5 * time.Minute) // Reutilizar conexiones por máximo 5 minutos

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
