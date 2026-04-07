# 📚 Documentación - Sistema de Autenticación

## Índice
- [Overview](#overview)
- [Características](#características)
- [Instalación](#instalación)
- [Configuración](#configuración)
- [API Endpoints](#api-endpoints)
- [Flujo de Autenticación](#flujo-de-autenticación)
- [Google OAuth](#google-oauth)

---

## Overview

Sistema de autenticación completo construido con **Go + Fiber** que incluye:

- ✅ Registro de usuarios con email/contraseña
- ✅ Login con email/contraseña
- ✅ Autenticación con Google OAuth 2.0
- ✅ Tokens JWT con expiración de 24 horas
- ✅ Validación de roles (usuario, agente, inmobiliaria)
- ✅ Middleware de autenticación
- ✅ Base de datos PostgreSQL

---

## Características

### Seguridad
- 🔒 Contraseñas hasheadas con bcrypt
- 🔑 Tokens JWT para autenticación sin estado
- 🛡️ Middleware de protección de rutas
- ✅ Verificación de email (opcional)
- 🚫 Protección contra inyección SQL con GORM

### Gestión de Usuarios
- 👤 3 tipos de roles: Usuario, Agente, Inmobiliaria
- 📱 Teléfono obligatorio para completar perfil
- 🏢 Campos específicos para inmobiliarias (NIT, nombre)
- ✅ Marca de verificación de email

### Google OAuth 2.0
- 🔐 Autenticación segura con Google
- 🔄 Creación automática de usuarios
- 📝 Completar perfil post-autenticación
- 🎯 Soporte para múltiples orígenes

---

## Instalación

### Prerrequisitos
- Go 1.26 o superior
- PostgreSQL 15+
- Docker y Docker Compose (opcional)

### Dependencias
```bash
go get -u github.com/gofiber/fiber/v2
go get -u github.com/golang-jwt/jwt/v5
go get -u golang.org/x/crypto/bcrypt
go get -u gorm.io/gorm
go get -u gorm.io/driver/postgres
go get -u golang.org/x/oauth2
go get -u golang.org/x/oauth2/google
go get -u github.com/joho/godotenv
```

### Variables de Entorno

Crea un archivo `.env` en la raíz del proyecto:

```env
# Servidor
PORT=3001
APP_ENV=development

# Base de Datos
DATABASE_URL=postgres://user:password@localhost:5432/dbname?sslmode=disable

# JWT
JWT_SECRET=tu_secreto_super_seguro_cambialo_en_produccion

# Google OAuth
GOOGLE_CLIENT_ID=tu_client_id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=tu_client_secret
GOOGLE_CALLBACK_URL=http://localhost:3001/api/auth/google/callback

# Frontend
FRONTEND_URL=http://localhost:3000
```

### Ejecutar con Docker
```bash
docker compose up --build backend
```

### Ejecutar Localmente
```bash
# Ejecutar migraciones
go run cmd/database/migrate.go

# Iniciar servidor
go run cmd/api/main.go
```

---

## API Endpoints

### Rutas Públicas

#### 1. Registro de Usuario
```http
POST /api/auth/register
Content-Type: application/json

{
  "nombre": "Juan Pérez",
  "email": "juan@example.com",
  "password": "Password123!",
  "telefono": "+59170000000",
  "rol": "usuario"
}
```

**Roles válidos:**
- `"usuario"` - Usuario regular
- `"agente"` - Agente inmobiliario
- `"inmobiliaria"` - Inmobiliaria (requiere `nombre_inmobiliaria` y `nit`)

**Respuesta Exitosa (201):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "nombre": "Juan Pérez",
    "email": "juan@example.com",
    "telefono": "+59170000000",
    "rol": "usuario",
    "activo": true,
    "verificado": false,
    "created_at": "2026-04-07T22:00:00Z"
  }
}
```

#### 2. Login con Email/Contraseña
```http
POST /api/auth/login
Content-Type: application/json

{
  "email": "juan@example.com",
  "password": "Password123!"
}
```

**Respuesta Exitosa (200):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "nombre": "Juan Pérez",
    "email": "juan@example.com",
    "rol": "usuario"
  }
}
```

#### 3. Login con Google
```http
GET /api/auth/google
```

Redirige a Google para autenticación. Después de autenticar, Google redirige a:
```http
GET /api/auth/google/callback?code=...&state=...
```

**Respuesta:**
Redirección a:
- `/auth/success?token=jwt&nombre=xxx` (si perfil completo)
- `/auth/complete-profile?token=jwt&nombre=xxx&email=xxx` (si falta teléfono)

### Rutas Protegidas

Todas estas rutas requieren el header `Authorization: Bearer <token>`

#### 4. Obtener Usuario Actual
```http
GET /api/auth/me
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Respuesta (200):**
```json
{
  "id": 1,
  "nombre": "Juan Pérez",
  "email": "juan@example.com",
  "telefono": "+59170000000",
  "rol": "usuario",
  "activo": true,
  "verificado": false
}
```

#### 5. Completar Perfil de Google
```http
POST /api/auth/google/complete-profile
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

{
  "telefono": "+59170000000",
  "rol": "usuario"
}
```

**Respuesta (200):**
```json
{
  "message": "Perfil completado exitosamente",
  "user": {
    "id": 1,
    "nombre": "Juan Pérez",
    "email": "juan@example.com",
    "telefono": "+59170000000",
    "rol": "usuario"
  }
}
```

---

## Flujo de Autenticación

### 1. Registro con Email/Contraseña

```
┌─────────┐      POST /api/auth/register      ┌─────────┐
│ Frontend│ ────────────────────────────────> │ Backend │
└─────────┘                                     └─────────┘
     │                                               │
     │           201 Created + Token JWT              │
     │ <──────────────────────────────────────────────│
     │                                               │
     │    Guardar token en localStorage               │
     │                                               │
     │           Redirigir a /                        │
     └───────────────────────────────────────────────┘
```

### 2. Login con Email/Contraseña

```
┌─────────┐      POST /api/auth/login        ┌─────────┐
│ Frontend│ ────────────────────────────────> │ Backend │
└─────────┘                                     └─────────┘
     │                                               │
     │         Validar credenciales                   │
     │         Generar token JWT                      │
     │                                               │
     │           200 OK + Token JWT                   │
     │ <──────────────────────────────────────────────│
     │                                               │
     │    Guardar token y nombre en localStorage      │
     │                                               │
     │           Redirigir a /                        │
     └───────────────────────────────────────────────┘
```

### 3. Login con Google OAuth

```
┌─────────┐    GET /api/auth/google     ┌─────────┐    Google     ┌──────────┐
│ Frontend│ ──────────────────────────> │ Backend │ ──────────>  │  Google  │
└─────────┘                             └─────────┘              └──────────┘
     │                                          │                      │
     │         Redirección a Google              │                      │
     │ <─────────────────────────────────────────│                      │
     │                                          │                      │
     │  Usuario autentica con Google             │                      │
     └─────────────────────────────────────────────────────────────────>│
                                                                │
                                                                │ Callback con código
                                                                │
     │                                          │   <───────────────│
     │                                          │                      │
     │              Intercambiar código por token│                      │
     │              Obtener info de usuario     │                      │
     │              Crear o actualizar usuario  │                      │
     │              Generar JWT                 │                      │
     │                                          │                      │
     │  Redirección a /auth/success o           │                      │
     │  /auth/complete-profile                  │                      │
     │ <─────────────────────────────────────────│                      │
     │                                          │                      │
     │  Guardar token en localStorage           │                      │
     │  Completar perfil si es necesario         │                      │
     │                                          │                      │
```

---

## Google OAuth

### Configuración en Google Cloud Console

1. **Crear Proyecto**
   - Ve a https://console.cloud.google.com/
   - Crea un nuevo proyecto o selecciona existente

2. **Configurar Pantalla de Consentimiento**
   - APIs & Services → OAuth consent screen
   - Selecciona "External"
   - Completa:
     - App name: "Inmobiliaria 365SOFT"
     - User support email: tu email
     - Developer contact email: tu email

3. **Crear Credenciales OAuth**
   - APIs & Services → Credentials
   - Create Credentials → OAuth client ID
   - Application type: Web application
   - Name: "Inmobiliaria 365SOFT"
   - Authorized JavaScript origins:
     ```
     http://localhost:3000
     ```
   - Authorized redirect URIs:
     ```
     http://localhost:3001/api/auth/google/callback
     ```

4. **Copiar Credenciales**
   - Client ID: `xxxxx.apps.googleusercontent.com`
   - Client Secret: `GOCSPX-xxxxx`

### Configuración en Backend

En `docker-compose.yml`:
```yaml
environment:
  - GOOGLE_CLIENT_ID=xxxxx.apps.googleusercontent.com
  - GOOGLE_CLIENT_SECRET=GOCSPX-xxxxx
  - GOOGLE_CALLBACK_URL=http://localhost:3001/api/auth/google/callback
  - FRONTEND_URL=http://localhost:3000
```

### Usuarios de Prueba

⚠️ **Importante**: Google OAuth en modo prueba solo funciona con usuarios agregados manualmente:

1. OAuth consent screen
2. Scroll hasta "Test users"
3. Click en "ADD USERS"
4. Agrega los emails que pueden probar la app

**Para producción**: Debes verificar tu app con Google.

---

## Estructura del Código

### Handlers
```
internal/handlers/
├── auth.go           # AuthHandler para email/contraseña
└── google_auth.go    # GoogleAuthHandler para OAuth
```

### Middleware
```
internal/middleware/
└── auth.go           # AuthMiddleware para proteger rutas
```

### Models
```
internal/models/
└── user.go           # Modelo User con roles y campos
```

### Rutas
```
cmd/api/main.go       # Configuración de rutas Fiber
```

---

## Errores Comunes

### 1. redirect_uri_mismatch
**Error**: `Error 400: redirect_uri_mismatch`

**Solución**: Verifica que la URL en `GOOGLE_CALLBACK_URL` coincida exactamente con la configurada en Google Cloud Console (incluyendo el puerto).

### 2. Token inválido
**Error**: `Error 401: Token inválido`

**Solución**: Verifica que:
- El token no haya expirado (24 horas)
- El JWT_SECRET sea el mismo en backend y frontend
- El token se envíe en el header `Authorization: Bearer <token>`

### 3. Usuario no encontrado
**Error**: `Error 404: Usuario no encontrado`

**Solución**: El usuario no existe en la base de datos o fue eliminado.

### 4. Credenciales inválidas
**Error**: `Error 401: Credenciales inválidas`

**Solución**: Verifica que el email y password sean correctos.

---

## Testing

### con cURL
```bash
# Registro
curl -X POST http://localhost:3001/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "nombre": "Juan Pérez",
    "email": "juan@example.com",
    "password": "Password123!",
    "telefono": "+59170000000",
    "rol": "usuario"
  }'

# Login
curl -X POST http://localhost:3001/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "juan@example.com",
    "password": "Password123!"
  }'

# Obtener usuario actual (reemplaza TOKEN con el JWT)
curl -X GET http://localhost:3001/api/auth/me \
  -H "Authorization: Bearer TOKEN"
```

### con Postman
1. Importa la colección de endpoints
2. Configura la variable `base_url` = `http://localhost:3001`
3. Para rutas protegidas, agrega header `Authorization` con valor `Bearer {{token}}`

---

## Seguridad

### Best Practices Implementadas
✅ Contraseñas hasheadas con bcrypt (cost 10)
✅ Tokens JWT con expiración (24 horas)
✅ Validación de email único
✅ Middleware de autenticación en rutas protegidas
✅ HTTPS obligatorio en producción
✅ Variables de entorno para secrets
✅ Protección contra inyección SQL (GORM)

### Recomendaciones Adicionales
🔒 Usar HTTPS en producción
📧 Implementar verificación de email
🔄 Implementar refresh tokens
🚫 Rate limiting en endpoints de login
📝 Logging de intentos fallidos
🔐 Validación de fuerza de contraseña

---

## Soporte

Para problemas o preguntas:
- Revisa los logs del backend: `docker compose logs backend`
- Verifica las variables de entorno
- Asegúrate que PostgreSQL esté corriendo
- Confirma que las credenciales de Google sean correctas

---

**Última actualización**: 7 de abril de 2026
**Versión**: 1.0.0
