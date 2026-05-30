package main

import (
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/database"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
)

// seedUser define la estructura para los datos de seed
type seedUser struct {
	nombre    string
	email     string
	password  string
	telefono  string
	rol       models.UserRole
	inmob     string
	nit       string
}

func main() {
	// Cargar variables de entorno
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No se encontró archivo .env, usando variables del sistema")
	}

	log.Println("🌱 Ejecutando seeders...")

	// Ejecutar migraciones automáticamente
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("❌ Error ejecutando migraciones: %v", err)
	}

	// Conectar a la base de datos
	db, err := config.NewDatabase()
	if err != nil {
		log.Fatalf("❌ Error conectando a la base de datos: %v", err)
	}

	// Definir todos los usuarios seed
	users := []seedUser{
		// ——— Administrador del sistema ———
		{
			nombre:   "Administrador",
			email:    "admin@inmobiliaria.bo",
			password: "superadmin",
			telefono: "+591 70000000",
			rol:      models.RoleAdmin,
		},
		// ——— Usuarios de prueba ———
		{
			nombre:   "Agente Demo",
			email:    "agente@inmobiliaria.bo",
			password: "agente123",
			telefono: "+591 71111111",
			rol:      models.RoleAgente,
		},
		{
			nombre:   "Inmobiliaria ABC",
			email:    "inmobiliaria@test.com",
			password: "Inmob123!",
			telefono: "+591 72222222",
			rol:      models.RoleInmobiliaria,
			inmob:    "Inmobiliaria ABC S.R.L.",
			nit:      "1234567890",
		},
		{
			nombre:   "Usuario Demo",
			email:    "usuario@test.com",
			password: "User123!",
			telefono: "+591 73333333",
			rol:      models.RoleUsuario,
		},
	}

	// Ejecutar seed de usuarios
	created := 0
	skipped := 0
	for _, u := range users {
		wasCreated, err := seedUserRecord(db.DB, u)
		if err != nil {
			log.Fatalf("❌ Error creando usuario %s: %v", u.email, err)
		}
		if wasCreated {
			created++
		} else {
			skipped++
		}
	}

	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("✅ Usuarios: %d creados, %d existentes", created, skipped)
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Ejecutar seed de propiedades y pagos
	if err := seedPropertiesAndPayments(db.DB); err != nil {
		log.Fatalf("❌ Error sembrando propiedades y pagos: %v", err)
	}
}

// seedUserRecord crea un usuario si no existe, retorna true si fue creado
func seedUserRecord(db *gorm.DB, u seedUser) (bool, error) {
	// Verificar si ya existe
	var existing models.User
	result := db.Where("email = ?", u.email).First(&existing)
	if result.Error == nil {
		log.Printf("  ⏭️  Ya existe usuario: %s (%s)", existing.Email, existing.Rol)
		return false, nil
	}

	// Hashear password con bcrypt
	hashed, err := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)
	if err != nil {
		return false, err
	}

	user := models.User{
		Nombre:             u.nombre,
		Email:              u.email,
		Password:           string(hashed),
		Telefono:           u.telefono,
		Rol:                u.rol,
		NombreInmobiliaria: u.inmob,
		NIT:                u.nit,
		Activo:             true,
		Verificado:         true,
	}

	if err := db.Create(&user).Error; err != nil {
		return false, err
	}

	log.Printf("  ✅ Creado usuario: %s (%s) | password: %s", user.Email, user.Rol, u.password)
	return true, nil
}

// seedPropertiesAndPayments siembra propiedades y pagos simulados
func seedPropertiesAndPayments(db *gorm.DB) error {
	// 1. Obtener usuario agente para asociarle las propiedades
	var agent models.User
	if err := db.Where("rol = ?", models.RoleAgente).First(&agent).Error; err != nil {
		return fmt.Errorf("no se encontró un agente para asociar propiedades: %w", err)
	}

	// 2. Definir tipos de propiedades y operaciones
	tiposPropiedad := []models.PropertyType{
		models.PropertyTypeCasa,
		models.PropertyTypeDepartamento,
		models.PropertyTypeTerreno,
		models.PropertyTypeOficina,
		models.PropertyTypeLocal,
	}

	operaciones := []models.OperationType{
		models.OperationTypeVenta,
		models.OperationTypeAlquiler,
		models.OperationTypeAnticretico,
	}

	zonas := []string{"Equipetrol", "Urbarí", "Las Palmas", "Zona Sur", "Achumani", "Calacoto", "Sopocachi", "San Pedro"}
	titulos := []string{
		"Hermosa Casa de Lujo en Zona Residencial",
		"Departamento Amplio y Moderno de 3 Dormitorios",
		"Terreno Ideal para Edificio de Oficinas",
		"Oficina Céntrica y Totalmente Equipada",
		"Local Comercial sobre Avenida Principal",
	}

	// Sembrar propiedades si hay pocas
	var countProps int64
	db.Model(&models.Property{}).Count(&countProps)
	if countProps > 10 {
		log.Println("  ⏭️  Ya existen suficientes propiedades, omitiendo sembrado de propiedades.")
	} else {
		log.Println("🌱 Sembrando propiedades...")
		now := time.Now()
		
		for i := 0; i < 30; i++ {
			createdDate := now.AddDate(0, 0, -i)
			tipoProp := tiposPropiedad[i%len(tiposPropiedad)]
			op := operaciones[i%len(operaciones)]
			zona := zonas[i%len(zonas)]
			titulo := fmt.Sprintf("%s en %s", titulos[i%len(titulos)], zona)
			
			status := models.PropertyStatusActivo
			if i%5 == 0 {
				status = models.PropertyStatusPendiente
			} else if i%8 == 0 {
				status = models.PropertyStatusInactivo
			}
			
			precio := float64(150000 + (i * 12345))
			if op == models.OperationTypeAlquiler {
				precio = float64(450 + (i * 75))
			}

			prop := models.Property{
				UserID:          agent.ID,
				Titulo:          titulo,
				TipoOperacion:   op,
				Precio:          precio,
				Moneda:          "BOB",
				TipoPropiedad:   tipoProp,
				Direccion:       fmt.Sprintf("Av. Busch Calle %d Nro %d", i+1, i*10),
				Zona:            zona,
				Descripcion:     fmt.Sprintf("Esta es una descripción detallada para la propiedad nro %d. Ubicada en excelente zona con transporte público, seguridad y servicios básicos.", i),
				EstadoPropiedad: status,
				VistasContador:  (i * 17) + 5,
				CreatedAt:       createdDate,
				UpdatedAt:       createdDate,
			}
			
			if err := db.Create(&prop).Error; err != nil {
				log.Printf("❌ Error creando propiedad %d: %v", i, err)
			}
		}
		log.Println("✅ 30 propiedades sembradas exitosamente.")
	}

	// Sembrar pagos
	var countPayments int64
	db.Model(&models.Payment{}).Count(&countPayments)
	if countPayments > 5 {
		log.Println("  ⏭️  Ya existen pagos, omitiendo sembrado de pagos.")
	} else {
		log.Println("🌱 Sembrando pagos...")
		now := time.Now()
		metodos := []string{"qr_bisa", "transferencia", "tarjeta"}
		
		for i := 0; i < 24; i++ {
			monthOffset := i / 2
			paymentDate := time.Date(now.Year(), now.Month()-time.Month(monthOffset), 15-(i%2)*5, 12, 0, 0, 0, now.Location())
			
			monto := float64(250 + (i * 50))
			metodo := metodos[i%len(metodos)]
			
			pay := models.Payment{
				UserID:         agent.ID,
				Monto:          monto,
				Moneda:         "BOB",
				Metodo:         metodo,
				Estado:         "completado",
				ReferenciaPago: fmt.Sprintf("PAY-REF-%d-%d", monthOffset, i),
				FechaPago:      &paymentDate,
				Notas:          fmt.Sprintf("Pago de prueba para el mes offset %d", monthOffset),
				CreatedAt:      paymentDate,
				UpdatedAt:      paymentDate,
			}
			
			if err := db.Create(&pay).Error; err != nil {
				log.Printf("❌ Error creando pago %d: %v", i, err)
			}
		}
		log.Println("✅ 24 pagos sembrados exitosamente.")
	}

	return nil
}
