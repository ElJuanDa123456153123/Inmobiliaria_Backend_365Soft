package models

// models.go - Archivo centralizado de registros de modelos
// Este archivo facilita la importación y referencia de todos los modelos

//go:generate go run github.com/markbates/gentran/v3

// Lista completa de modelos del sistema:
//
// Usuarios y Autenticación:
// - User (user.go)
//
// Propiedades:
// - Property (property.go)
// - PropertyPhoto (property_photo.go)
// - Category (category.go)
// - PropertyCategory (category.go) - Tabla pivote
//
// Auditoría:
// - AuditLog (audit.go)
//
// CRM (no implementado aún):
// - Client
// - Interaction
//
// Publicaciones (no implementado aún):
// - Publication
// - Payment
//
// Notificaciones (no implementado aún):
// - Notification

// GetAllModels - Retorna todos los modelos para migración
func GetAllModels() []interface{} {
	return []interface{}{
		&User{},
		&Category{},
		&Property{},
		&PropertyPhoto{},
		&PropertyCategory{},
		&AuditLog{},
	}
}
