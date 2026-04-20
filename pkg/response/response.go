package response

import (
	"time"

	apperrors "github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/errors"
)

// Response - Respuesta estándar de la API
type Response struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	Data      any    `json:"data,omitempty"`
	Error     any    `json:"error,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	Timestamp string `json:"timestamp"`
}

// ErrorDetail - Detalle de error para validaciones
type ErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ErrorResponse - Respuesta de error
type ErrorResponse struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []interface{} `json:"details,omitempty"`
}

// ========== FACTORY FUNCTIONS ==========

// Success - Respuesta exitosa
func Success(data any) Response {
	return Response{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// SuccessWithMessage - Respuesta exitosa con mensaje
func SuccessWithMessage(message string, data any) Response {
	return Response{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// Error - Respuesta de error
func Error(message string) Response {
	return Response{
		Success:   false,
		Error:     message,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// ErrorWithCode - Respuesta de error con código
func ErrorWithCode(code apperrors.ErrorCode, message string) Response {
	return Response{
		Success: false,
		Error: ErrorResponse{
			Code:    string(code),
			Message: message,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// ErrorWithDetails - Respuesta de error con detalles
func ErrorWithDetails(code apperrors.ErrorCode, message string, details []interface{}) Response {
	return Response{
		Success: false,
		Error: ErrorResponse{
			Code:    string(code),
			Message: message,
			Details: details,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// FromAppError - Convierte AppError a Response
func FromError(err error) Response {
	if appErr, ok := err.(*apperrors.AppError); ok {
		return Response{
			Success: false,
			Error: ErrorResponse{
				Code:    string(appErr.Code),
				Message: appErr.Message,
				Details: appErr.Details,
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	// Error genérico
	return Response{
		Success: false,
		Error: ErrorResponse{
			Code:    string(apperrors.ErrCodeInternal),
			Message: "Error interno del servidor",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// ValidationError - Respuesta de error de validación
func ValidationError(field, message string) Response {
	return ErrorWithDetails(
		apperrors.ErrCodeValidation,
		"Error de validación",
		[]interface{}{
			ErrorDetail{
				Field:   field,
				Message: message,
			},
		},
	)
}

// ValidationErrors - Respuesta de múltiples errores de validación
func ValidationErrors(details []ErrorDetail) Response {
	detailsInterface := make([]interface{}, len(details))
	for i, d := range details {
		detailsInterface[i] = d
	}

	return ErrorWithDetails(
		apperrors.ErrCodeValidation,
		"Error de validación en los datos enviados",
		detailsInterface,
	)
}

// NotFound - Respuesta de recurso no encontrado
func NotFound(resource string) Response {
	return ErrorWithCode(
		apperrors.ErrCodeNotFound,
		resource+" no encontrado",
	)
}

// Unauthorized - Respuesta de no autenticado
func Unauthorized(message string) Response {
	if message == "" {
		message = "Autenticación requerida"
	}
	return ErrorWithCode(apperrors.ErrCodeUnauthorized, message)
}

// Forbidden - Respuesta de permiso denegado
func Forbidden(message string) Response {
	if message == "" {
		message = "No tienes permiso para realizar esta acción"
	}
	return ErrorWithCode(apperrors.ErrCodeForbidden, message)
}
