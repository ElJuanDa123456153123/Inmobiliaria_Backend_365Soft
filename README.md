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

---

## 🚀 ¿Cuál sección seguir?

- **Si vas a CLONAR el repositorio** (equipo de desarrollo, despliegue): → Ve a la sección **"📥 Instalación desde Repositorio"**
- **Si quieres crear el proyecto DESDE CERO** (aprendizaje, nuevo proyecto): → Ve a la sección **"🔧 Crear desde Cero"**

---

## 🔧 Crear desde Cero

```bash
# 1. Instalar Go 1.26+
# Descargar desde https://go.dev/dl/

# 2. Crear proyecto
mkdir inmobiliaria-backend
cd inmobiliaria-backend
go mod init github.com/365Soft-Bolivia/inmobiliaria-backend

# 3. Instalar Fiber
go get github.com/gofiber/fiber/v2

# 4. Crear estructura
mkdir -p cmd/api internal/{shared/{config,middleware,utils},database}

# 5. Crear main.go en cmd/api/

# 6. Iniciar
go run cmd/api/main.go
```

## 📥 Instalación desde Repositorio

**Esta es la sección que deben seguir los desarrolladores del equipo.**

⚠️ **IMPORTANTE:** Para evitar problemas con las rutas en Docker Compose, sigue estos pasos exactamente:

```bash
# 1. Crear carpeta principal del proyecto
mkdir Inmobiliaria365Soft
cd Inmobiliaria365Soft

# 2. Clonar el repositorio frontend
git clone https://github.com/365Soft-Bolivia/Inmobiliaria_Frontend_365Soft.git

# 3. Volver a la carpeta raiz
cd ..

# 4. Clonar el repositorio backend
git clone https://github.com/365Soft-Bolivia/Inmobiliaria_Backend_365Soft.git

# 5. Entrar a la carpeta backend
cd Inmobiliaria_Backend_365Soft

# Instalar dependencias de Go
go mod tidy

# Copiar variables de entorno
cp .env.example .env

# Editar .env con tus credenciales
```

**Estructura esperada:**
```
Inmobiliaria365Soft/
├── Inmobiliaria_Frontend_365Soft/
└── Inmobiliaria_Backend_365Soft/
```

Esta estructura es necesaria para que el `docker-compose.yml` funcione correctamente con las rutas relativas de los volúmenes y las redes.

## ⚙️ Variables de Entorno

```env
# Servidor
PORT=3001
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
# Servidor en http://localhost:3001
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

## 🐳 Docker (Desarrollo Completo)

**Requiere Docker y Docker Compose instalados.**

⚠️ **IMPORTANTE:** Asegúrate de tener ambos repositorios clonados en la misma carpeta principal siguiendo las instrucciones de la sección "Instalación desde Repositorio".

```bash
# Desde la carpeta principal Inmobiliaria365Soft
cd Inmobiliaria365Soft

# Levantar todos los servicios (Frontend + Backend + PostgreSQL + Redis)
docker-compose up --build

# Detener servicios
docker-compose down

# Ver logs
docker-compose logs -f backend
```

**Servicios disponibles:**
- Frontend: http://localhost:3000
- Backend: http://localhost:3001
- PostgreSQL: localhost:5432
- Redis: localhost:6379

## 🧪 Probar la Instalación

```bash
# 1. Backend - Probar endpoint de salud
curl http://localhost:3001/health
# Respuesta esperada: {"status":"ok","message":"API is running"}

# 2. Backend - Probar API
curl http://localhost:3001/api
# Respuesta esperada: {"message":"Bienvenido a la API de Inmobiliaria Bolivia"}

# 3. Verificar PostgreSQL (Docker)
docker exec -it inmobiliaria-postgres psql -U inmobiliaria_user -d inmobiliaria
# Comandos útiles:
# \dt (lista tablas)
# \q (salir)

# 4. Verificar Redis (Docker)
docker exec -it inmobiliaria-redis redis-cli ping
# Respuesta esperada: PONG

# 5. Verificar logs de contenedores
docker-compose ps
docker-compose logs backend
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
    ├── api            ← Go + Fiber (puerto 3001)
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
