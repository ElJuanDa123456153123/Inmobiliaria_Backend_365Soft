package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode - Tipo de código de error
type ErrorCode string

const (
	// Errores de Validación
	ErrCodeValidation    ErrorCode = "VALIDATION_ERROR"
	ErrCodeInvalidInput  ErrorCode = "INVALID_INPUT"
	ErrCodeRequiredField ErrorCode = "REQUIRED_FIELD"

	// Errores de No Encontrado
	ErrCodeNotFound      ErrorCode = "NOT_FOUND"
	ErrCodeUserNotFound  ErrorCode = "USER_NOT_FOUND"
	ErrCodePropertyNotFound ErrorCode = "PROPERTY_NOT_FOUND"
	ErrCodeCategoryNotFound ErrorCode = "CATEGORY_NOT_FOUND"

	// Errores de Autorización
	ErrCodeUnauthorized   ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden      ErrorCode = "FORBIDDEN"
	ErrCodeNotOwner       ErrorCode = "NOT_OWNER"

	// Errores de Estado
	ErrCodeInvalidState   ErrorCode = "INVALID_STATE"
	ErrCodeAlreadyExists  ErrorCode = "ALREADY_EXISTS"
	ErrCodeConflict       ErrorCode = "CONFLICT"

	// Errores de Negocio
	ErrCodeBusinessRule   ErrorCode = "BUSINESS_RULE"
	ErrCodeLimitExceeded  ErrorCode = "LIMIT_EXCEEDED"
	ErrCodeInvalidPrice   ErrorCode = "INVALID_PRICE"

	// Errores de Archivos
	ErrCodeInvalidFile    ErrorCode = "INVALID_FILE"
	ErrCodeFileTooLarge   ErrorCode = "FILE_TOO_LARGE"
	ErrCodeInvalidFormat  ErrorCode = "INVALID_FORMAT"

	// Errores de Base de Datos
	ErrCodeDatabase       ErrorCode = "DATABASE_ERROR"
	ErrCodeDuplicateKey   ErrorCode = "DUPLICATE_KEY"

	// Errores de Servicios Externos
	ErrCodeExternalService ErrorCode = "EXTERNAL_SERVICE"
	ErrCodeStorageUpload  ErrorCode = "STORAGE_UPLOAD"

	// Errores Internos
	ErrCodeInternal       ErrorCode = "INTERNAL_ERROR"
)

// AppError - Error de aplicación con contexto
type AppError struct {
	Code       ErrorCode   `json:"code"`
	Message    string      `json:"message"`
	HTTPStatus int         `json:"-"`
	Details    []interface{} `json:"details,omitempty"`
	Err        error       `json:"-"` // Error original para debugging
}

// Error - Implementa interfaz error
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap - Implementa errors.Unwrap
func (e *AppError) Unwrap() error {
	return e.Err
}

// ========== FACTORY FUNCTIONS ==========

// NewError - Crea un nuevo AppError
func NewError(code ErrorCode, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// NewErrorWithDetails - Crea AppError con detalles
func NewErrorWithDetails(code ErrorCode, message string, httpStatus int, details []interface{}) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Details:    details,
	}
}

// Wrap - Envuelve un error con contexto
func Wrap(err error, code ErrorCode, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Err:        err,
	}
}

// Wrapf - Envuelve un error con formato
func Wrapf(err error, code ErrorCode, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       code,
		Message:    fmt.Sprintf(format, args...),
		HTTPStatus: httpStatusForCode(code),
		Err:        err,
	}
}

// ========== ERROR SPECIFIC FACTORIES ==========

// ValidationErrors - Error de validación
func ValidationErrors(details []interface{}) *AppError {
	return NewErrorWithDetails(
		ErrCodeValidation,
		"Error de validación en los datos enviados",
		http.StatusBadRequest,
		details,
	)
}

// ValidationError - Error de validación simple
func ValidationError(field, message string) *AppError {
	return NewError(
		ErrCodeValidation,
		fmt.Sprintf("%s: %s", field, message),
		http.StatusBadRequest,
	)
}

// NotFound - Recurso no encontrado
func NotFound(resource string) *AppError {
	return NewError(
		ErrCodeNotFound,
		fmt.Sprintf("%s no encontrado", resource),
		http.StatusNotFound,
	)
}

// PropertyNotFound - Propiedad no encontrada
func PropertyNotFound(id uint) *AppError {
	return NewError(
		ErrCodePropertyNotFound,
		fmt.Sprintf("Propiedad con ID %d no encontrada", id),
		http.StatusNotFound,
	)
}

// CategoryNotFound - Categoría no encontrada
func CategoryNotFound(slug string) *AppError {
	return NewError(
		ErrCodeCategoryNotFound,
		fmt.Sprintf("Categoría '%s' no encontrada", slug),
		http.StatusNotFound,
	)
}

// Unauthorized - No autorizado
func Unauthorized() *AppError {
	return NewError(
		ErrCodeUnauthorized,
		"No autorizado para realizar esta acción",
		http.StatusUnauthorized,
	)
}

// Forbidden - Prohibido
func Forbidden() *AppError {
	return NewError(
		ErrCodeForbidden,
		"No tienes permisos para realizar esta acción",
		http.StatusForbidden,
	)
}

// NotOwner - Usuario no es dueño
func NotOwner(resource string) *AppError {
	return NewError(
		ErrCodeNotOwner,
		fmt.Sprintf("No eres dueño de este %s", resource),
		http.StatusForbidden,
	)
}

// AlreadyExists - Recurso ya existe
func AlreadyExists(resource, identifier string) *AppError {
	return NewError(
		ErrCodeAlreadyExists,
		fmt.Sprintf("%s con identificador '%s' ya existe", resource, identifier),
		http.StatusConflict,
	)
}

// InvalidPrice - Precio inválido
func InvalidPrice(details string) *AppError {
	return NewError(
		ErrCodeInvalidPrice,
		fmt.Sprintf("Precio inválido: %s", details),
		http.StatusBadRequest,
	)
}

// LimitExceeded - Límite excedido
func LimitExceeded(limit string, current int) *AppError {
	return NewError(
		ErrCodeLimitExceeded,
		fmt.Sprintf("Límite de %s excedido (%d)", limit, current),
		http.StatusBadRequest,
	)
}

// FileTooLarge - Archivo demasiado grande
func FileTooLarge(maxSizeMB int) *AppError {
	return NewError(
		ErrCodeFileTooLarge,
		fmt.Sprintf("El archivo excede el tamaño máximo de %d MB", maxSizeMB),
		http.StatusBadRequest,
	)
}

// InvalidFileFormat - Formato de archivo inválido
func InvalidFileFormat(validFormats []string) *AppError {
	return NewError(
		ErrCodeInvalidFormat,
		fmt.Sprintf("Formato de archivo inválido. Formatos válidos: %v", validFormats),
		http.StatusBadRequest,
	)
}

// DatabaseError - Error de base de datos
func DatabaseError(err error) *AppError {
	return Wrap(
		err,
		ErrCodeDatabase,
		"Error de base de datos",
		http.StatusInternalServerError,
	)
}

// InternalError - Error interno
func InternalError(err error) *AppError {
	return Wrap(
		err,
		ErrCodeInternal,
		"Error interno del servidor",
		http.StatusInternalServerError,
	)
}

// ========== HELPER FUNCTIONS ==========

// IsAppError - Verifica si es un AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetHTTPStatus - Obtiene HTTP status de un error
func GetHTTPStatus(err error) int {
	if appErr, ok := err.(*AppError); ok {
		return appErr.HTTPStatus
	}
	return http.StatusInternalServerError
}

// GetErrorCode - Obtiene código de error
func GetErrorCode(err error) ErrorCode {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return ErrCodeInternal
}

// httpStatusForCode - Mapea ErrorCode a HTTP status
func httpStatusForCode(code ErrorCode) int {
	switch code {
	case ErrCodeValidation, ErrCodeInvalidInput, ErrCodeRequiredField,
	     ErrCodeInvalidPrice, ErrCodeFileTooLarge, ErrCodeInvalidFormat:
		return http.StatusBadRequest

	case ErrCodeNotFound, ErrCodeUserNotFound, ErrCodePropertyNotFound, ErrCodeCategoryNotFound:
		return http.StatusNotFound

	case ErrCodeUnauthorized:
		return http.StatusUnauthorized

	case ErrCodeForbidden, ErrCodeNotOwner:
		return http.StatusForbidden

	case ErrCodeAlreadyExists, ErrCodeConflict, ErrCodeDuplicateKey:
		return http.StatusConflict

	case ErrCodeLimitExceeded, ErrCodeInvalidState:
		return http.StatusBadRequest

	case ErrCodeBusinessRule:
		return http.StatusUnprocessableEntity

	case ErrCodeExternalService, ErrCodeStorageUpload:
		return http.StatusBadGateway

	case ErrCodeDatabase:
		return http.StatusInternalServerError

	default:
		return http.StatusInternalServerError
	}
}

// ToMap - Convierte AppError a map para JSON response
func ToMap(err error) map[string]interface{} {
	if appErr, ok := err.(*AppError); ok {
		result := map[string]interface{}{
			"code":    appErr.Code,
			"message": appErr.Message,
		}

		if len(appErr.Details) > 0 {
			result["details"] = appErr.Details
		}

		return result
	}

	// Error genérico
	return map[string]interface{}{
		"code":    ErrCodeInternal,
		"message": "Error interno del servidor",
	}
}
