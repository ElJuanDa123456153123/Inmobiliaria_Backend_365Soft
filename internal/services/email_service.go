package services

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/gomail.v2"
)

// EmailService - Servicio para envío de correos electrónicos
type EmailService struct {
	host      string
	port      int
	user      string
	password  string
	fromName  string
	fromEmail string
}

// NewEmailService - Crea una nueva instancia del servicio de email
func NewEmailService() *EmailService {
	port := 587
	if p := os.Getenv("SMTP_PORT"); p != "" {
		if _, err := fmt.Sscanf(p, "%d", &port); err != nil {
			port = 587
		}
	}

	return &EmailService{
		host:      getEnvOrDefault("SMTP_HOST", "smtp.gmail.com"),
		port:      port,
		user:      os.Getenv("SMTP_USER"),
		password:  os.Getenv("SMTP_PASSWORD"),
		fromName:  getEnvOrDefault("SMTP_FROM_NAME", "Inmobiliaria 365Soft"),
		fromEmail: getEnvOrDefault("SMTP_FROM_EMAIL", os.Getenv("SMTP_USER")),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// IsConfigured - Verifica si el servicio de email está configurado
func (s *EmailService) IsConfigured() bool {
	return s.user != "" && s.password != ""
}

// SendEmail - Envía un correo electrónico
func (s *EmailService) SendEmail(to string, subject string, body string) error {
	if !s.IsConfigured() {
		log.Println("⚠️  Email: SMTP no configurado, saltando envío")
		return nil
	}

	m := gomail.NewMessage()
	m.SetAddressHeader("From", s.fromEmail, s.fromName)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(s.host, s.port, s.user, s.password)

	if err := d.DialAndSend(m); err != nil {
		log.Printf("❌ Email: Error enviando a %s: %v", to, err)
		return err
	}

	log.Printf("✅ Email: Enviado a %s - Asunto: %s", to, subject)
	return nil
}

// emailWrapper - Plantilla base profesional para todos los emails
func emailWrapper(headerColor, headerIcon, headerTitle, content string) string {
	frontendURL := getEnvOrDefault("FRONTEND_URL", "http://localhost:3000")
	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="es">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin:0;padding:0;background-color:#f3f4f6;font-family:'Segoe UI',Tahoma,Geneva,Verdana,sans-serif;">
	<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f3f4f6;padding:24px 0;">
		<tr>
			<td align="center">
				<table width="560" cellpadding="0" cellspacing="0" style="background-color:#ffffff;border-radius:16px;overflow:hidden;box-shadow:0 4px 6px -1px rgba(0,0,0,0.1),0 2px 4px -1px rgba(0,0,0,0.06);">

					<!-- Header con gradiente -->
					<tr>
						<td style="background:linear-gradient(135deg,%s 0%%,%s 100%%);padding:32px 40px;text-align:center;">
							<div style="font-size:42px;margin-bottom:8px;">%s</div>
							<h1 style="color:#ffffff;font-size:24px;font-weight:700;margin:0;letter-spacing:-0.5px;">%s</h1>
						</td>
					</tr>

					<!-- Contenido -->
					<tr>
						<td style="padding:36px 40px;">
							%s
						</td>
					</tr>

					<!-- Footer -->
					<tr>
						<td style="background-color:#f9fafb;padding:24px 40px;border-top:1px solid #e5e7eb;">
							<table width="100%%" cellpadding="0" cellspacing="0">
								<tr>
									<td align="center">
										<p style="margin:0 0 8px 0;font-size:16px;font-weight:700;color:#1f2937;">🏠 Inmobiliaria 365Soft</p>
										<p style="margin:0;font-size:13px;color:#9ca3af;">
											Este es un correo automático, por favor no respondas a este mensaje.
										</p>
										<p style="margin:4px 0 0 0;font-size:13px;">
											<a href="%s" style="color:#6b7280;text-decoration:none;">Visitar sitio web</a>
										</p>
									</td>
								</tr>
							</table>
						</td>
					</tr>

				</table>
			</td>
		</tr>
	</table>
</body>
</html>
	`, headerColor, darkenColor(headerColor), headerIcon, headerTitle, content, frontendURL)
}

func darkenColor(hex string) string {
	if len(hex) == 7 && hex[0] == '#' {
		r := hex[1:3]
		g := hex[3:5]
		b := hex[5:7]
		return "#" + r + g + b
	}
	return hex
}

// infoCard - Genera una tarjeta de información con filas clave-valor
func infoCard(rows [][2]string) string {
	html := `<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f9fafb;border:1px solid #e5e7eb;border-radius:12px;margin:20px 0;">`
	for _, row := range rows {
		html += fmt.Sprintf(`
			<tr>
				<td style="padding:12px 20px;border-bottom:1px solid #f3f4f6;">
					<span style="font-size:15px;color:#6b7280;font-weight:500;">%s</span>
					<span style="font-size:15px;color:#1f2937;font-weight:600;float:right;">%s</span>
				</td>
			</tr>`, row[0], row[1])
	}
	html += `</table>`
	return html
}

// ctaButton - Genera un botón de llamada a la acción
func ctaButton(text, url, color string) string {
	return fmt.Sprintf(`
		<table width="100%%" cellpadding="0" cellspacing="0" style="margin-top:24px;">
			<tr>
				<td align="center">
					<a href="%s" style="display:inline-block;background:linear-gradient(135deg,%s 0%%,%s 100%%);color:#ffffff;padding:14px 40px;border-radius:10px;text-decoration:none;font-size:16px;font-weight:600;letter-spacing:0.3px;box-shadow:0 2px 4px rgba(0,0,0,0.1);">
						%s
					</a>
				</td>
			</tr>
		</table>`,
		url, color, darkenColor(color), text)
}

// SendVisitRequestEmail - Notifica al vendedor de una nueva solicitud de visita
func (s *EmailService) SendVisitRequestEmail(vendedorEmail, vendedorNombre string, propiedadTitulo, clienteNombre, clienteTelefono, horario string) error {
	subject := "Nueva solicitud de visita para tu propiedad"
	frontendURL := getEnvOrDefault("FRONTEND_URL", "http://localhost:3000")

	card := infoCard([][2]string{
		{"🏠 Propiedad", propiedadTitulo},
		{"👤 Cliente", clienteNombre},
		{"📞 Teléfono", clienteTelefono},
		{"🕐 Horario preferido", horario},
	})

	button := ctaButton("Ir a mi panel", frontendURL+"/panel/visitas?tab=recibidas", "#2563eb")

	content := fmt.Sprintf(`
		<p style="font-size:17px;color:#1f2937;margin:0 0 8px 0;">Hola <strong>%s</strong> 👋</p>
		<p style="font-size:16px;color:#4b5563;margin:0 0 8px 0;">Tienes una nueva solicitud de visita para tu propiedad.</p>
		%s
		<p style="font-size:15px;color:#6b7280;margin:16px 0 0 0;">
			Ingresa a tu panel para confirmar la fecha y hora de la visita.
		</p>
		%s
	`, vendedorNombre, card, button)

	body := emailWrapper("#2563eb", "📅", "Nueva Solicitud de Visita", content)
	return s.SendEmail(vendedorEmail, subject, body)
}

// SendVisitConfirmedEmail - Notifica al cliente que su visita fue confirmada
func (s *EmailService) SendVisitConfirmedEmail(clienteEmail, clienteNombre string, propiedadTitulo string, fechaVisita string, vendedorNombre string) error {
	subject := "Tu visita ha sido confirmada"
	frontendURL := getEnvOrDefault("FRONTEND_URL", "http://localhost:3000")

	card := infoCard([][2]string{
		{"🏠 Propiedad", propiedadTitulo},
		{"📅 Fecha y hora", fechaVisita},
		{"👤 Confirmado por", vendedorNombre},
	})

	button := ctaButton("Ver mis visitas", frontendURL+"/perfil?tab=visitas&subtab=mias", "#16a34a")

	content := fmt.Sprintf(`
		<p style="font-size:17px;color:#1f2937;margin:0 0 8px 0;">Hola <strong>%s</strong> 👋</p>
		<p style="font-size:16px;color:#4b5563;margin:0 0 8px 0;">¡Tu solicitud de visita ha sido confirmada!</p>
		%s
		<p style="font-size:15px;color:#6b7280;margin:16px 0 0 0;">
			Te esperamos en la fecha y hora indicadas. Si necesitas reprogramar, puedes hacerlo desde tu panel.
		</p>
		%s
	`, clienteNombre, card, button)

	body := emailWrapper("#16a34a", "✅", "Visita Confirmada", content)
	return s.SendEmail(clienteEmail, subject, body)
}

// SendVisitCancelledEmail - Notifica al cliente que su visita fue cancelada
func (s *EmailService) SendVisitCancelledEmail(clienteEmail, clienteNombre string, propiedadTitulo string) error {
	subject := "Actualización sobre tu visita"
	frontendURL := getEnvOrDefault("FRONTEND_URL", "http://localhost:3000")

	card := infoCard([][2]string{
		{"🏠 Propiedad", propiedadTitulo},
		{"📊 Estado", "Cancelada"},
	})

	button := ctaButton("Ver propiedades", frontendURL+"/propiedades", "#2563eb")

	content := fmt.Sprintf(`
		<p style="font-size:17px;color:#1f2937;margin:0 0 8px 0;">Hola <strong>%s</strong>,</p>
		<p style="font-size:16px;color:#4b5563;margin:0 0 8px 0;">
			Lamentamos informarte que la visita para la propiedad <strong>%s</strong> ha sido cancelada por el vendedor.
		</p>
		%s
		<p style="font-size:15px;color:#6b7280;margin:16px 0 0 0;">
			Puedes buscar otras propiedades o contactar directamente al vendedor para reprogramar.
		</p>
		%s
	`, clienteNombre, propiedadTitulo, card, button)

	body := emailWrapper("#dc2626", "❌", "Visita Cancelada", content)
	return s.SendEmail(clienteEmail, subject, body)
}
