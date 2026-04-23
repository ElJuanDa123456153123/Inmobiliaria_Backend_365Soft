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
	ErrCodeNotFound         ErrorCode = "NOT_FOUND"
	ErrCodeUserNotFound     ErrorCode = "USER_NOT_FOUND"
	ErrCodePropertyNotFound ErrorCode = "PROPERTY_NOT_FOUND"
	ErrCodeImageNotFound    ErrorCode = "IMAGE_NOT_FOUND"

	// Errores de Autorización
	ErrCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden    ErrorCode = "FORBIDDEN"
	ErrCodeNotOwner     ErrorCode = "NOT_OWNER"

	// Errores de Estado
	ErrCodeInvalidState  ErrorCode = "INVALID_STATE"
	ErrCodeAlreadyExists ErrorCode = "ALREADY_EXISTS"
	ErrCodeConflict      ErrorCode = "CONFLICT"

	// Errores de Negocio
	ErrCodeBusinessRule  ErrorCode = "BUSINESS_RULE"
	ErrCodeLimitExceeded ErrorCode = "LIMIT_EXCEEDED"
	ErrCodeInvalidPrice  ErrorCode = "INVALID_PRICE"

	// Errores de Archivos
	ErrCodeInvalidFile   ErrorCode = "INVALID_FILE"
	ErrCodeFileTooLarge  ErrorCode = "FILE_TOO_LARGE"
	ErrCodeInvalidFormat ErrorCode = "INVALID_FORMAT"

	// Errores de Base de Datos
	ErrCodeDatabase     ErrorCode = "DATABASE_ERROR"
	ErrCodeDuplicateKey ErrorCode = "DUPLICATE_KEY"

	// Errores de Servicios Externos
	ErrCodeExternalService ErrorCode = "EXTERNAL_SERVICE"
	ErrCodeStorageUpload   ErrorCode = "STORAGE_UPLOAD"

	// Errores Internos
	ErrCodeInternal ErrorCode = "INTERNAL_ERROR"
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

// ValidationErrors - Error de validación con múltiples campos
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

// UserNotFound - Usuario no encontrado
func UserNotFound(id uint) *AppError {
	return NewError(
		ErrCodeUserNotFound,
		fmt.Sprintf("Usuario con ID %d no encontrado", id),
		http.StatusNotFound,
	)
}

// Unauthorized - No autenticado
func Unauthorized(message string) *AppError {
	if message == "" {
		message = "Autenticación requerida"
	}
	return NewError(
		ErrCodeUnauthorized,
		message,
		http.StatusUnauthorized,
	)
}

// Forbidden - No tiene permiso
func Forbidden(message string) *AppError {
	if message == "" {
		message = "No tienes permiso para realizar esta acción"
	}
	return NewError(
		ErrCodeForbidden,
		message,
		http.StatusForbidden,
	)
}

// NotOwner - No es dueño del recurso
func NotOwner(resource string) *AppError {
	return NewError(
		ErrCodeNotOwner,
		fmt.Sprintf("No eres dueño de este %s", resource),
		http.StatusForbidden,
	)
}

// AlreadyExists - Recurso ya existe
func AlreadyExists(resource string) *AppError {
	return NewError(
		ErrCodeAlreadyExists,
		fmt.Sprintf("%s ya existe", resource),
		http.StatusConflict,
	)
}

// LimitExceeded - Límite excedido
func LimitExceeded(message string) *AppError {
	if message == "" {
		message = "Has excedido el límite permitido"
	}
	return NewError(
		ErrCodeLimitExceeded,
		message,
		http.StatusBadRequest,
	)
}

// DatabaseError - Error de base de datos
func DatabaseError(err error) *AppError {
	return Wrap(err, ErrCodeDatabase, "Error de base de datos", http.StatusInternalServerError)
}

// InternalError - Error interno
func InternalError(err error) *AppError {
	return Wrap(err, ErrCodeInternal, "Error interno del servidor", http.StatusInternalServerError)
}

// InvalidFile - Archivo inválido
func InvalidFile(message string) *AppError {
	if message == "" {
		message = "Archivo inválido"
	}
	return NewError(
		ErrCodeInvalidFile,
		message,
		http.StatusBadRequest,
	)
}

// FileTooLarge - Archivo demasiado grande
func FileTooLarge(maxSizeMB int) *AppError {
	return NewError(
		ErrCodeFileTooLarge,
		fmt.Sprintf("El archivo es demasiado grande. Máximo %dMB", maxSizeMB),
		http.StatusBadRequest,
	)
}

// ========== HELPERS ==========

// httpStatusForCode - Retorna HTTP status para un código de error
func httpStatusForCode(code ErrorCode) int {
	switch code {
	case ErrCodeValidation, ErrCodeInvalidInput, ErrCodeRequiredField,
	     ErrCodeInvalidState, ErrCodeInvalidPrice, ErrCodeInvalidFile,
	     ErrCodeFileTooLarge, ErrCodeInvalidFormat:
		return http.StatusBadRequest
	case ErrCodeNotFound, ErrCodeUserNotFound, ErrCodePropertyNotFound, ErrCodeImageNotFound:
		return http.StatusNotFound
	case ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrCodeForbidden, ErrCodeNotOwner:
		return http.StatusForbidden
	case ErrCodeAlreadyExists, ErrCodeConflict:
		return http.StatusConflict
	case ErrCodeLimitExceeded:
		return http.StatusBadRequest
	case ErrCodeDatabase:
		return http.StatusInternalServerError
	case ErrCodeInternal, ErrCodeExternalService, ErrCodeStorageUpload:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// GetHTTPStatus - Retorna HTTP status de un AppError
func GetHTTPStatus(err *AppError) int {
	if err.HTTPStatus != 0 {
		return err.HTTPStatus
	}
	return httpStatusForCode(err.Code)
}
