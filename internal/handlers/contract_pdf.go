package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/go-pdf/fpdf"
)

// mesesEs - Nombres de meses en español
var mesesEs = []string{"enero", "febrero", "marzo", "abril", "mayo", "junio",
	"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"}

func fechaLarga(t time.Time) string {
	return fmt.Sprintf("%d de %s de %d", t.Day(), mesesEs[int(t.Month())-1], t.Year())
}

func formatMonto(monto float64, moneda string) string {
	s := fmt.Sprintf("%.2f", monto)
	if moneda == "USD" {
		return "USD " + s
	}
	return "Bs. " + s
}

// Rutas donde pueden estar las fuentes UTF-8 (local vs contenedor)
var rutasFuentes = []string{"internal/handlers/fonts", "fonts"}

func existeArchivo(ruta string) bool {
	info, err := os.Stat(ruta)
	return err == nil && !info.IsDir()
}

// cargarFuentesUTF8 registra DejaVu Sans (soporte completo de acentos/ñ).
// Devuelve true si la fuente quedó disponible.
func cargarFuentesUTF8(pdf *fpdf.Fpdf) bool {
	for _, dir := range rutasFuentes {
		regular := filepath.Join(dir, "DejaVuSans.ttf")
		bold := filepath.Join(dir, "DejaVuSans-Bold.ttf")
		oblicua := filepath.Join(dir, "DejaVuSans-Oblique.ttf")
		if existeArchivo(regular) && existeArchivo(bold) && existeArchivo(oblicua) {
			pdf.AddUTF8Font("DejaVu", "", regular)
			pdf.AddUTF8Font("DejaVu", "B", bold)
			pdf.AddUTF8Font("DejaVu", "I", oblicua)
			return true
		}
	}
	return false
}

// transliterar reemplaza caracteres fuera de ASCII (usado solo como
// respaldo si la fuente UTF-8 no está disponible).
func transliterar(s string) string {
	reemplazos := map[rune]string{
		'á': "a", 'é': "e", 'í': "i", 'ó': "o", 'ú': "u", 'ü': "u", 'ñ': "n",
		'Á': "A", 'É': "E", 'Í': "I", 'Ó': "O", 'Ú': "U", 'Ü': "U", 'Ñ': "N",
		'“': "\"", '”': "\"", '‘': "'", '’': "'", '—': "-", '–': "-", '·': "-",
	}
	var b strings.Builder
	for _, r := range s {
		if rep, ok := reemplazos[r]; ok {
			b.WriteString(rep)
		} else if r < 128 || unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune('?')
		}
	}
	return b.String()
}

// generarPDFContrato - Genera el PDF de un contrato de alquiler en memoria
func generarPDFContrato(contract *models.RentalContract) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 24)
	pdf.AddPage()

	utf8OK := cargarFuentesUTF8(pdf)
	t := func(s string) string {
		if utf8OK {
			return s
		}
		return transliterar(s)
	}
	fuente := func(estilo string, tam float64) {
		if utf8OK {
			pdf.SetFont("DejaVu", estilo, tam)
		} else {
			pdf.SetFont("Arial", estilo, tam)
		}
	}

	width := 180.0 // ancho útil A4 con márgenes de 15mm

	// Código de verificación: hash SHA-256 de los datos clave del contrato
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d|%s|%s|%s|%s|%s|%.2f|%s",
		contract.ID,
		contract.Vendedor.CI, contract.Inquilino.CI,
		contract.FechaInicio.Format("2006-01-02"), contract.FechaFin.Format("2006-01-02"),
		contract.Property.Titulo, contract.MontoMensual,
		contract.Property.Moneda,
	)))
	codigoHex := hex.EncodeToString(hash[:])
	codigoFormateado := func() string {
		partes := make([]string, 0, 8)
		for i := 0; i < 32; i += 4 {
			partes = append(partes, codigoHex[i:i+4])
		}
		return strings.Join(partes, "-")
	}

	// ===== ENCABEZADO =====
	pdf.SetFillColor(30, 58, 138) // azul oscuro
	pdf.RoundedRect(15, 15, width, 20, 2.5, "1234", "F")
	pdf.SetTextColor(255, 255, 255)
	fuente("B", 15)
	pdf.SetXY(15, 18)
	pdf.CellFormat(width, 8, t("CONTRATO DE ALQUILER DE INMUEBLE"), "", 1, "C", false, 0, "")
	fuente("", 9)
	pdf.SetX(15)
	pdf.CellFormat(width, 5, t(fmt.Sprintf("365Soft Inmobiliaria  ·  Contrato Nro. %d", contract.ID)), "", 1, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(6)

	// Helper: fila etiqueta (negrita) + valor con ajuste de línea
	kv := func(label, valor string, lw float64) {
		fuente("B", 9.5)
		pdf.CellFormat(lw, 5.6, label, "", 0, "L", false, 0, "")
		fuente("", 9.5)
		pdf.MultiCell(width-lw, 5.6, valor, "", "L", false)
	}

	// Helper: título de sección con banda
	seccion := func(titulo string) {
		pdf.Ln(2)
		pdf.SetFillColor(234, 240, 252)
		pdf.SetTextColor(30, 58, 138)
		fuente("B", 11)
		pdf.CellFormat(width, 7, "  "+t(titulo), "", 1, "L", true, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.Ln(2.5)
	}

	// ===== I. PARTES =====
	seccion("I. PARTES DEL CONTRATO")

	cajaParte := func(x float64, rol, nombre, ci, tel, email string) {
		boxW := 88.0
		pdf.SetDrawColor(200, 210, 230)
		pdf.SetFillColor(249, 250, 252)
		pdf.RoundedRect(x, pdf.GetY(), boxW, 30, 2, "1234", "FD")
		y := pdf.GetY() + 3
		fuente("B", 9)
		pdf.SetXY(x, y)
		pdf.SetTextColor(30, 58, 138)
		pdf.CellFormat(boxW, 5, t(rol), "", 2, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		fuente("B", 9.5)
		pdf.SetX(x + 4)
		pdf.CellFormat(boxW-8, 5, nombre, "", 2, "L", false, 0, "")
		fuente("", 8.5)
		pdf.SetX(x + 4)
		pdf.CellFormat(boxW-8, 4.5, t("CI: ")+ci, "", 2, "L", false, 0, "")
		pdf.SetX(x + 4)
		pdf.CellFormat(boxW-8, 4.5, t("Tel: ")+tel, "", 2, "L", false, 0, "")
		pdf.SetX(x + 4)
		pdf.CellFormat(boxW-8, 4.5, "Email: "+email, "", 0, "L", false, 0, "")
	}

	yCajas := pdf.GetY()
	cajaParte(15, "ARRENDADOR (Propietario)",
		strings.TrimSpace(contract.Vendedor.Nombre+" "+contract.Vendedor.Apellido),
		contract.Vendedor.CI, contract.Vendedor.Telefono, contract.Vendedor.Email)
	cajaParte(15+92, "ARRENDATARIO (Inquilino)",
		strings.TrimSpace(contract.Inquilino.Nombre+" "+contract.Inquilino.Apellido),
		contract.Inquilino.CI, contract.Inquilino.Telefono, contract.Inquilino.Email)
	// Ambas cajas se dibujaron a la misma altura; avanzar debajo de ellas
	pdf.SetY(yCajas + 32)
	pdf.Ln(1)

	// ===== II. INMUEBLE =====
	seccion("II. INMUEBLE OBJETO DEL CONTRATO")
	direccion := contract.Property.Direccion
	if direccion == "" {
		direccion = "(Dirección no especificada)"
	}
	kv(t("Título:"), contract.Property.Titulo, 42)
	kv(t("Tipo:"), string(contract.Property.TipoPropiedad), 42)
	kv(t("Dirección:"), direccion, 42)
	kv(t("Zona:"), contract.Property.Zona, 42)

	// ===== III. CONDICIONES ECONÓMICAS =====
	seccion("III. CONDICIONES ECONÓMICAS")
	moneda := contract.Property.Moneda
	kv(t("Renta mensual:"), formatMonto(contract.MontoMensual, moneda), 52)
	kv(t("Fecha de inicio:"), fechaLarga(contract.FechaInicio), 52)
	kv(t("Fecha de finalización:"), fechaLarga(contract.FechaFin), 52)
	kv(t("Plazo:"), fmt.Sprintf("%d meses", mesesEntre(contract.FechaInicio, contract.FechaFin)), 52)
	if contract.MontoDeposito > 0 {
		kv(t("Depósito de garantía:"), formatMonto(contract.MontoDeposito, moneda)+" (reembolsable al finalizar el contrato)", 52)
	} else {
		kv(t("Depósito de garantía:"), "Sin depósito", 52)
	}

	// ===== IV. CLÁUSULAS =====
	seccion("IV. CLÁUSULAS")
	clausulas := []string{
		fmt.Sprintf("PRIMERA.- El ARRENDADOR cede en alquiler al ARRENDATARIO el inmueble descrito, por el plazo indicado, a partir del %s.", fechaLarga(contract.FechaInicio)),
		fmt.Sprintf("SEGUNDA.- El ARRENDATARIO pagará la renta mensual de %s, por adelantado, dentro de los primeros días de cada mes.", formatMonto(contract.MontoMensual, moneda)),
		"TERCERA.- El ARRENDATARIO destinará el inmueble exclusivamente a vivienda, no pudiendo subarrendarlo ni cederlo sin autorización escrita del ARRENDADOR.",
		"CUARTA.- El ARRENDATARIO se hace cargo de los servicios básicos (luz, agua, gas, internet) y del mantenimiento por uso ordinario del inmueble.",
		"QUINTA.- El depósito de garantía se devolverá al ARRENDATARIO al finalizar el contrato, previa verificación del estado del inmueble y cumplimiento de sus obligaciones.",
		"SEXTA.- El incumplimiento de pago de dos o más cuotas faculta al ARRENDADOR a resolver el contrato y exigir la desocupación del inmueble.",
		"SEPTIMA.- Para la resolución de cualquier controversia, las partes se someten a la legislación boliviana y a la mediación competente.",
	}
	fuente("", 9.5)
	for _, clausula := range clausulas {
		pdf.MultiCell(width, 5.4, t(clausula), "", "J", false)
		pdf.Ln(1.5)
	}

	// ===== V. ACEPTACIÓN Y FIRMA =====
	if pdf.GetY() > 215 {
		pdf.AddPage()
	}
	seccion("V. ACEPTACIÓN Y FIRMA ELECTRÓNICA")

	pdf.Ln(8)
	firmaY := pdf.GetY()

	// Nombre + línea + aceptación electrónica por parte
	firmaParte := func(x float64, rol, nombre, ci string) {
		fuente("B", 9.5)
		pdf.SetXY(x, firmaY)
		pdf.CellFormat(85, 5, nombre, "", 2, "C", false, 0, "")
		pdf.SetDrawColor(60, 60, 60)
		pdf.Line(x+12, firmaY+9, x+73, firmaY+9)
		fuente("", 8.5)
		pdf.SetXY(x, firmaY+11)
		pdf.CellFormat(85, 4.5, t(rol), "", 2, "C", false, 0, "")
		pdf.SetXY(x, firmaY+15.5)
		pdf.CellFormat(85, 4.5, t("CI: ")+ci, "", 2, "C", false, 0, "")
		pdf.SetXY(x, firmaY+20)
		fuente("I", 7.5)
		pdf.SetTextColor(110, 120, 140)
		pdf.CellFormat(85, 4, t("Firma electrónica de aceptación"), "", 2, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	}
	firmaParte(15, "ARRENDADOR", strings.TrimSpace(contract.Vendedor.Nombre+" "+contract.Vendedor.Apellido), contract.Vendedor.CI)
	firmaParte(110, "ARRENDATARIO", strings.TrimSpace(contract.Inquilino.Nombre+" "+contract.Inquilino.Apellido), contract.Inquilino.CI)

	pdf.SetY(firmaY + 28)
	pdf.Ln(2)

	// Código de verificación del documento
	pdf.SetFillColor(248, 250, 252)
	pdf.SetDrawColor(210, 218, 230)
	pdf.RoundedRect(15, pdf.GetY(), width, 14, 2, "1234", "FD")
	fuente("B", 8.5)
	pdf.SetXY(15, pdf.GetY()+2.5)
	pdf.CellFormat(width, 4.5, t("CÓDIGO DE VERIFICACIÓN DEL DOCUMENTO"), "", 2, "C", false, 0, "")
	fuente("", 9)
	pdf.SetX(15)
	pdf.CellFormat(width, 5, codigoFormateado(), "", 2, "C", false, 0, "")

	// Nota legal
	pdf.Ln(3)
	fuente("I", 7.5)
	pdf.SetTextColor(110, 120, 140)
	pdf.MultiCell(width, 4,
		t(fmt.Sprintf("Documento generado electrónicamente por 365Soft el %s. Ambas partes manifestaron su aceptación a través de la plataforma. La autenticidad del documento puede verificarse con el código anterior, calculado mediante SHA-256 sobre los datos esenciales del contrato.", time.Now().Format("02/01/2006 15:04"))),
		"", "C", false)
	pdf.SetTextColor(0, 0, 0)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// mesesEntre - Cantidad de meses completos entre dos fechas
func mesesEntre(inicio, fin time.Time) int {
	meses := (fin.Year()-inicio.Year())*12 + int(fin.Month()) - int(inicio.Month())
	if fin.Day() < inicio.Day() && meses > 0 {
		meses--
	}
	if meses < 1 {
		meses = 1
	}
	return meses
}
