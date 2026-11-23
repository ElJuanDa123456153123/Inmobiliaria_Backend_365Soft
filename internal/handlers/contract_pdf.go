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

// generarPDFContrato - Genera el PDF del contrato con formato de plantilla
// legal oficial: título formal, declaración de partes, descripción del
// inmueble, tabla de condiciones económicas, cláusulas numeradas y firmas.
func generarPDFContrato(contract *models.RentalContract) ([]byte, error) {
	const margenX = 20.0
	const ancho = 170.0 // ancho útil A4 (210mm) con márgenes de 20mm
	const grisTexto = 100
	const grisFooter = 130

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(margenX, 18, margenX)
	pdf.SetAutoPageBreak(true, 28)
	pdf.AliasNbPages("")

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

	// Pie de página en todas las hojas: regla fina + numeración
	pdf.SetFooterFunc(func() {
		pdf.SetY(-17)
		pdf.SetDrawColor(150, 150, 150)
		pdf.SetLineWidth(0.2)
		pdf.Line(margenX, pdf.GetY(), margenX+ancho, pdf.GetY())
		pdf.SetY(-14)
		fuente("", 7.5)
		pdf.SetTextColor(grisFooter, grisFooter, grisFooter)
		pdf.SetX(margenX)
		pdf.CellFormat(ancho/2, 4.5, t("365Soft Inmobiliaria"), "", 0, "L", false, 0, "")
		pdf.CellFormat(ancho/2, 4.5, t(fmt.Sprintf("Contrato Nro. %d  ·  Página %d de %d",
			contract.ID, pdf.PageNo(), pdf.PageCount())), "", 0, "R", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	})

	pdf.AddPage()

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

	// Helper: párrafo de cuerpo con ajuste de línea
	parrafo := func(s string, alineacion string) {
		fuente("", 10)
		pdf.SetTextColor(0, 0, 0)
		pdf.SetX(margenX)
		pdf.MultiCell(ancho, 5.6, t(s), "", alineacion, false)
	}

	// Helper: título de sección numerado, centrado y subrayado con regla fina
	seccion := func(titulo string) {
		pdf.Ln(4)
		fuente("B", 10.5)
		pdf.SetTextColor(0, 0, 0)
		pdf.SetX(margenX)
		pdf.CellFormat(ancho, 6, t(titulo), "", 2, "C", false, 0, "")
		pdf.SetDrawColor(60, 60, 60)
		pdf.SetLineWidth(0.3)
		pdf.Line(margenX+40, pdf.GetY()+0.8, margenX+ancho-40, pdf.GetY()+0.8)
		pdf.Ln(3.5)
	}

	// ===== ENCABEZADO =====
	fuente("I", 8)
	pdf.SetTextColor(grisTexto, grisTexto, grisTexto)
	pdf.SetX(margenX)
	pdf.CellFormat(ancho, 4, t("365Soft Inmobiliaria  ·  Plataforma de gestión inmobiliaria"), "", 2, "R", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(2)

	tituloDoc := "CONTRATO DE ALQUILER DE INMUEBLE"
	if contract.TipoContrato == models.ContractTypeAnticretico {
		tituloDoc = "CONTRATO DE ANTICRÉTICO DE INMUEBLE"
	}
	fuente("B", 14)
	pdf.SetX(margenX)
	pdf.CellFormat(ancho, 8, t(tituloDoc), "", 2, "C", false, 0, "")
	fuente("", 10)
	pdf.SetX(margenX)
	pdf.CellFormat(ancho, 5.5, t(fmt.Sprintf("Contrato Nro. %d", contract.ID)), "", 2, "C", false, 0, "")
	pdf.Ln(1)
	pdf.SetX(margenX)
	pdf.CellFormat(ancho, 5.5, t("Cochabamba, "+fechaLarga(contract.FechaInicio)), "", 2, "R", false, 0, "")
	pdf.Ln(1.5)
	pdf.SetDrawColor(0, 0, 0)
	pdf.SetLineWidth(0.5)
	pdf.Line(margenX, pdf.GetY(), margenX+ancho, pdf.GetY())
	pdf.Ln(6)

	// ===== I. PARTES =====
	seccion("I. PARTES DEL CONTRATO")

	descripcionParte := func(nombre, apellido, ci, tel, email string) string {
		var b strings.Builder
		b.WriteString("Don " + strings.ToUpper(strings.TrimSpace(nombre+" "+apellido)) + ", mayor de edad")
		if strings.TrimSpace(ci) != "" {
			b.WriteString(", con Cédula de Identidad Nro. " + ci)
		}
		if strings.TrimSpace(tel) != "" {
			b.WriteString(", teléfono " + tel)
		}
		if strings.TrimSpace(email) != "" {
			b.WriteString(", correo electrónico " + email)
		}
		return b.String()
	}
	parrafo(descripcionParte(contract.Vendedor.Nombre, contract.Vendedor.Apellido, contract.Vendedor.CI,
		contract.Vendedor.Telefono, contract.Vendedor.Email)+", en adelante EL ARRENDADOR;", "J")
	pdf.Ln(1)
	parrafo(descripcionParte(contract.Inquilino.Nombre, contract.Inquilino.Apellido, contract.Inquilino.CI,
		contract.Inquilino.Telefono, contract.Inquilino.Email)+", en adelante EL ARRENDATARIO;", "J")
	pdf.Ln(1)
	introContrato := "quienes, en pleno uso de sus facultades legales, convienen en celebrar el presente contrato de alquiler de inmueble, conforme a las cláusulas y condiciones siguientes:"
	if contract.TipoContrato == models.ContractTypeAnticretico {
		introContrato = "quienes, en pleno uso de sus facultades legales, convienen en celebrar el presente contrato de anticrético de inmueble, conforme a las cláusulas y condiciones siguientes:"
	}
	parrafo(introContrato, "J")

	// ===== II. INMUEBLE =====
	seccion("II. DEL INMUEBLE OBJETO DEL CONTRATO")
	direccion := contract.Property.Direccion
	if direccion == "" {
		direccion = "(dirección no especificada)"
	}
	zona := contract.Property.Zona
	if zona == "" {
		zona = "(zona no especificada)"
	}
	parrafo(fmt.Sprintf("El inmueble materia del presente contrato es el señalado como \"%s\", del tipo %s, ubicado en %s, zona %s. "+
		"El ARRENDATARIO declara conocer el inmueble y recibirlo en buen estado de conservación, obligándose a devolverlo en las mismas condiciones al finalizar el contrato, salvo el deterioro producto del uso ordinario.",
		contract.Property.Titulo, contract.Property.TipoPropiedad, direccion, zona), "J")

	// ===== III. CONDICIONES ECONÓMICAS =====
	seccion("III. CONDICIONES ECONÓMICAS")
	moneda := contract.Property.Moneda
	filaTabla := func(etiqueta, valor string) {
		fuente("B", 9.5)
		pdf.SetTextColor(0, 0, 0)
		pdf.SetX(margenX)
		pdf.SetFillColor(242, 242, 242)
		pdf.CellFormat(58, 7.2, t(etiqueta), "1", 0, "L", true, 0, "")
		fuente("", 9.5)
		pdf.CellFormat(ancho-58, 7.2, t(valor), "1", 1, "L", false, 0, "")
	}
	if contract.TipoContrato == models.ContractTypeAnticretico {
		filaTabla("Anticrético:", formatMonto(contract.MontoDeposito, moneda))
	} else {
		filaTabla("Renta mensual:", formatMonto(contract.MontoMensual, moneda))
	}
	filaTabla("Plazo del contrato:", fmt.Sprintf("%d meses", mesesEntre(contract.FechaInicio, contract.FechaFin)))
	filaTabla("Fecha de inicio:", fechaLarga(contract.FechaInicio))
	filaTabla("Fecha de finalización:", fechaLarga(contract.FechaFin))
	if contract.MontoDeposito > 0 && contract.TipoContrato != models.ContractTypeAnticretico {
		filaTabla("Depósito de garantía:", formatMonto(contract.MontoDeposito, moneda)+" (reembolsable al finalizar el contrato)")
	} else {
		filaTabla("Depósito de garantía:", "Sin depósito")
	}
	filaTabla("Destino del inmueble:", "Exclusivamente vivienda")

	// ===== IV. CLÁUSULAS =====
	seccion("IV. CLÁUSULAS")
	var clausulas []string
	if contract.TipoContrato == models.ContractTypeAnticretico {
		clausulas = []string{
			fmt.Sprintf("PRIMERA.- El ARRENDADOR recibe del ARRENDATARIO la suma de %s en calidad de anticrético, cantidad que será devuelta al ARRENDATARIO de una sola vez al finalizar el contrato, a la devolución del inmueble, sin lugar a reembolso o reclamo adicional alguno.", formatMonto(contract.MontoDeposito, moneda)),
			fmt.Sprintf("SEGUNDA.- El presente contrato se celebra por el plazo indicado, a partir del %s, no existiendo pago de renta mensual entre las partes.", fechaLarga(contract.FechaInicio)),
			"TERCERA.- El ARRENDATARIO destinará el inmueble exclusivamente a vivienda, no pudiendo subarrendarlo total ni parcialmente, ni ceder sus derechos sobre el presente contrato, sin autorización previa y escrita del ARRENDADOR.",
			"CUARTA.- El ARRENDATARIO se hará cargo del pago de los servicios básicos del inmueble (energía eléctrica, agua, gas e internet) y del mantenimiento por uso ordinario del mismo.",
			"QUINTA.- Al finalizar el contrato, el ARRENDATARIO desocupará el inmueble y entregará las llaves al ARRENDADOR, quien verificará el estado del inmueble antes de efectuar la devolución del anticrético.",
			"SEXTA.- Cualquiera de las partes podrá dar por concluido el presente contrato de manera anticipada, mediante notificación escrita cursada con al menos treinta días de anticipación.",
			"SÉPTIMA.- Para todos los efectos derivados del presente contrato, las partes se someten a la legislación boliviana y a la autoridad competente del lugar donde se encuentra el inmueble.",
			"OCTAVA.- Las partes aceptan que la firma electrónica aplicada a través de la plataforma tiene la misma validez y eficacia legal que la firma manuscrita, y que el presente documento digital reproduce fielmente el acuerdo alcanzado.",
		}
	} else {
		clausulas = []string{
			fmt.Sprintf("PRIMERA.- El ARRENDADOR cede en alquiler al ARRENDATARIO el inmueble descrito en este contrato, por el plazo indicado, a partir del %s.", fechaLarga(contract.FechaInicio)),
			fmt.Sprintf("SEGUNDA.- El ARRENDATARIO pagará la renta mensual de %s, por adelantado, dentro de los primeros cinco días de cada mes.", formatMonto(contract.MontoMensual, moneda)),
			"TERCERA.- El ARRENDATARIO destinará el inmueble exclusivamente a vivienda, no pudiendo subarrendarlo total ni parcialmente, ni ceder sus derechos sobre el presente contrato, sin autorización previa y escrita del ARRENDADOR.",
			"CUARTA.- El ARRENDATARIO se hará cargo del pago de los servicios básicos del inmueble (energía eléctrica, agua, gas e internet) y del mantenimiento por uso ordinario del mismo.",
			"QUINTA.- El depósito de garantía se devolverá al ARRENDATARIO al finalizar el contrato, previa verificación del estado del inmueble y del cumplimiento de todas las obligaciones asumidas.",
			"SEXTA.- El incumplimiento del pago de dos o más cuotas facultará al ARRENDADOR a resolver el contrato de manera anticipada y exigir la desocupación del inmueble, sin perjuicio de las acciones legales que correspondan.",
			"SÉPTIMA.- Para todos los efectos derivados del presente contrato, las partes se someten a la legislación boliviana y a la autoridad competente del lugar donde se encuentra el inmueble.",
			"OCTAVA.- Las partes aceptan que la firma electrónica aplicada a través de la plataforma tiene la misma validez y eficacia legal que la firma manuscrita, y que el presente documento digital reproduce fielmente el acuerdo alcanzado.",
		}
	}
	for _, clausula := range clausulas {
		parrafo(clausula, "J")
		pdf.Ln(1.5)
	}

	// ===== V. FIRMAS =====
	if pdf.GetY() > 232 {
		pdf.AddPage()
	}
	seccion("V. ACEPTACIÓN Y FIRMA")
	parrafo(fmt.Sprintf("En la ciudad de Cochabamba, a los %d días del mes de %s del año %d, previa lectura del presente contrato, las partes lo aceptan en todas sus partes y lo firman de conformidad, en dos ejemplares de igual tenor y a un solo efecto.",
		contract.FechaInicio.Day(), mesesEs[int(contract.FechaInicio.Month())-1], contract.FechaInicio.Year()), "J")

	pdf.Ln(7)
	// Ambas columnas de firma se dibujan a la MISMA altura (firmaY explícita)
	firmaY := pdf.GetY()
	firmaParte := func(x float64, rol, nombre, ci string) {
		fuente("", 9)
		pdf.SetTextColor(0, 0, 0)
		pdf.SetXY(x, firmaY)
		pdf.CellFormat(70, 5, t(rol), "", 0, "C", false, 0, "")
		pdf.SetDrawColor(0, 0, 0)
		pdf.SetLineWidth(0.35)
		pdf.Line(x+5, firmaY+12, x+65, firmaY+12)
		fuente("B", 9.5)
		pdf.SetXY(x, firmaY+14)
		pdf.CellFormat(70, 5, nombre, "", 0, "C", false, 0, "")
		ciTexto := "C.I. Nro. " + ci
		if strings.TrimSpace(ci) == "" {
			ciTexto = "C.I. Nro. ......................."
		}
		fuente("", 8.5)
		pdf.SetXY(x, firmaY+18.5)
		pdf.CellFormat(70, 4.5, t(ciTexto), "", 0, "C", false, 0, "")
		fuente("I", 7.5)
		pdf.SetTextColor(grisTexto, grisTexto, grisTexto)
		pdf.SetXY(x, firmaY+23)
		pdf.CellFormat(70, 4, t("Firma electrónica aplicada en la plataforma"), "", 0, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	}
	firmaParte(margenX, "EL ARRENDADOR",
		strings.TrimSpace(contract.Vendedor.Nombre+" "+contract.Vendedor.Apellido), contract.Vendedor.CI)
	firmaParte(margenX+100, "EL ARRENDATARIO",
		strings.TrimSpace(contract.Inquilino.Nombre+" "+contract.Inquilino.Apellido), contract.Inquilino.CI)
	// Continuar debajo de la columna de firma más profunda
	pdf.SetY(firmaY + 29)

	// Nota de generación electrónica
	fuente("I", 8)
	pdf.SetTextColor(grisTexto, grisTexto, grisTexto)
	pdf.SetX(margenX)
	pdf.CellFormat(ancho, 4.5, t(fmt.Sprintf("Documento firmado electrónicamente a través de 365Soft el %s a horas %s.",
		time.Now().Format("02/01/2006"), time.Now().Format("15:04"))), "", 2, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(4)

	// Código de verificación del documento (si no cabe completo, página nueva)
	if pdf.GetY() > 250 {
		pdf.AddPage()
	}
	pdf.SetDrawColor(0, 0, 0)
	pdf.SetLineWidth(0.35)
	boxY := pdf.GetY()
	pdf.Rect(margenX, boxY, ancho, 17, "D")
	fuente("B", 8)
	pdf.SetXY(margenX, boxY+2.5)
	pdf.CellFormat(ancho, 4, t("CÓDIGO DE VERIFICACIÓN DEL DOCUMENTO (SHA-256)"), "", 2, "C", false, 0, "")
	fuente("B", 9)
	pdf.SetX(margenX)
	pdf.CellFormat(ancho, 5, codigoFormateado(), "", 2, "C", false, 0, "")

	// Nota legal (debajo del recuadro, sin cruzar el borde)
	pdf.SetY(boxY + 17)
	pdf.Ln(3)
	fuente("I", 7.5)
	pdf.SetTextColor(grisTexto, grisTexto, grisTexto)
	pdf.SetX(margenX)
	pdf.MultiCell(ancho, 4,
		t("La autenticidad de este documento puede verificarse contrastando el código anterior, calculado mediante el algoritmo SHA-256 sobre los datos esenciales del contrato (partes, fechas, inmueble y montos)."),
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
