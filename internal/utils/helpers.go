package utils

import (
	"fmt"
	"math"
	mrand "math/rand"
	"strconv"
	"strings"
	"time"
)

// ========== FUNCIONES DE TIEMPO ==========

// NowUTC - Retorna timestamp actual en UTC
func NowUTC() time.Time {
	return time.Now().UTC()
}

// FormatTimestamp - Formatea timestamp para respuesta JSON
func FormatTimestamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// ParseTimestamp - Parsea timestamp desde string
func ParseTimestamp(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// AddBusinessDays - Añade días hábiles a una fecha
func AddBusinessDays(t time.Time, days int) time.Time {
	count := 0
	current := t

	for count < days {
		current = current.AddDate(0, 0, 1)

		// Si es sábado (6) o domingo (0), no contar
		if current.Weekday() != time.Saturday && current.Weekday() != time.Sunday {
			count++
		}
	}

	return current
}

// IsBusinessDay - Verifica si una fecha es día hábil
func IsBusinessDay(t time.Time) bool {
	weekday := t.Weekday()
	return weekday != time.Saturday && weekday != time.Sunday
}

// ========== FUNCIONES DE MONEDA ==========

// FormatPrice - Formatea precio para mostrar
func FormatPrice(precio float64, moneda string) string {
	switch moneda {
	case "USD", "usd":
		return fmt.Sprintf("$%.2f", precio)
	case "BOB", "bob":
		return fmt.Sprintf("Bs%.2f", precio)
	default:
		return fmt.Sprintf("%.2f %s", precio, moneda)
	}
}

// ParsePrice - Parsea precio desde string
func ParsePrice(s string) (float64, error) {
	// Eliminar símbolos de moneda y espacios
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, "Bs", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)

	return strconv.ParseFloat(s, 64)
}

// ConvertCurrency - Convierte entre monedas (tasa fija por ahora)
func ConvertCurrency(amount float64, from, to string) float64 {
	// Tasas de conversión hardcoded (idealmente desde API)
	rates := map[string]map[string]float64{
		"USD": {
			"BOB": 6.96, // 1 USD = 6.96 BOB
			"USD": 1.0,
		},
		"BOB": {
			"USD": 0.144, // 1 BOB = 0.144 USD
			"BOB": 1.0,
		},
	}

	if rate, ok := rates[from][to]; ok {
		return amount * rate
	}

	return amount // Si no hay tasa, retornar mismo monto
}

// ========== FUNCIONES DE STRING ==========

// Truncate - Trunca string a longitud máxima
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// TruncateWords - Trunca string a número máximo de palabras
func TruncateWords(s string, maxWords int) string {
	words := strings.Fields(s)
	if len(words) <= maxWords {
		return s
	}
	return strings.Join(words[:maxWords], " ") + "..."
}

// CleanWhitespace - Limpia espacios extras
func CleanWhitespace(s string) string {
	words := strings.Fields(s)
	return strings.Join(words, " ")
}

// Contains - Verifica si un string está en un slice
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// RemoveDuplicates - Elimina duplicados de slice
func RemoveDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// ========== FUNCIONES DE VALIDACIÓN ==========

// IsValidEmail - Valida formato de email (básico)
func IsValidEmail(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

// IsValidPhone - Valida formato de teléfono boliviano
func IsValidPhone(phone string) bool {
	// Formatos válidos: +591 XXXXXXXX, 0XXXXXXXX
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")

	if strings.HasPrefix(phone, "+591") {
		return len(phone) == 12 // +591 + 8 dígitos
	}

	if strings.HasPrefix(phone, "0") {
		return len(phone) == 9 // 0 + 8 dígitos
	}

	return len(phone) == 8 // Solo 8 dígitos
}

// IsValidURL - Valida formato de URL (básico)
func IsValidURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

// ========== FUNCIONES DE ARCHIVOS ==========

// GetFileExtension - Obtiene extensión de archivo
func GetFileExtension(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) < 2 {
		return ""
	}
	return strings.ToLower(parts[len(parts)-1])
}

// IsImageFile - Verifica si es una imagen
func IsImageFile(filename string) bool {
	ext := GetFileExtension(filename)
	imageExtensions := []string{"jpg", "jpeg", "png", "gif", "webp"}
	return Contains(imageExtensions, ext)
}

// GenerateFilename - Genera nombre de archivo único
func GenerateFilename(originalFilename string) string {
	ext := GetFileExtension(originalFilename)
	base := SanitizeFilename(strings.TrimSuffix(originalFilename, "."+ext))

	timestamp := time.Now().Unix()
	random := mrand.Intn(1000)

	return fmt.Sprintf("%s_%d_%d.%s", base, timestamp, random, ext)
}

// FormatFileSize - Formatea tamaño de archivo
func FormatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// ========== FUNCIONES DE COORDENADAS ==========

// FormatCoordinates - Formatea coordenadas para mostrar
func FormatCoordinates(lat, lng float64) string {
	return fmt.Sprintf("%.6f, %.6f", lat, lng)
}

// GenerateGoogleMapsURL - Genera URL de Google Maps
func GenerateGoogleMapsURL(lat, lng float64) string {
	return fmt.Sprintf("https://maps.google.com/?q=%.6f,%.6f", lat, lng)
}

// ParseCoordinates - Parsea coordenadas desde string "lat,lng"
func ParseCoordinates(s string) (lat, lng float64, err error) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("formato inválido, se espera 'lat,lng'")
	}

	lat, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("latitud inválida: %w", err)
	}

	lng, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("longitud inválida: %w", err)
	}

	return lat, lng, nil
}

// IsValidCoordinates - Valida rango de coordenadas
func IsValidCoordinates(lat, lng float64) bool {
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

// IsInBolivia - Verifica si coordenadas están en Bolivia
func IsInBolivia(lat, lng float64) bool {
	// Bolivia aprox: -9° a -23° latitud, -57° a -69° longitud
	return lat >= -23 && lat <= -9 && lng >= -69 && lng <= -57
}

// CalculateDistance - Calcula distancia entre dos puntos (fórmula Haversine)
func CalculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371 // km

	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLng := (lng2 - lng1) * math.Pi / 180.0

	lat1Rad := lat1 * math.Pi / 180.0
	lat2Rad := lat2 * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLng/2)*math.Sin(dLng/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// ========== FUNCIONES DE NÚMEROS ==========

// PtrFloat64 - Crea puntero de float64
func PtrFloat64(v float64) *float64 {
	return &v
}

// PtrInt - Crea puntero de int
func PtrInt(v int) *int {
	return &v
}

// PtrString - Crea puntero de string
func PtrString(v string) *string {
	return &v
}

// DerefFloat64 - Desreferencia float64 o retorna default
func DerefFloat64(v *float64, defaultVal float64) float64 {
	if v == nil {
		return defaultVal
	}
	return *v
}

// DerefInt - Desreferencia int o retorna default
func DerefInt(v *int, defaultVal int) int {
	if v == nil {
		return defaultVal
	}
	return *v
}

// DerefString - Desreferencia string o retorna default
func DerefString(v *string, defaultVal string) string {
	if v == nil {
		return defaultVal
	}
	return *v
}

// ========== FUNCIONES DE ALEATORIEDAD ==========

// RandomString - Genera string aleatorio
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)

	for i := range b {
		b[i] = charset[mrand.Intn(len(charset))]
	}

	return string(b)
}

// RandomNumber - Genera número aleatorio en rango
func RandomNumber(min, max int) int {
	return mrand.Intn(max-min+1) + min
}

// ========== FUNCIONES DE API ==========

// BuildURL - Construye URL con query params
func BuildURL(base string, params map[string]string) string {
	if len(params) == 0 {
		return base
	}

	parts := make([]string, 0, len(params))
	for key, value := range params {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}

	return fmt.Sprintf("%s?%s", base, strings.Join(parts, "&"))
}

// BuildPath - Construye path uniendo segmentos
func BuildPath(segments ...string) string {
	path := strings.Join(segments, "/")
	return strings.TrimRight(path, "/")
}
