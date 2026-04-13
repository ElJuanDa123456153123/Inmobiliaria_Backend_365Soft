package utils

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// SlugGenerator - Generador de slugs únicos
type SlugGenerator struct {
	separator string
}

// NewSlugGenerator - Crea un nuevo generador de slugs
func NewSlugGenerator() *SlugGenerator {
	return &SlugGenerator{
		separator: "-",
	}
}

// Generate - Genera un slug a partir de un string
func (sg *SlugGenerator) Generate(text string) string {
	// 1. Convertir a minúsculas
	text = strings.ToLower(text)

	// 2. Normalizar caracteres Unicode (eliminar tildes)
	text = sg.normalize(text)

	// 3. Reemplazar espacios con guiones
	text = strings.ReplaceAll(text, " ", sg.separator)

	// 4. Eliminar caracteres inválidos
	reg := regexp.MustCompile("[^a-z0-9-"+regexp.QuoteMeta(sg.separator)+"]")
	text = reg.ReplaceAllString(text, "")

	// 5. Eliminar separadores duplicados
	reg = regexp.MustCompile(regexp.QuoteMeta(sg.separator) + "+")
	text = reg.ReplaceAllString(text, sg.separator)

	// 6. Eliminar separadores al inicio y final
	text = strings.Trim(text, sg.separator)

	// 7. Limitar longitud (máximo 200 caracteres para BD)
	if len(text) > 200 {
		text = text[:200]
		// Asegurar que no termine en separador
		text = strings.TrimRight(text, sg.separator)
	}

	// 8. Si quedó vacío, generar slug genérico
	if text == "" {
		text = "sin-titulo"
	}

	return text
}

// GenerateUnique - Genera un slug único añadiendo sufijo si es necesario
func (sg *SlugGenerator) GenerateUnique(base string, existsFunc func(slug string) bool) string {
	slug := sg.Generate(base)

	// Si el slug no existe, retornarlo
	if !existsFunc(slug) {
		return slug
	}

	// Si existe, añadir sufijo numérico
	counter := 2
	newSlug := slug

	for existsFunc(newSlug) {
		newSlug = fmt.Sprintf("%s%s%d", slug, sg.separator, counter)

		// Prevenir overflow muy largo
		if len(newSlug) > 200 {
			// Truncar el slug base y añadir sufijo
			maxBaseLen := 200 - len(fmt.Sprintf("%s%d", sg.separator, counter))
			newSlug = fmt.Sprintf("%s%s%d", slug[:maxBaseLen], sg.separator, counter)
		}

		counter++
	}

	return newSlug
}

// normalize - Normaliza texto eliminando acentos y caracteres especiales
func (sg *SlugGenerator) normalize(text string) string {
	// Usar Unicode normalization para descomponer caracteres
	// Por ejemplo: "á" -> "a" + "́"
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn))
	text, _, _ = transform.String(t, text)
	return text
}

// isMn - Verifica si el rune es un mark non-spacing (para eliminar tildes)
func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: Nonspacing_Mark
}

// ========== FUNCIONES DE UTILIDAD ==========

// GenerateSlug - Función de conveniencia para generar slug
func GenerateSlug(text string) string {
	generator := NewSlugGenerator()
	return generator.Generate(text)
}

// GenerateSlugWithID - Genera slug añadiendo ID al final
func GenerateSlugWithID(text string, id uint) string {
	slug := GenerateSlug(text)
	return fmt.Sprintf("%s%s%d", slug, "-", id)
}

// SanitizeFilename - Sanitiza nombre de archivo para uso en URL
func SanitizeFilename(filename string) string {
	// Obtener extensión
	parts := strings.Split(filename, ".")
	if len(parts) < 2 {
		return "file"
	}

	ext := parts[len(parts)-1]
	base := strings.Join(parts[:len(parts)-1], ".")

	// Generar slug del nombre base
	generator := NewSlugGenerator()
	slug := generator.Generate(base)

	// Retornar slug + extensión
	return fmt.Sprintf("%s.%s", slug, ext)
}

// IsValidSlug - Valida que un slug tenga formato válido
func IsValidSlug(slug string) bool {
	if slug == "" {
		return false
	}

	// Debe contener solo letras, números y guiones
	reg := regexp.MustCompile("^[a-z0-9-]+$")
	return reg.MatchString(slug)
}

// ExtractIDFromSlug - Extrae ID de un slug que termina con "-id"
func ExtractIDFromSlug(slug string) (int, error) {
	// Buscar el último separador
	lastIndex := strings.LastIndex(slug, "-")
	if lastIndex == -1 {
		return 0, fmt.Errorf("no separator found in slug")
	}

	// Extraer la parte después del último separador
	idStr := slug[lastIndex+1:]

	// Verificar que sea un número
	var id int
	_, err := fmt.Sscanf(idStr, "%d", &id)
	if err != nil {
		return 0, fmt.Errorf("invalid ID format in slug")
	}

	return id, nil
}

// CompareSlugs - Compara dos slugs de forma segura
func CompareSlugs(slug1, slug2 string) bool {
	return strings.EqualFold(slug1, slug2)
}

// SlugFromTitle - Genera slug desde título con manejo de duplicados
func SlugFromTitle(title string, existingSlugs []string) string {
	generator := NewSlugGenerator()
	baseSlug := generator.Generate(title)

	// Si no existe en la lista, retornar
	for _, existing := range existingSlugs {
		if CompareSlugs(baseSlug, existing) {
			// Existe, generar con sufijo
			return generator.GenerateUnique(title, func(slug string) bool {
				for _, existing := range existingSlugs {
					if CompareSlugs(slug, existing) {
						return true
					}
				}
				return false
			})
		}
	}

	return baseSlug
}

// NormalizeForSearch - Normaliza texto para búsqueda
func NormalizeForSearch(text string) string {
	text = strings.ToLower(text)
	text = strings.TrimSpace(text)

	// Eliminar artículos comunes
	stopWords := []string{"el", "la", "los", "las", "un", "una", "unos", "unas", "de", "en", "con", "por", "para"}
	words := strings.Fields(text)

	filtered := make([]string, 0, len(words))
	for _, word := range words {
		isStopWord := false
		for _, stop := range stopWords {
			if word == stop {
				isStopWord = true
				break
			}
		}
		if !isStopWord {
			filtered = append(filtered, word)
		}
	}

	return strings.Join(filtered, " ")
}

// GenerateKeywordSlug - Genera slug optimizado para keywords
func GenerateKeywordSlug(text string) string {
	generator := NewSlugGenerator()

	// Generar slug normal
	slug := generator.Generate(text)

	// Reemplazar guiones con espacios para búsqueda
	terms := strings.Split(slug, "-")

	// Eliminar duplicados
	uniqueTerms := make(map[string]bool)
	filteredTerms := make([]string, 0, len(terms))

	for _, term := range terms {
		if !uniqueTerms[term] && term != "" {
			uniqueTerms[term] = true
			filteredTerms = append(filteredTerms, term)
		}
	}

	return strings.Join(filteredTerms, " ")
}
