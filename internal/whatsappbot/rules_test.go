package whatsappbot

import "testing"

// TestEsMensajeCliente — réplica del cálculo isIncoming del nodo Code "Detectar Tipo"
// (workflows/11-chatwoot-bot-v4-crm.json)
func TestEsMensajeCliente(t *testing.T) {
	casos := []struct {
		nombre      string
		event       string
		messageType string
		private     bool
		content     string
		tieneAudio  bool
		esperado    bool
	}{
		{"texto del cliente", "message_created", "incoming", false, "busco departamento", false, true},
		{"nota de voz del cliente", "message_created", "incoming", false, "", true, true},
		{"respuesta del bot se ignora", "message_created", "outgoing", false, "hola", false, false},
		{"nota privada del asesor se ignora", "message_created", "incoming", true, "#confirmar", false, false},
		{"evento ajeno se ignora", "message_updated", "incoming", false, "hola", false, false},
		{"mensaje vacío sin audio se ignora", "message_created", "incoming", false, "", false, false},
	}
	for _, c := range casos {
		if got := EsMensajeCliente(c.event, c.messageType, c.private, c.content, c.tieneAudio); got != c.esperado {
			t.Errorf("%s: EsMensajeCliente = %v, esperado %v", c.nombre, got, c.esperado)
		}
	}
}

// TestBotActivo — réplica de botActive del nodo "Detectar Tipo"
func TestBotActivo(t *testing.T) {
	if !BotActivo("open") {
		t.Error("conversación open debe mantener el bot activo")
	}
	if BotActivo("pending") {
		t.Error("conversación pending (asesor a cargo) debe silenciar al bot")
	}
	if BotActivo("resolved") {
		t.Error("conversación resolved debe silenciar al bot")
	}
}

// TestDetectaHandoff — réplica de needsHuman del nodo "Detectar Tipo"
func TestDetectaHandoff(t *testing.T) {
	casos := []struct {
		texto    string
		esperado bool
	}{
		{"quiero hablar con un asesor", true},
		{"necesito una persona real", true},
		{"hay algún encargado?", true},
		{"HABLAR CON ALGUIEN", true},
		{"busco casa en Sacaba", false},
		{"", false},
	}
	for _, c := range casos {
		if got := DetectaHandoff(c.texto); got != c.esperado {
			t.Errorf("DetectaHandoff(%q) = %v, esperado %v", c.texto, got, c.esperado)
		}
	}
}

// TestParsearComandoVisita — réplica de la detección de hashtags del nodo "Detectar Tipo"
func TestParsearComandoVisita(t *testing.T) {
	casos := []struct {
		nombre         string
		esNotaPrivada  bool
		content        string
		cmdEsperado    string
		horarioEsperado string
	}{
		{"confirmar", true, "#confirmar", "confirmar", ""},
		{"confirmar con espacios", true, "  #Confirmar ", "confirmar", ""},
		{"cancelar", true, "#cancelar", "cancelar", ""},
		{"posponer con horario", true, "#posponer viernes 15:00", "posponer", "viernes 15:00"},
		{"reprogramar (alias)", true, "#reprogramar lunes 09:30", "posponer", "lunes 09:30"},
		{"mensaje del cliente no es comando", false, "#confirmar", "", ""},
		{"nota privada sin hashtag", true, "revise la propiedad", "", ""},
	}
	for _, c := range casos {
		cmd, horario := ParsearComandoVisita(c.esNotaPrivada, c.content)
		if cmd != c.cmdEsperado || horario != c.horarioEsperado {
			t.Errorf("%s: obtuvo (%q,%q), esperado (%q,%q)", c.nombre, cmd, horario, c.cmdEsperado, c.horarioEsperado)
		}
	}
}

// TestNormalizarWaID — réplica de la normalización de wa_id del nodo "Detectar Tipo"
func TestNormalizarWaID(t *testing.T) {
	if got := NormalizarWaID("+59177123456"); got != "59177123456" {
		t.Errorf("NormalizarWaID = %q, esperado 59177123456", got)
	}
	if got := NormalizarWaID("59177123456"); got != "59177123456" {
		t.Errorf("NormalizarWaID sin + = %q", got)
	}
}

// TestSanearFiltrosBusqueda — réplica del nodo Code "Sanear Filtros"
// (workflows/14-buscar-propiedades-tool-crm.json)
func TestSanearFiltrosBusqueda(t *testing.T) {
	casos := []struct {
		nombre   string
		f        FiltrosBusqueda
		esperado string
	}{
		{
			"consulta coloquial completa",
			FiltrosBusqueda{PropertyType: "depas", ListingType: "alquilar", Zone: "Sacaba", MaxPrice: 2000, Bedrooms: 2, Currency: "bs"},
			"tipo_propiedad=departamento&tipo_operacion=alquiler&zona=Sacaba&moneda=BOB&precio_max=2000&dormitorios=2&limit=4",
		},
		{
			"alias apartamento y venta",
			FiltrosBusqueda{PropertyType: "apartamentos", ListingType: "comprar"},
			"tipo_propiedad=departamento&tipo_operacion=venta&limit=4",
		},
		{
			"moneda bolivianos",
			FiltrosBusqueda{Currency: "bolivianos"},
			"moneda=BOB&limit=4",
		},
		{
			"moneda USD",
			FiltrosBusqueda{Currency: "usd", MaxPrice: 80000},
			"moneda=USD&precio_max=80000&limit=4",
		},
		{
			"acentos eliminados en la zona",
			FiltrosBusqueda{Zone: "Quillacollo", ListingType: "alquiler"},
			"tipo_operacion=alquiler&zona=Quillacollo&limit=4",
		},
		{
			"tipo fuera de la lista blanca se descarta",
			FiltrosBusqueda{PropertyType: "castillo"},
			"limit=4",
		},
		{
			"operación desconocida se descarta",
			FiltrosBusqueda{ListingType: "permutar"},
			"limit=4",
		},
		{
			"valores no positivos se descartan",
			FiltrosBusqueda{MaxPrice: -5, Bedrooms: 0},
			"limit=4",
		},
		{
			"depto singularizado",
			FiltrosBusqueda{PropertyType: "depto", ListingType: "alquiler", Bedrooms: 3},
			"tipo_propiedad=departamento&tipo_operacion=alquiler&dormitorios=3&limit=4",
		},
	}
	for _, c := range casos {
		if got := SanearFiltrosBusqueda(c.f); got != c.esperado {
			t.Errorf("%s:\n obtuvo:   %s\n esperado: %s", c.nombre, got, c.esperado)
		}
	}
}

// TestDeduplicacionTelefono — réplica de la deduplicación por sufijo de 8 dígitos
// del nodo Code "Upsert Cliente CRM" (workflows/11-chatwoot-bot-v4-crm.json)
func TestDeduplicacionTelefono(t *testing.T) {
	existentes := []string{"+591 700 11223", "70044556", "77712345"}

	if idx := IndiceClientePorTelefono(existentes, "+59177712345"); idx != 2 {
		t.Errorf("mismo número con formato distinto debe coincidir (idx=%d, esperado 2)", idx)
	}
	if idx := IndiceClientePorTelefono(existentes, "59170011223"); idx != 0 {
		t.Errorf("sufijo de 8 dígitos debe coincidir (idx=%d, esperado 0)", idx)
	}
	if idx := IndiceClientePorTelefono(existentes, "700-99-88"); idx != -1 {
		t.Errorf("número nuevo no debe coincidir (idx=%d, esperado -1)", idx)
	}
	if idx := IndiceClientePorTelefono(existentes, ""); idx != -1 {
		t.Errorf("teléfono vacío no debe coincidir (idx=%d, esperado -1)", idx)
	}
	if got := DigitosTelefono("+591 (7) 00-11-223"); got != "59170011223" {
		t.Errorf("DigitosTelefono = %q, esperado 59170011223", got)
	}
	if got := SufijoTelefono("777-12-345"); got != "77712345" {
		t.Errorf("SufijoTelefono = %q, esperado 77712345", got)
	}
}
