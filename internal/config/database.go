package config

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database - Estructura para la conexión de base de datos
type Database struct {
	DB *gorm.DB
}

// NewDatabase - Crea una nueva conexión a PostgreSQL
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

	// Abrir conexión
	db, err := gorm.Open(postgres.Open(databaseURL), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("error conectando a la base de datos: %w", err)
	}

	// Configurar pool de conexiones
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("error obteniendo conexión SQL: %w", err)
	}

	// Configurar pool
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
