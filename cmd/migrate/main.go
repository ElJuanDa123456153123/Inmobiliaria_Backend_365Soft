package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/database"
)

func main() {
	// Cargar variables de entorno desde .env
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No se encontró archivo .env, usando variables del sistema")
	}

	// Obtener argumento (up / down)
	action := "up"
	if len(os.Args) > 1 {
		action = os.Args[1]
	}

	switch action {
	case "up":
		log.Println("🔄 Ejecutando migraciones...")
		if err := database.AutoMigrate(); err != nil {
			log.Fatalf("❌ Error ejecutando migraciones: %v", err)
		}
		log.Println("✅ Migraciones completadas")
	default:
		log.Printf("❓ Acción desconocida: %s. Usa: up\n", action)
		os.Exit(1)
	}
}
