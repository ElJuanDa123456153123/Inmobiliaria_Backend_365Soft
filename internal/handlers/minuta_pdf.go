package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/go-pdf/fpdf"
)

// etiquetaTipoPago - Traducción legible del tipo de pago de venta
func etiquetaTipoPago(tipo string) string {
	switch tipo {
	case "unico":
		return "Pago único"
	case "arras":
		return "Arras"
	case "saldo":
		return "Saldo"
	}
	return tipo
}

// etiquetaEstadoPago - Traducción legible del estado de un pago de venta
func etiquetaEstadoPago(estado models.SalePaymentStatus) string {
	switch estado {
	case models.SalePaymentStatusPendiente:
		return "Pendiente"
	case models.SalePaymentStatusComprobante:
		return "Comprobante enviado"
	case models.SalePaymentStatusConfirmado:
		return "Confirmado"
	}
	return string(estado)
}

// generarPDFMinuta - Genera el PDF de la Minuta/Promesa de Compraventa de una
// venta aceptada, con el mismo formato legal del contrato de alquiler: título
// formal, partes con CI, descripción del inmueble, precio y PLAN DE PAGOS,
// cláusulas numeradas justificadas, advertencia legal, firmas alineadas y
// código de verificación SHA-256.
func generarPDFMinuta(solicitud *models.PurchaseRequest) ([]byte, error) {
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

	// Fecha del acuerdo (aceptación de la oferta; respaldo: creación de la solicitud)
	fechaAcuerdo := time.Now()
	if solicitud.FechaAceptada != nil {
		fechaAcuerdo = *solicitud.FechaAceptada
	} else {
		fechaAcuerdo = solicitud.CreatedAt
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
		pdf.CellFormat(ancho/2, 4.5, t(fmt.Sprintf("Minuta Nro. %d  ·  Página %d de %d",
			solicitud.ID, pdf.PageNo(), pdf.PageCount())), "", 0, "R", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	})

	pdf.AddPage()

	// Código de verificación: hash SHA-256 de los datos clave de la venta
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d|%s|%s|%.2f|%s|%s|%s",
		solicitud.ID,
		solicitud.Vendedor.CI, solicitud.Comprador.CI,
		solicitud.MontoAceptado,
		fechaAcuerdo.Format("2006-01-02"),
		solicitud.Property.Titulo, solicitud.Property.Moneda,
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

	fuente("B", 14)
	pdf.SetX(margenX)
	pdf.CellFormat(ancho, 8, t("MINUTA DE COMPRAVENTA DE INMUEBLE"), "", 2, "C", false, 0, "")
	fuente("", 10)
	pdf.SetX(margenX)
	pdf.CellFormat(ancho, 5.5, t(fmt.Sprintf("Promesa de compraventa  ·  Minuta Nro. %d", solicitud.ID)), "", 2, "C", false, 0, "")
	pdf.Ln(1)
	pdf.SetX(margenX)
	pdf.CellFormat(ancho, 5.5, t("Cochabamba, "+fechaLarga(fechaAcuerdo)), "", 2, "R", false, 0, "")
	pdf.Ln(1.5)
	pdf.SetDrawColor(0, 0, 0)
	pdf.SetLineWidth(0.5)
	pdf.Line(margenX, pdf.GetY(), margenX+ancho, pdf.GetY())
	pdf.Ln(6)

	// ===== I. PARTES =====
	seccion("I. PARTES DE LA COMPRAVENTA")

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
	parrafo(descripcionParte(solicitud.Vendedor.Nombre, solicitud.Vendedor.Apellido, solicitud.Vendedor.CI,
		solicitud.Vendedor.Telefono, solicitud.Vendedor.Email)+", en adelante EL VENDEDOR;", "J")
	pdf.Ln(1)
	parrafo(descripcionParte(solicitud.Comprador.Nombre, solicitud.Comprador.Apellido, solicitud.Comprador.CI,
		solicitud.Comprador.Telefono, solicitud.Comprador.Email)+", en adelante EL COMPRADOR;", "J")
	pdf.Ln(1)
	parrafo("quienes, en pleno uso de sus facultades legales, convienen en celebrar la presente minuta de compraventa de inmueble sobre la base de la oferta aceptada a través de la plataforma 365Soft, conforme a las cláusulas y condiciones siguientes:", "J")

	// ===== II. INMUEBLE =====
	seccion("II. DEL INMUEBLE OBJETO DE LA COMPRAVENTA")
	direccion := solicitud.Property.Direccion
	if direccion == "" {
		direccion = "(dirección no especificada)"
	}
	zona := solicitud.Property.Zona
	if zona == "" {
		zona = "(zona no especificada)"
	}
	descripcionInmueble := fmt.Sprintf("El inmueble materia de la presente compraventa es el señalado como \"%s\", del tipo %s, ubicado en %s, zona %s.",
		solicitud.Property.Titulo, solicitud.Property.TipoPropiedad, direccion, zona)
	if solicitud.Property.MetrosTerreno > 0 {
		descripcionInmueble += fmt.Sprintf(" Superficie de terreno: %.2f m².", solicitud.Property.MetrosTerreno)
	}
	if solicitud.Property.MetrosEdificados > 0 {
		descripcionInmueble += fmt.Sprintf(" Superficie edificada: %.2f m².", solicitud.Property.MetrosEdificados)
	}
	descripcionInmueble += " EL COMPRADOR declara haber conocido el inmueble y aceptarlo en el estado en que se encuentra."
	parrafo(descripcionInmueble, "J")

	// ===== III. PRECIO Y PLAN DE PAGOS =====
	seccion("III. DEL PRECIO Y PLAN DE PAGOS")
	moneda := solicitud.Property.Moneda
	filaTabla := func(etiqueta, valor string) {
		fuente("B", 9.5)
		pdf.SetTextColor(0, 0, 0)
		pdf.SetX(margenX)
		pdf.SetFillColor(242, 242, 242)
		pdf.CellFormat(58, 7.2, t(etiqueta), "1", 0, "L", true, 0, "")
		fuente("", 9.5)
		pdf.CellFormat(ancho-58, 7.2, t(valor), "1", 1, "L", false, 0, "")
	}
	filaTabla("Precio acordado de la venta:", formatMonto(solicitud.MontoAceptado, moneda))
	filaTabla("Fecha del acuerdo:", fechaLarga(fechaAcuerdo))
	pdf.Ln(3)

	// Tabla del plan de pagos (cada pago registrado de la venta)
	fuente("B", 8.5)
	pdf.SetFillColor(242, 242, 242)
	pdf.SetX(margenX)
	pdf.CellFormat(40, 7, t("TIPO"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 7, t("MONTO"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(45, 7, t("VENCIMIENTO"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(45, 7, t("ESTADO"), "1", 1, "C", true, 0, "")
	fuente("", 8.5)
	if len(solicitud.Pagos) == 0 {
		pdf.SetX(margenX)
		pdf.CellFormat(ancho, 7, t("Sin pagos registrados"), "1", 1, "C", false, 0, "")
	}
	for _, pago := range solicitud.Pagos {
		vencimiento := "(sin fecha definida)"
		if pago.FechaVencimiento != nil {
			vencimiento = pago.FechaVencimiento.Format("02/01/2006")
		}
		pdf.SetX(margenX)
		pdf.CellFormat(40, 7, t(etiquetaTipoPago(pago.Tipo)), "1", 0, "C", false, 0, "")
		pdf.CellFormat(40, 7, t(formatMonto(pago.Monto, moneda)), "1", 0, "C", false, 0, "")
		pdf.CellFormat(45, 7, t(vencimiento), "1", 0, "C", false, 0, "")
		pdf.CellFormat(45, 7, t(etiquetaEstadoPago(pago.Estado)), "1", 1, "C", false, 0, "")
	}

	// ===== IV. CLÁUSULAS =====
	seccion("IV. CLÁUSULAS")

	// Detalle textual del plan de pagos para la cláusula segunda
	var detallePagos strings.Builder
	for i, pago := range solicitud.Pagos {
		if i > 0 {
			if i == len(solicitud.Pagos)-1 {
				detallePagos.WriteString(" y ")
			} else {
				detallePagos.WriteString(", ")
			}
		}
		detallePagos.WriteString(fmt.Sprintf("%s de %s", strings.ToLower(etiquetaTipoPago(pago.Tipo)), formatMonto(pago.Monto, moneda)))
		if pago.FechaVencimiento != nil {
			detallePagos.WriteString(" con vencimiento el " + fechaLarga(*pago.FechaVencimiento))
		} else {
			detallePagos.WriteString(" sin fecha de vencimiento definida entre las partes")
		}
	}

	clausulas := []string{
		fmt.Sprintf("PRIMERA.- El precio total acordado por la venta del inmueble descrito en esta minuta es de %s, suma que EL COMPRADOR paga o se obliga a pagar a EL VENDEDOR conforme al plan de pagos detallado en esta minuta.", formatMonto(solicitud.MontoAceptado, moneda)),
		"SEGUNDA.- EL COMPRADOR efectuará el pago del precio según el siguiente plan: " + detallePagos.String() + ". Cada pago será registrado en la plataforma 365Soft con su comprobante y quedará firmado por EL VENDEDOR al confirmar su recepción.",
		"TERCERA.- EL VENDEDOR se obliga a entregar el inmueble a EL COMPRADOR, libre de gravámenes, arrendatarios y ocupantes, con el pago total del precio, junto con las llaves, planillas de servicios básicos y documentación de su posesión.",
		"CUARTA.- EL VENDEDOR declara que la documentación del inmueble (cédula de identidad, certificado de gravámenes, ficha catastral e impuesto IPVA al día) será presentada y mantenida vigente durante el proceso de la compraventa, conforme al checklist documental registrado en la plataforma.",
		"QUINTA.- Las partes se obligan a suscribir la escritura pública de transferencia ante notario de fe pública una vez cumplido el plan de pagos, siendo los gastos de otorgamiento de la escritura y de registro a cargo de quien corresponda según la ley y lo que las partes convengan.",
		"SEXTA.- Si EL COMPRADOR incumpliera el pago de las arras o del saldo en las fechas pactadas, EL VENDEDOR podrá resolver la presente minuta, sin perjuicio del resarcimiento que corresponda conforme a ley.",
		"SÉPTIMA.- Para todos los efectos derivados de la presente minuta, las partes se someten a la legislación boliviana y a la autoridad competente del lugar donde se encuentra el inmueble.",
		"OCTAVA.- Las partes aceptan que la firma electrónica aplicada a través de la plataforma tiene la misma validez y eficacia legal que la firma manuscrita, y que el presente documento digital reproduce fielmente el acuerdo alcanzado.",
	}
	for _, clausula := range clausulas {
		parrafo(clausula, "J")
		pdf.Ln(1.5)
	}

	// ===== ADVERTENCIA LEGAL (obligatoria) =====
	if pdf.GetY() > 235 {
		pdf.AddPage()
	}
	pdf.Ln(2)
	pdf.SetDrawColor(120, 120, 120)
	pdf.SetLineWidth(0.4)
	advY := pdf.GetY()
	pdf.SetFillColor(245, 245, 245)
	pdf.Rect(margenX, advY, ancho, 18, "FD")
	fuente("B", 8.5)
	pdf.SetXY(margenX, advY+2.5)
	pdf.CellFormat(ancho, 4.5, t("ADVERTENCIA LEGAL"), "", 2, "C", false, 0, "")
	fuente("I", 8)
	pdf.SetXY(margenX+6, advY+7.5)
	pdf.MultiCell(ancho-12, 4,
		t("Documento privado previo; la transferencia de propiedad se formaliza mediante escritura pública ante notario de fe pública y se registra en Derechos Reales."),
		"", "C", false)
	pdf.SetY(advY + 18)

	// ===== V. FIRMAS =====
	if pdf.GetY() > 232 {
		pdf.AddPage()
	}
	seccion("V. ACEPTACIÓN Y FIRMA")
	parrafo(fmt.Sprintf("En la ciudad de Cochabamba, a los %d días del mes de %s del año %d, previa lectura de la presente minuta, las partes la aceptan en todas sus partes y la firman de conformidad, en dos ejemplares de igual tenor y a un solo efecto.",
		fechaAcuerdo.Day(), mesesEs[int(fechaAcuerdo.Month())-1], fechaAcuerdo.Year()), "J")

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
	firmaParte(margenX, "EL VENDEDOR",
		strings.TrimSpace(solicitud.Vendedor.Nombre+" "+solicitud.Vendedor.Apellido), solicitud.Vendedor.CI)
	firmaParte(margenX+100, "EL COMPRADOR",
		strings.TrimSpace(solicitud.Comprador.Nombre+" "+solicitud.Comprador.Apellido), solicitud.Comprador.CI)
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
		t("La autenticidad de este documento puede verificarse contrastando el código anterior, calculado mediante el algoritmo SHA-256 sobre los datos esenciales de la venta (partes, inmueble, precio y plan de pagos)."),
		"", "C", false)
	pdf.SetTextColor(0, 0, 0)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
