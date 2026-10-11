// Package whatsappbot contiene réplicas en Go de la lógica pura implementada en los
// nodos Code de los flujos n8n del Asistente de WhatsApp, con el propósito de poder
// someterla a pruebas unitarias (los flujos .json de n8n no son directamente testeables).
//
// Origen de cada réplica:
//   - Detectar Tipo          -> workflows/11-chatwoot-bot-v4-crm.json (nodo Code "Detectar Tipo")
//   - Sanear Filtros         -> workflows/14-buscar-propiedades-tool-crm.json (nodo Code "Sanear Filtros")
//   - Upsert Cliente CRM     -> workflows/11-chatwoot-bot-v4-crm.json (nodo Code "Upsert Cliente CRM")
package whatsappbot

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// EsMensajeCliente replica el cálculo isIncoming del nodo Code "Detectar Tipo":
// solo los mensajes entrantes del cliente (no notas privadas del asesor) con texto
// o nota de voz activan al bot.
func EsMensajeCliente(event, messageType string, private bool, content string, tieneAudio bool) bool {
	return event == "message_created" &&
		messageType == "incoming" &&
		!private &&
		(len(content) > 0 || tieneAudio)
}

// BotActivo replica botActive del nodo "Detectar Tipo": el bot solo responde en
// conversaciones en estado "open" (pending/resolved = asesor humano a cargo).
func BotActivo(status string) bool {
	return status == "open"
}

// regex de handoff idéntica a la del nodo "Detectar Tipo"
var regexHandoff = regexp.MustCompile(`(?i)(asesor|humano|agente|persona real|encargado|hablar con alguien)`)

// DetectaHandoff replica needsHuman del nodo "Detectar Tipo": detecta la intención
// del cliente de hablar con un asesor humano.
func DetectaHandoff(texto string) bool {
	return regexHandoff.MatchString(texto)
}

var regexPosponer = regexp.MustCompile(`(?i)^#(?:posponer|reprogramar)\s+(.+)`)

// ParsearComandoVisita replica la detección de comandos del asesor en notas privadas
// del nodo "Detectar Tipo": #confirmar | #cancelar | #posponer <horario> | #reprogramar <horario>.
// Devuelve el comando ("confirmar", "cancelar", "posponer" o "") y el nuevo horario si aplica.
func ParsearComandoVisita(esNotaPrivada bool, content string) (string, string) {
	if !esNotaPrivada {
		return "", ""
	}
	t := strings.ToLower(strings.TrimSpace(content))
	if strings.HasPrefix(t, "#confirmar") {
		return "confirmar", ""
	}
	if strings.HasPrefix(t, "#cancelar") {
		return "cancelar", ""
	}
	if m := regexPosponer.FindStringSubmatch(strings.TrimSpace(content)); m != nil {
		return "posponer", strings.TrimSpace(m[1])
	}
	return "", ""
}

// NormalizarWaID replica la normalización del identificador de WhatsApp del nodo
// "Detectar Tipo": elimina el prefijo "+" del número del cliente.
func NormalizarWaID(tel string) string {
	return strings.ReplaceAll(tel, "+", "")
}

// FiltrosBusqueda representa los filtros que el agente de IA llena al consultar
// el catálogo (entrada del sub-flujo "Buscar Propiedades").
type FiltrosBusqueda struct {
	PropertyType string
	ListingType  string
	Zone         string
	MaxPrice     float64
	Bedrooms     float64
	Currency     string
}

// stripAccents sustituye la normalización Unicode NFD de JavaScript por un mapeo
// directo de los caracteres acentuados del español (efecto equivalente).
var stripAccents = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
	"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U", "Ü", "U", "Ñ", "N",
)

func cleanValor(v string) string {
	return strings.TrimSpace(stripAccents.Replace(v))
}

// alias de tipos de inmueble idénticos a los del nodo "Sanear Filtros"
var aliasTipoPropiedad = map[string]string{
	"depa":         "departamento",
	"depto":        "departamento",
	"apartamento":  "departamento",
	"casa":         "casa",
	"departamento": "departamento",
	"terreno":      "terreno",
	"oficina":      "oficina",
	"local":        "local",
}

// SanearFiltrosBusqueda replica el nodo Code "Sanear Filtros": lista blanca de
// parámetros, alias de términos coloquiales y moneda normalizada (Bs -> BOB).
// Devuelve el querystring ordenado que se anexa a /api/properties.
func SanearFiltrosBusqueda(f FiltrosBusqueda) string {
	parts := make([]string, 0, 7)
	add := func(k, v string) {
		if v != "" {
			parts = append(parts, k+"="+url.QueryEscape(v))
		}
	}

	// Tipo de inmueble (con alias comunes y singularización)
	tipo := strings.ToLower(cleanValor(f.PropertyType))
	tipo = strings.TrimSuffix(tipo, "s")
	if alias, ok := aliasTipoPropiedad[tipo]; ok {
		add("tipo_propiedad", alias)
	}

	// Operación: venta o alquiler
	lt := strings.ToLower(cleanValor(f.ListingType))
	switch lt {
	case "venta", "compra", "comprar":
		add("tipo_operacion", "venta")
	case "alquiler", "alquilar":
		add("tipo_operacion", "alquiler")
	}

	// Zona (la API matchea parcial e insensible a mayúsculas)
	if zone := cleanValor(f.Zone); zone != "" {
		add("zona", zone)
	}

	// Moneda del presupuesto (Bs -> BOB)
	curr := strings.ToUpper(cleanValor(f.Currency))
	switch curr {
	case "USD":
		add("moneda", "USD")
	case "BS", "BOLIVIANOS":
		add("moneda", "BOB")
	}

	// Presupuesto máximo
	if f.MaxPrice > 0 {
		add("precio_max", strconv.FormatFloat(f.MaxPrice, 'f', -1, 64))
	}

	// Dormitorios
	if f.Bedrooms > 0 {
		add("dormitorios", strconv.Itoa(int(f.Bedrooms)))
	}

	// Límite de resultados fijo
	add("limit", "4")

	return strings.Join(parts, "&")
}

// DigitosTelefono replica norm() del nodo Code "Upsert Cliente CRM":
// conserva únicamente los dígitos del teléfono.
func DigitosTelefono(v string) string {
	var b strings.Builder
	for _, r := range v {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// SufijoTelefono devuelve los últimos 8 dígitos del teléfono (deduplicación).
func SufijoTelefono(v string) string {
	d := DigitosTelefono(v)
	if len(d) < 8 {
		return ""
	}
	return d[len(d)-8:]
}

// IndiceClientePorTelefono replica la deduplicación del nodo "Upsert Cliente CRM":
// busca en la lista de teléfonos de clientes existentes aquel cuyo sufijo de 8
// dígitos coincida con el del nuevo lead. Devuelve el índice encontrado o -1.
func IndiceClientePorTelefono(telefonosExistentes []string, telefonoNuevo string) int {
	suf := SufijoTelefono(telefonoNuevo)
	if suf == "" {
		return -1
	}
	for i, tel := range telefonosExistentes {
		if SufijoTelefono(tel) == suf {
			return i
		}
	}
	return -1
}
