package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
)

// ============================================
// ESTRUCTURAS PARA API DEL BANCO MC4
// ============================================

// BancoAuthRequest - Request para autenticación
type BancoAuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// BancoAuthResponse - Response de autenticación
type BancoAuthResponse struct {
	Codigo  string `json:"codigo"`
	Mensaje string `json:"mensaje"`
	Objeto  struct {
		Token     string `json:"token"`
		TokenType string `json:"tokenType"`
	} `json:"objeto"`
}

// BancoGenerarQrRequest - Request para generar QR
type BancoGenerarQrRequest struct {
	Alias            string  `json:"alias"`
	Monto            float64 `json:"monto"`
	Moneda           string  `json:"moneda"`
	DetalleGlosa     string  `json:"detalleGlosa"`
	FechaVencimiento string  `json:"fechaVencimiento"`
	TipoSolicitud    string  `json:"tipoSolicitud"`
	UnicoUso         bool    `json:"unicoUso"`
	Callback         string  `json:"callback"`
}

// BancoGenerarQrResponse - Response de generación de QR
type BancoGenerarQrResponse struct {
	Codigo  string `json:"codigo"`
	Mensaje string `json:"mensaje"`
	Objeto  struct {
		Alias    string `json:"alias"`
		ImagenQr string `json:"imagenQr"`
	} `json:"objeto"`
}

// BancoVerificarEstadoRequest - Request para verificar estado
type BancoVerificarEstadoRequest struct {
	Alias string `json:"alias"`
}

// BancoVerificarEstadoResponse - Response de verificación de estado
type BancoVerificarEstadoResponse struct {
	Codigo  string `json:"codigo"`
	Mensaje string `json:"mensaje"`
	Objeto  struct {
		Alias       string `json:"alias"`
		EstadoActual string `json:"estadoActual"` // PENDIENTE, PAGADO, EXPIRADO
	} `json:"objeto"`
}

// BancoWebhookPayload - Payload del webhook del banco
type BancoWebhookPayload struct {
	Alias                string `json:"alias"`
	NumeroOrdenOriginante string `json:"numeroOrdenOriginante"`
	NombreCliente        string `json:"nombreCliente"`
	DocumentoCliente     string `json:"documentoCliente"`
	CuentaCliente        string `json:"cuentaCliente"`
	Fechaproceso         string `json:"fechaproceso"`
}

// ============================================
// SERVICIO DE INTEGRACIÓN CON BANCO MC4
// ============================================

type BancoQrService struct {
	cfg           *config.Config
	tokenCache    struct {
		token     string
		expiresAt time.Time
	}
	httpClient *HTTPClient
}

// NewBancoQrService crea una nueva instancia del servicio
func NewBancoQrService(cfg *config.Config) *BancoQrService {
	return &BancoQrService{
		cfg:        cfg,
		httpClient: NewHTTPClient(cfg.BancoQr.Timeout),
	}
}

// GenerarToken genera o reutiliza un token de autenticación con el banco
func (s *BancoQrService) GenerarToken(ctx context.Context) (string, error) {
	// Verificar si hay un token válido en cache
	if s.tokenCache.token != "" && s.tokenCache.expiresAt.After(time.Now()) {
		return s.tokenCache.token, nil
	}

	// Generar nuevo token
	url := fmt.Sprintf("%s/autenticacion/v1/generarToken", s.cfg.BancoQr.APIURL)

	headers := map[string]string{
		"apikey":        s.cfg.BancoQr.APIKey,
		"Content-Type":  "application/json",
	}

	body := BancoAuthRequest{
		Username: s.cfg.BancoQr.Username,
		Password: s.cfg.BancoQr.Password,
	}

	resp, err := s.httpClient.Post(ctx, url, headers, body)
	if err != nil {
		return "", fmt.Errorf("error llamando a autenticación: %w", err)
	}
	defer resp.Body.Close()

	var authResp BancoAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", fmt.Errorf("error decodificando respuesta: %w", err)
	}

	if authResp.Codigo != "0000" && authResp.Codigo != "OK" {
		return "", fmt.Errorf("error generando token: %s", authResp.Mensaje)
	}

	token := authResp.Objeto.Token

	// Cache del token (expira en 1 hora)
	s.tokenCache.token = token
	s.tokenCache.expiresAt = time.Now().Add(60 * time.Minute)

	return token, nil
}

// GenerarQrDinamico genera un QR dinámico para pago
func (s *BancoQrService) GenerarQrDinamico(ctx context.Context, params struct {
	Alias            string
	Monto            float64
	DetalleGlosa     string
	FechaVencimiento time.Time
}) (*BancoGenerarQrResponse, error) {
	token, err := s.GenerarToken(ctx)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v1/generaQr", s.cfg.BancoQr.APIURL)

	headers := map[string]string{
		"apikeyServicio": s.cfg.BancoQr.ServiceKey,
		"Content-Type":   "application/json",
		"Authorization":  fmt.Sprintf("Bearer %s", token),
	}

	body := BancoGenerarQrRequest{
		Alias:            params.Alias,
		Monto:            params.Monto,
		Moneda:           "BOB",
		DetalleGlosa:     params.DetalleGlosa,
		FechaVencimiento: s.formatearFecha(params.FechaVencimiento),
		TipoSolicitud:    "API",
		UnicoUso:         true,
		Callback:         s.cfg.BancoQr.CallbackURL,
	}

	resp, err := s.httpClient.Post(ctx, url, headers, body)
	if err != nil {
		return nil, fmt.Errorf("error generando QR: %w", err)
	}
	defer resp.Body.Close()

	var qrResp BancoGenerarQrResponse
	if err := json.NewDecoder(resp.Body).Decode(&qrResp); err != nil {
		return nil, fmt.Errorf("error decodificando respuesta: %w", err)
	}

	if qrResp.Codigo != "0000" && qrResp.Codigo != "OK" {
		return nil, fmt.Errorf("error generando QR: %s", qrResp.Mensaje)
	}

	return &qrResp, nil
}

// VerificarEstadoQr verifica el estado de un QR
func (s *BancoQrService) VerificarEstadoQr(ctx context.Context, alias string) (*BancoVerificarEstadoResponse, error) {
	token, err := s.GenerarToken(ctx)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v1/estadoTransaccion", s.cfg.BancoQr.APIURL)

	headers := map[string]string{
		"apikeyServicio": s.cfg.BancoQr.ServiceKey,
		"Content-Type":   "application/json",
		"Authorization":  fmt.Sprintf("Bearer %s", token),
	}

	body := BancoVerificarEstadoRequest{
		Alias: alias,
	}

	resp, err := s.httpClient.Post(ctx, url, headers, body)
	if err != nil {
		return nil, fmt.Errorf("error verificando estado: %w", err)
	}
	defer resp.Body.Close()

	var estadoResp BancoVerificarEstadoResponse
	if err := json.NewDecoder(resp.Body).Decode(&estadoResp); err != nil {
		return nil, fmt.Errorf("error decodificando respuesta: %w", err)
	}

	if estadoResp.Codigo != "0000" && estadoResp.Codigo != "OK" {
		return nil, fmt.Errorf("error verificando estado: %s", estadoResp.Mensaje)
	}

	return &estadoResp, nil
}

// InhabilitarQr inhabilita un QR para que no acepte más pagos
func (s *BancoQrService) InhabilitarQr(ctx context.Context, alias string) error {
	token, err := s.GenerarToken(ctx)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/inhabilitarPago", s.cfg.BancoQr.APIURL)

	headers := map[string]string{
		"apikeyServicio": s.cfg.BancoQr.ServiceKey,
		"Content-Type":   "application/json",
		"Authorization":  fmt.Sprintf("Bearer %s", token),
	}

	body := map[string]string{
		"alias": alias,
	}

	resp, err := s.httpClient.Post(ctx, url, headers, body)
	if err != nil {
		return fmt.Errorf("error inhabilitando QR: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("error decodificando respuesta: %w", err)
	}

	codigo := result["codigo"].(string)
	if codigo != "0000" {
		return fmt.Errorf("error inhabilitando QR: %v", result)
	}

	return nil
}

// GenerarAliasUnico genera un alias único para el QR
// Formato: QR365I{timestamp}{random}
// Ejemplo: QR365I2025022015304512345678
func (s *BancoQrService) GenerarAliasUnico(propertyID *uint) string {
	// Incluir nanosegundos para mayor unicidad
	timestamp := time.Now().Format("20060102150405") // YYYYMMDDHHMMSS
	nanos := time.Now().UnixNano() % 1000000000       // Hasta 9 dígitos extra

	// Si propertyID es nil, usar 0
	id := uint(0)
	if propertyID != nil {
		id = *propertyID
	}

	// Generar número aleatorio más grande (6 dígitos)
	random := rand.Intn(1000000)

	return fmt.Sprintf("QR365I%s%d%d%06d", timestamp, id, nanos, random)
}

// formatearFecha formatea una fecha al formato dd/mm/yyyy
func (s *BancoQrService) formatearFecha(fecha time.Time) string {
	return fmt.Sprintf("%02d/%02d/%04d", fecha.Day(), fecha.Month(), fecha.Year())
}

// CalcularFechaVencimiento calcula la fecha de vencimiento (en minutos)
func (s *BancoQrService) CalcularFechaVencimiento(minutos int) time.Time {
	return time.Now().Add(time.Duration(minutos) * time.Minute)
}

// ValidarRespuestaBanco valida si la respuesta del banco es exitosa
func (s *BancoQrService) ValidarRespuestaBanco(codigo string) bool {
	return codigo == "0000" || codigo == "OK"
}

// ============================================
// HTTP CLIENT (SIMPLE)
// ============================================

type HTTPClient struct {
	timeout time.Duration
}

func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		timeout: timeout,
	}
}

func (c *HTTPClient) Post(ctx context.Context, url string, headers map[string]string, body interface{}) (*Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{
		Timeout: c.timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return &Response{
		Body:       resp.Body,
		StatusCode: resp.StatusCode,
	}, nil
}

type Response struct {
	Body       io.ReadCloser
	StatusCode int
}

