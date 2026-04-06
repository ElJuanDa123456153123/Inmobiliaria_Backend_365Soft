# 🏠 Plataforma Inmobiliaria — Backend

Backend de la plataforma inmobiliaria boliviana con publicación vía WhatsApp, CRM integrado, mapa interactivo y gestión de suscripciones.

## 🚀 Stack Tecnológico

| Capa | Tecnología |
|------|------------|
| Lenguaje | Go 1.22+ |
| Framework | Fiber v2 |
| Base de datos | PostgreSQL 15+ con PostGIS |
| Caché / Estado | Redis 7+ |
| Chatbot | Evolution API + n8n + Gemini Flash |
| Fotos | Cloudflare R2 (compatible S3) |
| Pagos | QR boliviano + Wompi |
| Contenedores | Docker + Docker Compose |

## 📋 Requisitos Previos

- **Go** 1.22+
- **PostgreSQL** 15+ con extensión PostGIS
- **Redis** 7+
- **Docker** y **Docker Compose**
- Cuenta en Cloudflare R2
- Cuenta en Wompi (opcional para pagos automáticos)

## 🆕 Guía de Instalación desde Cero (Sin Plantillas)

Si quieres crear el proyecto desde cero sin usar plantillas, sigue estos pasos basados en la documentación oficial de cada tecnología.

### 1. Instalar Go

**Documentación oficial:** https://go.dev/doc/install

**Windows:**
```bash
# Descargar instalador desde https://go.dev/dl/
# O usar Chocolatey
choco install golang

# Verificar instalación
go version  # debe ser go1.22+
```

**Linux (Ubuntu/Debian):**
```bash
# Descargar Go 1.22+
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz

# Extraer e instalar
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz

# Agregar al PATH (agregar a ~/.bashrc o ~/.zshrc)
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

# Recargar shell
source ~/.bashrc  # o source ~/.zshrc

# Verificar instalación
go version
```

**macOS:**
```bash
# Usar Homebrew
brew install go

# Verificar instalación
go version
```

### 2. Crear proyecto Go con Fiber

**Documentación oficial Fiber:** https://docs.gofiber.io/

**Documentación oficial Go Modules:** https://go.dev/doc/modules/manage-code

```bash
# Crear carpeta del proyecto
mkdir inmobiliaria-backend
cd inmobiliaria-backend

# Inicializar módulo Go
go mod init github.com/365Soft-Bolivia/inmobiliaria-backend

# Instalar Fiber v2
go get github.com/gofiber/fiber/v2

# Instalar dependencias adicionales
go get github.com/gofiber/fiber/v2/middleware/cors
go get github.com/gofiber/fiber/v2/middleware/logger
go get github.com/gofiber/fiber/v2/middleware/recover
go get github.com/gofiber/fiber/v2/middleware/limiter
```

**Estructura mínima de carpetas:**
```bash
mkdir -p cmd/api internal/{modules,shared}
mkdir -p internal/shared/{config,middleware,utils}
mkdir -p internal/database/{migrations,models}
```

**Archivo `cmd/api/main.go`:**
```go
package main

import (
    "log"
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/logger"
    "github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
    app := fiber.New(fiber.Config{
        AppName: "Inmobiliaria API v1.0",
    })

    // Middleware
    app.Use(logger.New())
    app.Use(recover.New())

    // Rutas básicas
    app.Get("/health", func(c *fiber.Ctx) error {
        return c.JSON(fiber.Map{
            "status": "ok",
            "message": "API is running",
        })
    })

    // Iniciar servidor
    log.Fatal(app.Listen(":8080"))
}
```

**Probar el servidor:**
```bash
go run cmd/api/main.go

# En otra terminal, probar:
curl http://localhost:8080/health
```

### 3. Configurar PostgreSQL con PostGIS

**Documentación oficial PostgreSQL:** https://www.postgresql.org/docs/

**Documentación oficial PostGIS:** https://postgis.net/documentation/

**Opción A: Instalar localmente**
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install postgresql-15 postgresql-15-postgis-3 postgresql-client-15

# Iniciar servicio
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Crear base de datos y usuario
sudo -u postgres psql

# Dentro de psql:
CREATE DATABASE inmobiliaria;
CREATE USER inmobiliaria_user WITH PASSWORD 'tu_password';
GRANT ALL PRIVILEGES ON DATABASE inmobiliaria TO inmobiliaria_user;
\c inmobiliaria
CREATE EXTENSION postgis;
\q
```

**Opción B: Usar Docker (Recomendado)**
```bash
# Crear docker-compose.yml para desarrollo
cat > docker-compose.yml << 'EOF'
version: '3.8'
services:
  postgres:
    image: postgis/postgis:15-3.3
    container_name: inmobiliaria-postgres
    environment:
      POSTGRES_USER: inmobiliaria_user
      POSTGRES_PASSWORD: tu_password
      POSTGRES_DB: inmobiliaria
    ports:
      - "5432:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    container_name: inmobiliaria-redis
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data

volumes:
  postgres-data:
  redis-data:
EOF

# Iniciar servicios
docker-compose up -d

# Verificar que están corriendo
docker-compose ps
```

### 4. Instalar driver de PostgreSQL para Go

```bash
# Docs: https://github.com/lib/pq
go get github.com/lib/pq

# Para consultas más fáciles (opcional pero recomendado)
go get github.com/jmoiron/sqlx
```

**Ejemplo de conexión:**
```go
// internal/shared/config/database.go
package config

import (
    "database/sql"
    "fmt"
    _ "github.com/lib/pq"
)

func ConnectDB() (*sql.DB, error) {
    connStr := "postgres://inmobiliaria_user:tu_password@localhost:5432/inmobiliaria?sslmode=disable"

    db, err := sql.Open("postgres", connStr)
    if err != nil {
        return nil, err
    }

    if err = db.Ping(); err != nil {
        return nil, err
    }

    fmt.Println("Conectado a PostgreSQL")
    return db, nil
}
```

### 5. Instalar dependencias adicionales

**JWT (Autenticación):**
```bash
# Docs: https://github.com/golang-jwt/jwt
go get github.com/golang-jwt/jwt/v5
```

**Dotenv (Variables de entorno):**
```bash
# Docs: https://github.com/joho/godotenv
go get github.com/joho/godotenv
```

**Validación:**
```bash
# Docs: https://github.com/go-playground/validator
go get github.com/go-playground/validator/v10
```

**UUID:**
```bash
# Docs: https://github.com/google/uuid
go get github.com/google/uuid
```

**Cliente HTTP:**
```bash
# Docs: https://github.com/imroc/req
go get github.com/imroc/req/v3
```

**Librería para QR:**
```bash
# Docs: https://github.com/skip2/go-qrcode
go get github.com/skip2/go-qrcode
```

### 6. Configurar Air (Hot Reload)

**Documentación oficial:** https://github.com/air-verse/air

```bash
# Instalar Air
go install github.com/air-verse/air@latest

# Crear configuración .air.toml
cat > .air.toml << 'EOF'
root = "."
tmp_dir = "tmp"

[build]
  cmd = "go build -o ./tmp/main ./cmd/api"
  bin = "tmp/main"
  include_ext = ["go", "tpl", "tmpl", "html"]
  exclude_dir = ["tmp", "vendor"]
  include_dir = []
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  args_bin = []
EOF

# Ejecutar con hot reload
air
```

### 7. Probar la instalación completa

```bash
# Asegurarse de que PostgreSQL y Redis estén corriendo
docker-compose ps

# Ejecutar el proyecto
go run cmd/api/main.go

# O con hot reload
air

# Probar endpoint
curl http://localhost:8080/health
```

### 8. Crear archivo .env.example

```bash
# Crear archivo de ejemplo
cat > .env.example << 'EOF'
# Servidor
PORT=8080
APP_ENV=development

# Base de datos
DATABASE_URL=postgres://inmobiliaria_user:tu_password@localhost:5432/inmobiliaria?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379

# JWT
JWT_SECRET=tu_secreto_super_seguro
EOF
```

## 🔧 Instalación (Desde Repositorio)

```bash
# Clonar el repositorio
git clone https://github.com/tu-usuario/inmobiliaria-backend.git
cd inmobiliaria-backend

# Instalar dependencias de Go
go mod tidy

# Copiar variables de entorno
cp .env.example .env

# Editar .env con tus credenciales
```

## ⚙️ Variables de Entorno

```env
# Servidor
PORT=8080
APP_ENV=development

# Base de datos
DATABASE_URL=postgres://usuario:password@localhost:5432/inmobiliaria?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379

# JWT
JWT_SECRET=tu_secreto_super_seguro

# Cloudflare R2
R2_ACCOUNT_ID=...
R2_ACCESS_KEY_ID=...
R2_SECRET_ACCESS_KEY=...
R2_BUCKET_NAME=inmobiliaria-fotos

# Evolution API (WhatsApp)
EVOLUTION_API_URL=http://localhost:8080
EVOLUTION_API_KEY=...

# Wompi (Pagos)
WOMPI_PUBLIC_KEY=...
WOMPI_PRIVATE_KEY=...

# Gemini Flash
GEMINI_API_KEY=...
```

## 🗄️ Configurar Base de Datos

```bash
# Levantar PostgreSQL con PostGIS via Docker
docker-compose up -d postgres redis

# Ejecutar migraciones
go run ./cmd/migrate/main.go up

# Seed de datos de prueba
go run ./cmd/seed/main.go
```

### 🔑 Credenciales de Prueba (después del seed)

| Email | Password | Rol |
|-------|----------|-----|
| admin@inmobiliaria.bo | superadmin | ADMIN |
| agente@inmobiliaria.bo | agente123 | AGENTE |

## 🎯 Modos de Ejecución

### Desarrollo

```bash
go run ./cmd/api/main.go
# Servidor en http://localhost:8080
```

### Con hot reload (Air)

```bash
# Instalar Air
go install github.com/air-verse/air@latest

# Ejecutar
air
```

### Producción

```bash
# Build
go build -o bin/api ./cmd/api/main.go

# Start
./bin/api
```

### Docker Compose (recomendado)

```bash
# Levantar todos los servicios
docker-compose up -d

# Ver logs
docker-compose logs -f api
```

## 📁 Estructura del Proyecto

```
.
├── cmd/
│   ├── api/           # Entry point del servidor
│   ├── migrate/       # Migraciones de base de datos
│   └── seed/          # Datos de prueba
├── internal/
│   ├── modules/
│   │   ├── auth/          # Autenticación JWT
│   │   ├── propiedades/   # Gestión de propiedades
│   │   ├── usuarios/      # Gestión de usuarios y CRM
│   │   ├── busqueda/      # Búsqueda con filtros + PostGIS
│   │   ├── chatbot/       # Integración Evolution API / n8n
│   │   ├── pagos/         # QR + Wompi
│   │   ├── suscripciones/ # Planes y renovaciones
│   │   └── admin/         # Panel de administración
│   ├── shared/
│   │   ├── config/        # Configuración (DB, Redis, R2)
│   │   ├── middleware/    # Auth, CORS, rate limit
│   │   └── utils/         # JWT, QR, imágenes
│   └── database/
│       ├── migrations/    # Archivos SQL de migración
│       └── models/        # Modelos de datos
├── docker-compose.yml
├── .env.example
└── README.md
```

## 🔌 API Endpoints

### Autenticación

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| POST | `/api/auth/register` | Registro de usuario |
| POST | `/api/auth/login` | Inicio de sesión |
| GET | `/api/auth/me` | Usuario actual |

### Propiedades

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/api/propiedades` | Listar con filtros |
| GET | `/api/propiedades/:id` | Detalle de propiedad |
| POST | `/api/propiedades` | Crear propiedad (autenticado) |
| PUT | `/api/propiedades/:id` | Editar propiedad |
| DELETE | `/api/propiedades/:id` | Eliminar propiedad |

### Búsqueda

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/api/busqueda` | Búsqueda con filtros avanzados |
| GET | `/api/busqueda/mapa` | Propiedades para mapa (GeoJSON) |

### Chatbot / WhatsApp

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| POST | `/api/chatbot/webhook` | Webhook de Evolution API |
| POST | `/api/chatbot/estado` | Actualizar estado de conversación |

### Favoritos

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| POST | `/api/favoritos/:id` | Guardar favorito |
| DELETE | `/api/favoritos/:id` | Quitar favorito |
| GET | `/api/favoritos` | Mis favoritos |

### Pagos

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| POST | `/api/pagos/qr` | Generar QR de pago |
| POST | `/api/pagos/webhook` | Webhook de confirmación Wompi |
| GET | `/api/pagos/historial` | Historial de pagos |

### Admin

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/api/admin/dashboard` | Métricas generales |
| GET | `/api/admin/propiedades` | Cola de revisión |
| PUT | `/api/admin/propiedades/:id` | Aprobar / rechazar / pausar |
| GET | `/api/admin/usuarios` | Gestión de usuarios |
| GET | `/api/admin/suscripciones` | Historial de suscripciones |

## 📁 Scripts Disponibles

```bash
go run ./cmd/api/main.go          # Servidor desarrollo
go run ./cmd/migrate/main.go up   # Ejecutar migraciones
go run ./cmd/migrate/main.go down # Revertir última migración
go run ./cmd/seed/main.go         # Cargar datos de prueba
go build -o bin/api ./cmd/api     # Compilar para producción
go test ./...                     # Ejecutar tests
```

## 🚀 Deploy en Contabo VPS con Docker

### Arquitectura en el servidor

```
Contabo VPS (Cloud VPS 20 — 6 vCPU / 12 GB RAM recomendado)
└── Docker Compose
    ├── nginx          ← Reverse proxy + SSL (puerto 80/443)
    ├── api            ← Go + Fiber (puerto 8080)
    ├── postgres       ← PostgreSQL + PostGIS (puerto 5432)
    ├── redis          ← Caché + estado chatbot (puerto 6379)
    ├── evolution-api  ← WhatsApp (puerto 8081)
    ├── n8n            ← Automatizaciones (puerto 5678)
    └── chatwoot       ← Bandeja de mensajes (puerto 3000)
```

### Pasos de deploy

```bash
# 1. Conectarse al VPS
ssh root@IP_DE_CONTABO

# 2. Instalar Docker y Docker Compose
apt update && apt upgrade -y
apt install -y docker.io docker-compose git

# 3. Clonar el repositorio
git clone https://github.com/tu-usuario/inmobiliaria-api.git
cd inmobiliaria-api

# 4. Configurar variables de entorno
cp .env.example .env
nano .env   # Completar con credenciales reales

# 5. Levantar todos los servicios
docker-compose -f docker-compose.prod.yml up -d

# 6. Ejecutar migraciones dentro del contenedor
docker-compose exec api go run ./cmd/migrate/main.go up

# 7. Ver que todo esté corriendo
docker-compose ps
docker-compose logs -f api
```

### Configurar SSL con Let's Encrypt

```bash
# Instalar Certbot
apt install -y certbot python3-certbot-nginx

# Generar certificado
certbot --nginx -d api.tudominio.com

# Renovación automática (ya se configura sola)
certbot renew --dry-run
```

### Actualizar en producción

```bash
# Traer últimos cambios
git pull origin main

# Rebuildar y reiniciar solo el servicio api
docker-compose -f docker-compose.prod.yml up -d --build api

# Ver logs en tiempo real
docker-compose logs -f api
```

## 📚 Recursos

- [Go](https://go.dev/)
- [Fiber](https://gofiber.io/)
- [PostgreSQL](https://www.postgresql.org/)
- [PostGIS](https://postgis.net/)
- [Redis](https://redis.io/)
- [Evolution API](https://doc.evolution-api.com/)
- [Cloudflare R2](https://www.cloudflare.com/products/r2/)

## 👨‍💻 Autor

**365Soft**

## 📄 Licencia

MIT
