package database

import (
	"fmt"
	"log"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
)

// SetupFullTextSearch - Configura búsqueda full-text en PostgreSQL
func SetupFullTextSearch() error {
	db, err := config.NewDatabase()
	if err != nil {
		return fmt.Errorf("error conectando a la base de datos: %w", err)
	}
	defer db.Close()

	log.Println("🔍 Configurando búsqueda full-text en PostgreSQL...")

	// Crear columna tsvector para búsqueda
	sql := `
		-- Agregar columna para búsqueda full-text
		ALTER TABLE propiedades
		ADD COLUMN IF NOT EXISTS search_vector tsvector;

		-- Crear índice GIN para búsqueda rápida
		CREATE INDEX IF NOT EXISTS idx_propiedades_search_vector
		ON propiedades USING GIN(search_vector);

		-- Crear función para actualizar el search_vector
		CREATE OR REPLACE FUNCTION propiedades_search_vector_update() RETURNS trigger AS $$
		BEGIN
			NEW.search_vector :=
				setweight(to_tsvector('spanish', COALESCE(NEW.titulo, '')), 'A') ||
				setweight(to_tsvector('spanish', COALESCE(NEW.descripcion, '')), 'B') ||
				setweight(to_tsvector('spanish', COALESCE(NEW.zona, '')), 'B') ||
				setweight(to_tsvector('spanish', COALESCE(NEW.tipo_propiedad, '')), 'C') ||
				setweight(to_tsvector('spanish', COALESCE(NEW.estado, '')), 'C') ||
				setweight(to_tsvector('spanish', COALESCE(NEW.direccion, '')), 'D');
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		-- Crear trigger para actualizar search_vector automáticamente
		DROP TRIGGER IF EXISTS tsvector_update ON propiedades;
		CREATE TRIGGER tsvector_update
		BEFORE INSERT OR UPDATE ON propiedades
		FOR EACH ROW
		EXECUTE FUNCTION propiedades_search_vector_update();

		-- Actualizar todas las propiedades existentes
		UPDATE propiedades
		SET search_vector =
			setweight(to_tsvector('spanish', COALESCE(titulo, '')), 'A') ||
			setweight(to_tsvector('spanish', COALESCE(descripcion, '')), 'B') ||
			setweight(to_tsvector('spanish', COALESCE(zona, '')), 'B') ||
			setweight(to_tsvector('spanish', COALESCE(tipo_propiedad, '')), 'C') ||
			setweight(to_tsvector('spanish', COALESCE(estado, '')), 'C') ||
			setweight(to_tsvector('spanish', COALESCE(direccion, '')), 'D');
	`

	if err := db.DB.Exec(sql).Error; err != nil {
		return fmt.Errorf("error configurando búsqueda full-text: %w", err)
	}

	log.Println("✅ Búsqueda full-text configurada exitosamente")
	return nil
}
