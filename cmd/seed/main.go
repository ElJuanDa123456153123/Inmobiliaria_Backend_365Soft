package main

import (
	"log"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
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

	// Ejecutar seed
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
	log.Printf("✅ Seeders completados: %d creados, %d existentes", created, skipped)
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// seedUserRecord crea un usuario si no existe, retorna true si fue creado
func seedUserRecord(db *gorm.DB, u seedUser) (bool, error) {
	// Verificar si ya existe
	var existing models.User
	result := db.Where("email = ?", u.email).First(&existing)
	if result.Error == nil {
		log.Printf("  ⏭️  Ya existe: %s (%s)", existing.Email, existing.Rol)
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

	log.Printf("  ✅ Creado: %s (%s) | password: %s", user.Email, user.Rol, u.password)
	return true, nil
}
