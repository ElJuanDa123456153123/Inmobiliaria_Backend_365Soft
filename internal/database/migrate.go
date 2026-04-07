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

	// Migrar el modelo User
	err = db.DB.AutoMigrate(&models.User{})
	if err != nil {
		return fmt.Errorf("error ejecutando migración: %w", err)
	}

	log.Println("✅ Migraciones ejecutadas exitosamente")
	return nil
}
