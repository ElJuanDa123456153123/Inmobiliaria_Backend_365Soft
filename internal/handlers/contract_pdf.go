package handlers

import (
	"bytes"
	"fmt"
	"strings"
	"time"

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

// generarPDFContrato - Genera el PDF de un contrato de alquiler en memoria
func generarPDFContrato(contract *models.RentalContract) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()

	width := 190.0 // ancho útil A4 con márgenes de 10mm

	// ===== ENCABEZADO =====
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(width, 10, "CONTRATO DE ALQUILER DE INMUEBLE", "0", 1, "C", false, 0, "")
	pdf.Ln(2)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(width, 6, fmt.Sprintf("Contrato Nro. %d - 365Soft Inmobiliaria", contract.ID), "0", 1, "C", false, 0, "")
	pdf.Ln(6)

	// ===== PARTES =====
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(width, 7, "PARTES", "0", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)

	partes := [][]string{
		{"ARRENDADOR (Propietario):", contract.Vendedor.Nombre + " " + contract.Vendedor.Apellido},
		{"CI:", contract.Vendedor.CI},
		{"Teléfono:", contract.Vendedor.Telefono},
		{"Email:", contract.Vendedor.Email},
		{"", ""},
		{"ARRENDATARIO (Inquilino):", contract.Inquilino.Nombre + " " + contract.Inquilino.Apellido},
		{"CI:", contract.Inquilino.CI},
		{"Teléfono:", contract.Inquilino.Telefono},
		{"Email:", contract.Inquilino.Email},
	}
	for _, row := range partes {
		if row[0] == "" {
			pdf.Ln(2)
			continue
		}
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(50, 6, row[0], "0", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(width-50, 6, row[1], "0", 1, "L", false, 0, "")
	}
	pdf.Ln(4)

	// ===== INMUEBLE =====
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(width, 7, "INMUEBLE OBJETO DEL CONTRATO", "0", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	direccion := contract.Property.Direccion
	if direccion == "" {
		direccion = "(Dirección no especificada)"
	}
	inmueble := [][]string{
		{"Título:", contract.Property.Titulo},
		{"Tipo:", string(contract.Property.TipoPropiedad)},
		{"Dirección:", direccion},
		{"Zona:", contract.Property.Zona},
	}
	for _, row := range inmueble {
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(35, 6, row[0], "0", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "", 10)
		pdf.MultiCell(width-35, 6, row[1], "0", "L", false)
	}
	pdf.Ln(4)

	// ===== CLÁUSULAS ECONÓMICAS =====
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(width, 7, "CLÁUSULAS ECONÓMICAS", "0", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)

	moneda := contract.Property.Moneda
	economicas := [][]string{
		{"Renta mensual:", formatMonto(contract.MontoMensual, moneda)},
		{"Fecha de inicio:", fechaLarga(contract.FechaInicio)},
		{"Fecha de finalización:", fechaLarga(contract.FechaFin)},
		{"Plazo:", fmt.Sprintf("%d meses", mesesEntre(contract.FechaInicio, contract.FechaFin))},
	}
	if contract.MontoDeposito > 0 {
		economicas = append(economicas, []string{"Depósito de garantía:", formatMonto(contract.MontoDeposito, moneda) + " (reembolsable al finalizar el contrato)"})
	} else {
		economicas = append(economicas, []string{"Depósito de garantía:", "Sin depósito"})
	}
	for _, row := range economicas {
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(50, 6, row[0], "0", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(width-50, 6, row[1], "0", 1, "L", false, 0, "")
	}
	pdf.Ln(4)

	// ===== CLÁUSULAS =====
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(width, 7, "CLÁUSULAS", "0", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)

	clausulas := []string{
		fmt.Sprintf("PRIMERA.- El ARRENDADOR cede en alquiler al ARRENDATARIO el inmueble descrito, por el plazo indicado, a partir del %s.", fechaLarga(contract.FechaInicio)),
		fmt.Sprintf("SEGUNDA.- El ARRENDATARIO pagará la renta mensual de %s, por adelantado, dentro de los primeros días de cada mes.", formatMonto(contract.MontoMensual, moneda)),
		"TERCERA.- El ARRENDATARIO destinará el inmueble exclusivamente a vivienda, no pudiendo subarrendarlo ni cederlo sin autorización escrita del ARRENDADOR.",
		"CUARTA.- El ARRENDATARIO se hace cargo de los servicios básicos (luz, agua, gas, internet) y del mantenimiento por uso ordinario del inmueble.",
		"QUINTA.- El depósito de garantía se devolverá al ARRENDATARIO al finalizar el contrato, previa verificación del estado del inmueble y cumplimiento de sus obligaciones.",
		"SEXTA.- El incumplimiento de pago de dos o más cuotas faculta al ARRENDADOR a resolver el contrato y exigir la desocupación del inmueble.",
		"SEPTIMA.- Para la resolución de cualquier controversia, las partes se someten a la legislación boliviana y a la mediación competente.",
	}
	for _, clausula := range clausulas {
		pdf.MultiCell(width, 6, clausula, "0", "J", false)
		pdf.Ln(1)
	}
	pdf.Ln(6)

	// ===== FIRMAS =====
	if pdf.GetY() > 230 {
		pdf.AddPage()
	}
	firmaY := pdf.GetY()
	pdf.Line(20, firmaY, 85, firmaY)
	pdf.Line(110, firmaY, 175, firmaY)
	pdf.SetFont("Arial", "", 9)
	pdf.SetXY(20, firmaY+2)
	pdf.CellFormat(65, 5, "ARRENDADOR", "0", 0, "C", false, 0, "")
	pdf.SetXY(110, firmaY+2)
	pdf.CellFormat(65, 5, "ARRENDATARIO", "0", 1, "C", false, 0, "")

	pdf.Ln(10)
	pdf.SetFont("Arial", "I", 8)
	generado := fmt.Sprintf("Documento generado electrónicamente por 365Soft el %s.", time.Now().Format("02/01/2006 15:04"))
	pdf.CellFormat(width, 5, strings.TrimSpace(generado), "0", 1, "C", false, 0, "")

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
