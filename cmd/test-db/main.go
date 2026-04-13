package main

import (
	"fmt"
	"log"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/database"
)

func main() {
	log.Println("🔍 Probando conexión y migraciones de base de datos...")

	// Ejecutar migraciones
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("❌ Error en migraciones: %v", err)
	}

	// Opcional: Ejecutar seeds
	// Descomentar la siguiente línea para insertar datos de prueba
	// if err := database.SeedAll(); err != nil {
	//     log.Fatalf("❌ Error en seeds: %v", err)
	// }

	fmt.Println("\n✅ Prueba completada exitosamente")
	fmt.Println("\n📊 Tablas creadas:")
	fmt.Println("   - usuarios")
	fmt.Println("   - categorias")
	fmt.Println("   - propiedades")
	fmt.Println("   - fotos_propiedad")
	fmt.Println("   - propiedades_categorias")
	fmt.Println("   - audit_logs")
	fmt.Println("\n📝 Índices creados para optimización")
}
