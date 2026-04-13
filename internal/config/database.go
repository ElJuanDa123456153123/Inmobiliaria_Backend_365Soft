package config

import (
	"fmt"
	"log"
	"os"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database - Estructura para la conexión de base de datos
type Database struct {
	DB *gorm.DB
}
var (
	dbInstance *Database
	dbErr      error
	dbOnce     sync.Once
)

// NewDatabase - Crea y retorna una única conexión a PostgreSQL (Singleton)
func NewDatabase() (*Database, error) {
	dbOnce.Do(func() {

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

		db, err := gorm.Open(postgres.Open(databaseURL), gormConfig)
		if err != nil {
			dbErr = fmt.Errorf("error conectando a la base de datos: %w", err)
			return
		}

		sqlDB, err := db.DB()
		if err != nil {
			dbErr = fmt.Errorf("error obteniendo conexión SQL: %w", err)
			return
		}

		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)

		log.Println("✅ Conectado a PostgreSQL exitosamente")
		dbInstance = &Database{DB: db}
	})

	return dbInstance, dbErr
}

// Close - Cierra la conexión a la base de datos
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
