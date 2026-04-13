package database

import (
	"fmt"
	"log"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
)

// SeedCategories - Inserta categorías iniciales
func SeedCategories() error {
	db, err := config.NewDatabase()
	if err != nil {
		return err
	}

	log.Println("🌱 Seeding categorías...")

	// Verificar si ya existen
	var count int64
	db.DB.Model(&models.Category{}).Count(&count)
	if count > 0 {
		log.Println("   ℹ️  Las categorías ya existen")
		return nil
	}

	categorias := []models.Category{
		{
			Nombre:         "Residencial",
			Slug:           "residencial",
			Icono:          "home",
			Activo:         true,
			Orden:          1,
			MetaDescripcion: "Propiedades residenciales: casas, departamentos, etc.",
		},
		{
			Nombre:         "Comercial",
			Slug:           "comercial",
			Icono:          "building",
			Activo:         true,
			Orden:          2,
			MetaDescripcion: "Propiedades comerciales: locales, oficinas, galpones.",
		},
		{
			Nombre:         "Industrial",
			Slug:           "industrial",
			Icono:          "factory",
			Activo:         true,
			Orden:          3,
			MetaDescripcion: "Propiedades industriales: fábricas, naves industriales.",
		},
		{
			Nombre:         "Terrenos",
			Slug:           "terrenos",
			Icono:          "land",
			Activo:         true,
			Orden:          4,
			MetaDescripcion: "Terrenos y lotes para venta o alquiler.",
		},
		{
			Nombre:         "Lujo",
			Slug:           "lujo",
			Icono:          "star",
			Activo:         true,
			Orden:          5,
			MetaDescripcion: "Propiedades de lujo y exclusivas.",
		},
	}

	if err := db.DB.Create(&categorias).Error; err != nil {
		return fmt.Errorf("error insertando categorías: %w", err)
	}

	log.Printf("✅ %d categorías creadas", len(categorias))
	return nil
}

// SeedProperties - Inserta propiedades de prueba (development only)
func SeedProperties() error {
	db, err := config.NewDatabase()
	if err != nil {
		return err
	}

	log.Println("🌱 Seeding propiedades de prueba...")

	// Verificar si ya existen propiedades
	var count int64
	db.DB.Model(&models.Property{}).Count(&count)
	if count > 0 {
		log.Println("   ℹ️  Ya existen propiedades")
		return nil
	}

	// Obtener primer usuario (si existe)
	var user models.User
	if err := db.DB.First(&user).Error; err != nil {
		log.Println("   ⚠️  No hay usuarios, creando usuario de prueba...")
		user = models.User{
			Nombre:  "Usuario Prueba",
			Email:   "test@example.com",
			Rol:     models.RoleAgente,
			Activo:  true,
		}
		if err := db.DB.Create(&user).Error; err != nil {
			return fmt.Errorf("error creando usuario de prueba: %w", err)
		}
	}

	// Crear propiedades de prueba
	propiedades := []models.Property{
		{
			Titulo:        "Hermosa casa en venta en Equipetrol",
			Descripcion:   "Bonita casa con 3 habitaciones, ideal para familia. Jardín amplio, piscina y seguridad 24hs. Cerca a colegios y centros comerciales.",
			TipoPropiedad: models.PropertyTypeCasa,
			Operacion:     models.OperationVenta,
			Estado:        models.StatusActiva,
			Ciudad:        "Santa Cruz",
			Zona:          "Equipetrol",
			Latitud:       ptrFloat64(-17.7641),
			Longitud:      ptrFloat64(-63.1825),
			Precio:        85000,
			Moneda:        "USD",
			MetrajeTotal:  ptrFloat64(250.5),
			MetrajeConstruido: ptrFloat64(180.0),
			NumeroHabitaciones: ptrInt(3),
			NumeroBanos:   ptrInt(2),
			NumeroGarages: ptrInt(2),
			AntiguedadAnos: ptrInt(5),
			TienePiscina:  true,
			TieneJardin:   true,
			TieneTerraza:  false,
			TieneServicio: true,
			TieneSeguridad: true,
			EsAmueblado:   false,
			AceptaMascotas: true,
			UserID:        user.ID,
			Slug:          "hermosa-casa-en-venta-en-equipetrol",
			Destacado:     true,
			NumeroVisitas: 125,
		},
		{
			Titulo:        "Departamento moderno en el centro",
			Descripcion:   "Departamento de 2 dormitorios en el corazón de la ciudad. Ideal para profesionales jóvenes. A estrenar.",
			TipoPropiedad: models.PropertyTypeDepartamento,
			Operacion:     models.OperationAlquiler,
			Estado:        models.StatusActiva,
			Ciudad:        "Santa Cruz",
			Zona:          "Centro",
			Latitud:       ptrFloat64(-17.7841),
			Longitud:      ptrFloat64(-63.1825),
			Precio:        450,
			Moneda:        "USD",
			MetrajeTotal:  ptrFloat64(85.0),
			MetrajeConstruido: ptrFloat64(75.0),
			NumeroHabitaciones: ptrInt(2),
			NumeroBanos:   ptrInt(1),
			NumeroGarages: ptrInt(1),
			AntiguedadAnos: ptrInt(0),
			TienePiscina:  false,
			TieneJardin:   false,
			TieneTerraza:  true,
			TieneServicio: true,
			TieneSeguridad: true,
			EsAmueblado:   true,
			AceptaMascotas: false,
			UserID:        user.ID,
			Slug:          "departamento-moderno-en-el-centro",
			Destacado:     false,
			NumeroVisitas: 45,
		},
	}

	for _, prop := range propiedades {
		if err := db.DB.Create(&prop).Error; err != nil {
			return fmt.Errorf("error creando propiedad: %w", err)
		}
		log.Printf("   ✅ Propiedad creada: %s", prop.Titulo)
	}

	log.Printf("✅ %d propiedades de prueba creadas", len(propiedades))
	return nil
}

// SeedAll - Ejecuta todos los seeds
func SeedAll() error {
	if err := SeedCategories(); err != nil {
		return err
	}

	if err := SeedProperties(); err != nil {
		return err
	}

	log.Println("✅ Todos los seeds completados")
	return nil
}

// Helper functions para crear punteros
func ptrFloat64(v float64) *float64 {
	return &v
}

func ptrInt(v int) *int {
	return &v
}
